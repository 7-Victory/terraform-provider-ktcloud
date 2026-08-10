package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultAPIBase 는 kt cloud Open API 공통 도메인입니다.
	DefaultAPIBase = "https://api.ucloudbiz.olleh.com"

	// DefaultDomainID 는 인증 요청 시 사용하는 기본 domain id 입니다.
	DefaultDomainID = "default"

	// 토큰 유효시간(60분)보다 이만큼 먼저 갱신합니다.
	tokenRefreshSkew = 5 * time.Minute
)

// Service 는 kt cloud 서비스별 API 엔드포인트 접미사입니다.
// (참고: API-END-POINT 는 https://api.ucloudbiz.olleh.com/{zone}/{service} 형태)
type Service string

const (
	ServiceIdentity Service = "identity"
	ServiceServer   Service = "server"        // Virtual Machine, SSH Keypair
	ServiceNetwork  Service = "nc"            // Networking, Tier
	ServiceVolume   Service = "volume"        // Volume, Snapshot
	ServiceImage    Service = "image"         // Image
	ServiceNAS      Service = "nas"           // NAS
	ServiceLB       Service = "loadbalancers" // Load Balancer
	ServiceGSLB     Service = "gslbs"         // GSLB
	ServiceWatch    Service = "watch"         // Watch
)

// Config 는 Client 생성에 필요한 설정입니다.
type Config struct {
	APIBase     string
	Zone        string // d1 / d2 / gd1 / gd4
	Username    string
	Password    string
	DomainID    string
	ProjectName string
	Insecure    bool
	Timeout     time.Duration

	// UseProjectIDPath 가 true 면 volume 이외 서비스의 요청 경로에도 project_id 를
	// 삽입합니다.
	//   false: /{zone}/server/servers
	//   true : /{zone}/server/{project_id}/servers
	// volume 서비스는 kt cloud 쪽에서 project_id 가 항상 필수라 이 옵션과 무관하게
	// 자동으로 삽입됩니다. 규격서와 맞지 않을 때만 true 로 바꾸세요.
	UseProjectIDPath bool
}

// Client 는 kt cloud Open API 호출용 클라이언트입니다. 토큰을 캐싱하고
// 만료 5분 전 또는 401 응답 시 자동으로 재발급합니다.
type Client struct {
	cfg  Config
	http *http.Client

	mu        sync.Mutex
	token     string
	expiresAt time.Time
	projectID string
}

// APIError 는 2xx 이외의 응답을 나타냅니다.
type APIError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("kt cloud API 오류: %s %s -> HTTP %d\n%s", e.Method, e.URL, e.StatusCode, e.Body)
}

// IsNotFound 는 404 여부를 반환합니다. (Read 단계에서 상태 제거 판단용)
// kt cloud 는 리소스가 없을 때 HTTP 상태코드는 400 인데 응답 본문에
// {"itemNotFound": {"code": 404, ...}} 를 담아 보내는 경우가 있어, 본문도
// 함께 확인합니다.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if ok := asAPIError(err, &apiErr); ok {
		if apiErr.StatusCode == http.StatusNotFound {
			return true
		}
		return strings.Contains(apiErr.Body, "itemNotFound")
	}
	return false
}

func asAPIError(err error, target **APIError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}

// New 는 설정을 검증하고 Client 를 생성합니다.
func New(cfg Config) (*Client, error) {
	if cfg.APIBase == "" {
		cfg.APIBase = DefaultAPIBase
	}
	cfg.APIBase = strings.TrimRight(cfg.APIBase, "/")

	if cfg.DomainID == "" {
		cfg.DomainID = DefaultDomainID
	}
	if cfg.ProjectName == "" {
		// 문서 기준 project.name 기본값은 사용자 ID 입니다.
		cfg.ProjectName = cfg.Username
	}
	if cfg.Zone == "" {
		return nil, fmt.Errorf("zone 이 비어 있습니다 (d1, d2, gd1, gd4 중 하나)")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username 이 비어 있습니다")
	}
	if cfg.Password == "" {
		return nil, fmt.Errorf("password 가 비어 있습니다")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec G402 - 사용자가 명시적으로 선택
	}

	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout, Transport: transport},
	}, nil
}

// Zone 은 설정된 zone 약어를 반환합니다.
func (c *Client) Zone() string { return c.cfg.Zone }

// ProjectID 는 인증 응답에서 받은 project id 를 반환합니다.
func (c *Client) ProjectID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.projectID
}

// ---------------------------------------------------------------------------
// 인증 (Step 1. 인증 토큰 발급)
// ---------------------------------------------------------------------------

type authDomain struct {
	ID string `json:"id"`
}

type authUser struct {
	Domain   authDomain `json:"domain"`
	Name     string     `json:"name"`
	Password string     `json:"password"`
}

type authPassword struct {
	User authUser `json:"user"`
}

type authIdentity struct {
	Methods  []string     `json:"methods"`
	Password authPassword `json:"password"`
}

type authProject struct {
	Domain authDomain `json:"domain"`
	Name   string     `json:"name"`
}

type authScope struct {
	Project authProject `json:"project"`
}

type authBody struct {
	Identity authIdentity `json:"identity"`
	Scope    authScope    `json:"scope"`
}

type authRequest struct {
	Auth authBody `json:"auth"`
}

type tokenResponse struct {
	Token struct {
		ExpiresAt time.Time `json:"expires_at"`
		Project   struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"project"`
	} `json:"token"`
}

// Authenticate 는 인증 토큰을 발급받아 캐시에 저장합니다.
func (c *Client) Authenticate(ctx context.Context) error {
	body := authRequest{
		Auth: authBody{
			Identity: authIdentity{
				Methods: []string{"password"},
				Password: authPassword{
					User: authUser{
						Domain:   authDomain{ID: c.cfg.DomainID},
						Name:     c.cfg.Username,
						Password: c.cfg.Password,
					},
				},
			},
			Scope: authScope{
				Project: authProject{
					Domain: authDomain{ID: c.cfg.DomainID},
					Name:   c.cfg.ProjectName,
				},
			},
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("인증 요청 본문 직렬화 실패: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/auth/tokens", c.cfg.APIBase, c.cfg.Zone, ServiceIdentity)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("인증 요청 생성 실패: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("인증 요청 실패: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return &APIError{Method: http.MethodPost, URL: url, StatusCode: resp.StatusCode, Body: string(raw)}
	}

	token := resp.Header.Get("X-Subject-Token")
	if token == "" {
		return fmt.Errorf("응답 헤더에 X-Subject-Token 이 없습니다 (HTTP %d)", resp.StatusCode)
	}

	var parsed tokenResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("인증 응답 파싱 실패: %w", err)
	}

	expires := parsed.Token.ExpiresAt
	if expires.IsZero() {
		// expires_at 이 없으면 문서상 최대 유효시간(60분) 기준으로 보수적으로 설정
		expires = time.Now().Add(55 * time.Minute)
	}

	c.mu.Lock()
	c.token = token
	c.expiresAt = expires
	c.projectID = parsed.Token.Project.ID
	c.mu.Unlock()

	return nil
}

// token 이 유효하면 그대로, 아니면 재발급 후 반환합니다.
func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	tok := c.token
	exp := c.expiresAt
	c.mu.Unlock()

	if tok != "" && time.Now().Add(tokenRefreshSkew).Before(exp) {
		return tok, nil
	}

	if err := c.Authenticate(ctx); err != nil {
		return "", err
	}

	c.mu.Lock()
	tok = c.token
	c.mu.Unlock()

	return tok, nil
}

// invalidateToken 은 캐시된 토큰을 폐기합니다. (401 수신 시)
func (c *Client) invalidateToken() {
	c.mu.Lock()
	c.token = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}

// ---------------------------------------------------------------------------
// 공통 요청 (Step 2. Open API 호출 / Step 3. 응답·에러 처리)
// ---------------------------------------------------------------------------

// URLFor 는 서비스와 하위 경로로 최종 URL 을 만듭니다.
func (c *Client) URLFor(svc Service, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if c.needsProjectIDPath(svc) {
		if pid := c.ProjectID(); pid != "" {
			return fmt.Sprintf("%s/%s/%s/%s%s", c.cfg.APIBase, c.cfg.Zone, svc, pid, path)
		}
	}
	return fmt.Sprintf("%s/%s/%s%s", c.cfg.APIBase, c.cfg.Zone, svc, path)
}

// needsProjectIDPath 는 서비스별로 경로에 project_id 를 넣어야 하는지 판단합니다.
// 실 계정으로 확인한 결과 volume(cinder) 은 project_id 가 없으면 500 이 나고,
// server/image 는 반대로 project_id 를 붙이면 500 이 난다. 서비스마다 동작이
// 다르므로 전역 옵션 하나로는 표현할 수 없어, volume 은 항상 강제하고 나머지는
// cfg.UseProjectIDPath 로 수동 오버라이드할 수 있게 둔다.
func (c *Client) needsProjectIDPath(svc Service) bool {
	if svc == ServiceVolume {
		return true
	}
	return c.cfg.UseProjectIDPath
}

// Do 는 인증 헤더를 붙여 요청하고 JSON 응답을 out 에 언마샬합니다.
// reqBody 가 nil 이면 본문 없이 호출하고, out 이 nil 이면 응답을 버립니다.
func (c *Client) Do(ctx context.Context, method string, svc Service, path string, reqBody any, out any) error {
	var payload []byte
	if reqBody != nil {
		var err error
		payload, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("요청 본문 직렬화 실패: %w", err)
		}
	}

	// 1차 시도
	status, raw, url, err := c.roundTrip(ctx, method, svc, path, payload)
	if err != nil {
		return err
	}

	// 토큰이 만료되었거나 무효화된 경우 1회 재시도
	if status == http.StatusUnauthorized {
		c.invalidateToken()
		status, raw, url, err = c.roundTrip(ctx, method, svc, path, payload)
		if err != nil {
			return err
		}
	}

	if status < 200 || status > 299 {
		return &APIError{Method: method, URL: url, StatusCode: status, Body: string(raw)}
	}

	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("응답 파싱 실패 (%s %s): %w\n원본: %s", method, url, err, string(raw))
	}

	return nil
}

func (c *Client) roundTrip(ctx context.Context, method string, svc Service, path string, payload []byte) (int, []byte, string, error) {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return 0, nil, "", err
	}

	url := c.URLFor(svc, path)

	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return 0, nil, url, fmt.Errorf("요청 생성 실패: %w", err)
	}

	req.Header.Set("X-Auth-Token", token)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, url, fmt.Errorf("요청 실패 (%s %s): %w", method, url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, url, fmt.Errorf("응답 본문 읽기 실패: %w", err)
	}

	return resp.StatusCode, raw, url, nil
}
