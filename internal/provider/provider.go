package provider

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/client"
)

// 인터페이스 구현 확인 (컴파일 타임 체크)
var _ provider.Provider = (*ktcloudProvider)(nil)

type ktcloudProvider struct {
	version string
}

// New 는 plugin server 에 넘길 provider 팩토리를 반환합니다.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ktcloudProvider{version: version}
	}
}

type providerModel struct {
	APIBase          types.String `tfsdk:"api_base"`
	Zone             types.String `tfsdk:"zone"`
	Username         types.String `tfsdk:"username"`
	Password         types.String `tfsdk:"password"`
	DomainID         types.String `tfsdk:"domain_id"`
	ProjectName      types.String `tfsdk:"project_name"`
	Insecure         types.Bool   `tfsdk:"insecure"`
	UseProjectIDPath types.Bool   `tfsdk:"use_project_id_path"`
	RequestTimeout   types.Int64  `tfsdk:"request_timeout"`
}

func (p *ktcloudProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ktcloud"
	resp.Version = p.version
}

func (p *ktcloudProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "kt cloud @D Platform Open API 용 Terraform Provider 입니다.",
		Attributes: map[string]schema.Attribute{
			"api_base": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "API 기본 도메인. 기본값 `https://api.ucloudbiz.olleh.com`. 환경변수 `KTCLOUD_API_BASE`.",
			},
			"zone": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Zone 약어. `d1`(DX-M1), `d2`(DX-Central), `gd1`(DX-G), `gd4`(DX-G-YS). 환경변수 `KTCLOUD_ZONE`.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "kt cloud 포털 사용자 ID. 환경변수 `KTCLOUD_USERNAME`.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "kt cloud 포털 비밀번호. 환경변수 `KTCLOUD_PASSWORD`.",
			},
			"domain_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "인증 domain id. 기본값 `default`. 환경변수 `KTCLOUD_DOMAIN_ID`.",
			},
			"project_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "인증 scope 의 project name. 미지정 시 `username` 과 동일하게 사용합니다. 환경변수 `KTCLOUD_PROJECT_NAME`.",
			},
			"insecure": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "TLS 인증서 검증을 건너뜁니다. 테스트 용도로만 사용하세요. 환경변수 `KTCLOUD_INSECURE`.",
			},
			"use_project_id_path": schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: "true 로 설정하면 요청 경로에 project_id 를 넣습니다 " +
					"(`/{zone}/server/{project_id}/servers`). 규격서와 경로가 다를 때만 사용하세요. 환경변수 `KTCLOUD_USE_PROJECT_ID_PATH`.",
			},
			"request_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "개별 HTTP 요청 타임아웃(초). 기본값 60.",
			},
		},
	}
}

func (p *ktcloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// HCL 값이 없으면 환경변수로 대체
	apiBase := firstNonEmpty(stringValue(cfg.APIBase), os.Getenv("KTCLOUD_API_BASE"), client.DefaultAPIBase)
	zone := firstNonEmpty(stringValue(cfg.Zone), os.Getenv("KTCLOUD_ZONE"))
	username := firstNonEmpty(stringValue(cfg.Username), os.Getenv("KTCLOUD_USERNAME"))
	password := firstNonEmpty(stringValue(cfg.Password), os.Getenv("KTCLOUD_PASSWORD"))
	domainID := firstNonEmpty(stringValue(cfg.DomainID), os.Getenv("KTCLOUD_DOMAIN_ID"), client.DefaultDomainID)
	projectName := firstNonEmpty(stringValue(cfg.ProjectName), os.Getenv("KTCLOUD_PROJECT_NAME"))

	insecure := boolValue(cfg.Insecure) || envBool("KTCLOUD_INSECURE")
	useProjectPath := boolValue(cfg.UseProjectIDPath) || envBool("KTCLOUD_USE_PROJECT_ID_PATH")

	timeout := 60 * time.Second
	if !cfg.RequestTimeout.IsNull() && cfg.RequestTimeout.ValueInt64() > 0 {
		timeout = time.Duration(cfg.RequestTimeout.ValueInt64()) * time.Second
	}

	if zone == "" {
		resp.Diagnostics.AddError(
			"zone 설정 누락",
			"provider 블록의 `zone` 또는 환경변수 `KTCLOUD_ZONE` 을 설정하세요. (예: d1)",
		)
	}
	if username == "" {
		resp.Diagnostics.AddError(
			"username 설정 누락",
			"provider 블록의 `username` 또는 환경변수 `KTCLOUD_USERNAME` 을 설정하세요.",
		)
	}
	if password == "" {
		resp.Diagnostics.AddError(
			"password 설정 누락",
			"provider 블록의 `password` 또는 환경변수 `KTCLOUD_PASSWORD` 를 설정하세요.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := client.New(client.Config{
		APIBase:          apiBase,
		Zone:             zone,
		Username:         username,
		Password:         password,
		DomainID:         domainID,
		ProjectName:      projectName,
		Insecure:         insecure,
		Timeout:          timeout,
		UseProjectIDPath: useProjectPath,
	})
	if err != nil {
		resp.Diagnostics.AddError("kt cloud 클라이언트 생성 실패", err.Error())
		return
	}

	// 잘못된 자격증명을 plan 단계에서 바로 잡기 위해 미리 인증합니다.
	if err := c.Authenticate(ctx); err != nil {
		resp.Diagnostics.AddError(
			"kt cloud 인증 실패",
			"인증 토큰 발급에 실패했습니다. username/password/zone 을 확인하세요.\n\n"+err.Error(),
		)
		return
	}

	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *ktcloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewServerResource,
		NewKeypairResource,
		NewVolumeResource,
		NewVolumeAttachmentResource,
	}
}

func (p *ktcloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFlavorsDataSource,
		NewImagesDataSource,
	}
}

// --- 작은 헬퍼들 -----------------------------------------------------------

func stringValue(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func boolValue(v types.Bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return false
	}
	return v.ValueBool()
}

func envBool(key string) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return false
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
