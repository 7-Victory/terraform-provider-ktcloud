package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// IDRef 는 flavor/image 처럼 id 만 담긴 참조 객체입니다.
type IDRef struct {
	ID string `json:"id"`
}

// UnmarshalJSON 은 {"id": "..."} 형태뿐 아니라 kt cloud 가 볼륨 부팅한 VM 에 대해
// image 필드로 돌려주는 빈 문자열("") 도 허용합니다.
func (r *IDRef) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.ID = s
		return nil
	}
	var obj struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	r.ID = obj.ID
	return nil
}

// Address 는 서버에 붙은 IP 정보입니다.
type Address struct {
	Addr    string `json:"addr"`
	Version int    `json:"version"`
	Type    string `json:"OS-EXT-IPS:type"` // fixed | floating
	MacAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
}

// Server 는 VM 한 대의 정보입니다.
type Server struct {
	ID        string               `json:"id"`
	Name      string               `json:"name"`
	Status    string               `json:"status"`
	Created   string               `json:"created"`
	Updated   string               `json:"updated"`
	KeyName   string               `json:"key_name"`
	Flavor    IDRef                `json:"flavor"`
	Image     IDRef                `json:"image"`
	Addresses map[string][]Address `json:"addresses"`
	Metadata  map[string]string    `json:"metadata"`
	Fault     *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"fault"`
}

// FixedIP 는 첫 번째 사설 IP 를 반환합니다.
func (s *Server) FixedIP() string {
	return s.firstIPOfType("fixed")
}

// FloatingIP 는 첫 번째 공인 IP 를 반환합니다.
func (s *Server) FloatingIP() string {
	return s.firstIPOfType("floating")
}

func (s *Server) firstIPOfType(kind string) string {
	for _, addrs := range s.Addresses {
		for _, a := range addrs {
			if strings.EqualFold(a.Type, kind) {
				return a.Addr
			}
		}
	}
	// 타입 정보가 없는 응답 대비: fixed 요청이면 첫 IPv4 를 반환
	if kind == "fixed" {
		for _, addrs := range s.Addresses {
			for _, a := range addrs {
				if a.Version == 4 || a.Version == 0 {
					return a.Addr
				}
			}
		}
	}
	return ""
}

// NetworkOpts 는 VM 생성 시 연결할 네트워크(Tier)입니다.
type NetworkOpts struct {
	UUID    string `json:"uuid,omitempty"`
	Port    string `json:"port,omitempty"`
	FixedIP string `json:"fixed_ip,omitempty"`
}

// BlockDevice 는 루트/데이터 볼륨 설정입니다.
// BootIndex 는 kt cloud Nova 게이트웨이가 문자열 타입을 요구하기 때문에
// string 입니다. 숫자로 보내면 500 Internal server error 가 납니다.
type BlockDevice struct {
	BootIndex           string `json:"boot_index"`
	UUID                string `json:"uuid,omitempty"`
	SourceType          string `json:"source_type"`      // image | volume | snapshot | blank
	DestinationType     string `json:"destination_type"` // volume | local
	VolumeSize          int64  `json:"volume_size,omitempty"`
	DeleteOnTermination bool   `json:"delete_on_termination"`
	VolumeType          string `json:"volume_type,omitempty"`
}

// CreateServerOpts 는 VM 생성 요청 파라미터입니다.
type CreateServerOpts struct {
	Name             string            `json:"name"`
	FlavorRef        string            `json:"flavorRef"`
	ImageRef         string            `json:"imageRef,omitempty"`
	KeyName          string            `json:"key_name,omitempty"`
	AvailabilityZone string            `json:"availability_zone,omitempty"`
	UserData         string            `json:"user_data,omitempty"` // base64 인코딩된 값
	Networks         []NetworkOpts     `json:"networks,omitempty"`
	BlockDevices     []BlockDevice     `json:"block_device_mapping_v2,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	AdminPass        string            `json:"adminPass,omitempty"`
}

type createServerRequest struct {
	Server CreateServerOpts `json:"server"`
}

type serverWrapper struct {
	Server Server `json:"server"`
}

type serversWrapper struct {
	Servers []Server `json:"servers"`
}

// CreateServer 는 VM 을 생성하고 생성 직후 응답을 반환합니다.
func (c *Client) CreateServer(ctx context.Context, opts CreateServerOpts) (*Server, error) {
	var out serverWrapper
	if err := c.Do(ctx, http.MethodPost, ServiceServer, PathServers, createServerRequest{Server: opts}, &out); err != nil {
		return nil, err
	}
	if out.Server.ID == "" {
		return nil, fmt.Errorf("VM 생성 응답에 server.id 가 없습니다")
	}
	return &out.Server, nil
}

// GetServer 는 VM 상세 정보를 조회합니다.
func (c *Client) GetServer(ctx context.Context, id string) (*Server, error) {
	var out serverWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceServer, fmt.Sprintf(PathServerFmt, id), nil, &out); err != nil {
		return nil, err
	}
	return &out.Server, nil
}

// ListServers 는 VM 목록을 조회합니다.
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	var out serversWrapper
	if err := c.Do(ctx, http.MethodGet, ServiceServer, PathServerDetail, nil, &out); err != nil {
		return nil, err
	}
	return out.Servers, nil
}

// RenameServer 는 VM 이름을 변경합니다.
func (c *Client) RenameServer(ctx context.Context, id, name string) error {
	body := map[string]any{"server": map[string]string{"name": name}}
	return c.Do(ctx, http.MethodPut, ServiceServer, fmt.Sprintf(PathServerFmt, id), body, nil)
}

// DeleteServer 는 VM 을 삭제합니다.
func (c *Client) DeleteServer(ctx context.Context, id string) error {
	err := c.Do(ctx, http.MethodDelete, ServiceServer, fmt.Sprintf(PathServerFmt, id), nil, nil)
	if err != nil && IsNotFound(err) {
		return nil // 이미 없으면 성공으로 처리
	}
	return err
}

// ServerAction 은 /servers/{id}/action 계열 요청을 보냅니다.
func (c *Client) ServerAction(ctx context.Context, id string, body any) error {
	return c.Do(ctx, http.MethodPost, ServiceServer, fmt.Sprintf(PathServerActFmt, id), body, nil)
}

// WaitForServerStatus 는 원하는 상태가 될 때까지 폴링합니다.
func (c *Client) WaitForServerStatus(ctx context.Context, id, target string, timeout time.Duration) (*Server, error) {
	const interval = 5 * time.Second
	deadline := time.Now().Add(timeout)

	var last string
	for {
		srv, err := c.GetServer(ctx, id)
		if err != nil {
			return nil, err
		}
		last = strings.ToUpper(srv.Status)

		if last == strings.ToUpper(target) {
			return srv, nil
		}
		if last == "ERROR" {
			msg := "원인 정보 없음"
			if srv.Fault != nil && srv.Fault.Message != "" {
				msg = srv.Fault.Message
			}
			return srv, fmt.Errorf("VM(%s) 이 ERROR 상태입니다: %s", id, msg)
		}
		if time.Now().After(deadline) {
			return srv, fmt.Errorf("VM(%s) 이 %s 상태가 되기를 %s 동안 기다렸으나 현재 상태는 %s 입니다",
				id, target, timeout, last)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// WaitForServerDeleted 는 VM 이 완전히 사라질 때까지 기다립니다.
func (c *Client) WaitForServerDeleted(ctx context.Context, id string, timeout time.Duration) error {
	const interval = 5 * time.Second
	deadline := time.Now().Add(timeout)

	for {
		_, err := c.GetServer(ctx, id)
		if err != nil {
			if IsNotFound(err) {
				return nil
			}
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("VM(%s) 삭제가 %s 안에 완료되지 않았습니다", id, timeout)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}
