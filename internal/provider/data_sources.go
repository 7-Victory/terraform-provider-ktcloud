package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/client"
)

// ---------------------------------------------------------------------------
// data.ktcloud_flavors
// ---------------------------------------------------------------------------

var (
	_ datasource.DataSource              = (*flavorsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*flavorsDataSource)(nil)
)

// NewFlavorsDataSource 는 data.ktcloud_flavors 를 생성합니다.
func NewFlavorsDataSource() datasource.DataSource { return &flavorsDataSource{} }

type flavorsDataSource struct {
	client *client.Client
}

type flavorModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	VCPUs types.Int64  `tfsdk:"vcpus"`
	RAM   types.Int64  `tfsdk:"ram"`
	Disk  types.Int64  `tfsdk:"disk"`
}

type flavorsDataSourceModel struct {
	NameContains types.String  `tfsdk:"name_contains"`
	Flavors      []flavorModel `tfsdk:"flavors"`
}

func (d *flavorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flavors"
}

func (d *flavorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Provider 데이터 타입 오류",
			fmt.Sprintf("*client.Client 를 기대했으나 %T 를 받았습니다.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *flavorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "사용 가능한 VM 스펙(flavor) 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "이름에 이 문자열이 포함된 스펙만 반환합니다 (대소문자 무시).",
			},
			"flavors": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "조회된 스펙 목록.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.StringAttribute{Computed: true, MarkdownDescription: "스펙 ID."},
						"name":  schema.StringAttribute{Computed: true, MarkdownDescription: "스펙 이름."},
						"vcpus": schema.Int64Attribute{Computed: true, MarkdownDescription: "vCPU 수."},
						"ram":   schema.Int64Attribute{Computed: true, MarkdownDescription: "메모리(MB)."},
						"disk":  schema.Int64Attribute{Computed: true, MarkdownDescription: "디스크(GB)."},
					},
				},
			},
		},
	}
}

func (d *flavorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg flavorsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flavors, err := d.client.ListFlavors(ctx)
	if err != nil {
		resp.Diagnostics.AddError("flavor 목록 조회 실패", err.Error())
		return
	}

	filter := strings.ToLower(stringValue(cfg.NameContains))

	cfg.Flavors = nil
	for _, f := range flavors {
		if filter != "" && !strings.Contains(strings.ToLower(f.Name), filter) {
			continue
		}
		cfg.Flavors = append(cfg.Flavors, flavorModel{
			ID:    types.StringValue(f.ID),
			Name:  types.StringValue(f.Name),
			VCPUs: types.Int64Value(f.VCPUs),
			RAM:   types.Int64Value(f.RAM),
			Disk:  types.Int64Value(f.Disk),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

// ---------------------------------------------------------------------------
// data.ktcloud_images
// ---------------------------------------------------------------------------

var (
	_ datasource.DataSource              = (*imagesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*imagesDataSource)(nil)
)

// NewImagesDataSource 는 data.ktcloud_images 를 생성합니다.
func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

type imagesDataSource struct {
	client *client.Client
}

type imageModel struct {
	ID      types.String `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Status  types.String `tfsdk:"status"`
	MinDisk types.Int64  `tfsdk:"min_disk"`
	MinRAM  types.Int64  `tfsdk:"min_ram"`
}

type imagesDataSourceModel struct {
	NameContains types.String `tfsdk:"name_contains"`
	Images       []imageModel `tfsdk:"images"`
}

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_images"
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Provider 데이터 타입 오류",
			fmt.Sprintf("*client.Client 를 기대했으나 %T 를 받았습니다.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "사용 가능한 OS 이미지 목록을 조회합니다.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "이름에 이 문자열이 포함된 이미지만 반환합니다 (대소문자 무시).",
			},
			"images": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "조회된 이미지 목록.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true, MarkdownDescription: "이미지 ID."},
						"name":     schema.StringAttribute{Computed: true, MarkdownDescription: "이미지 이름."},
						"status":   schema.StringAttribute{Computed: true, MarkdownDescription: "이미지 상태."},
						"min_disk": schema.Int64Attribute{Computed: true, MarkdownDescription: "최소 디스크(GB)."},
						"min_ram":  schema.Int64Attribute{Computed: true, MarkdownDescription: "최소 메모리(MB)."},
					},
				},
			},
		},
	}
}

func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg imagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, err := d.client.ListImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("이미지 목록 조회 실패", err.Error())
		return
	}

	filter := strings.ToLower(stringValue(cfg.NameContains))

	cfg.Images = nil
	for _, img := range images {
		if filter != "" && !strings.Contains(strings.ToLower(img.Name), filter) {
			continue
		}
		cfg.Images = append(cfg.Images, imageModel{
			ID:      types.StringValue(img.ID),
			Name:    types.StringValue(img.Name),
			Status:  types.StringValue(img.Status),
			MinDisk: types.Int64Value(img.MinDisk),
			MinRAM:  types.Int64Value(img.MinRAM),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}

// ---------------------------------------------------------------------------
// data.ktcloud_networks
// ---------------------------------------------------------------------------

var (
	_ datasource.DataSource              = (*networksDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*networksDataSource)(nil)
)

// NewNetworksDataSource 는 data.ktcloud_networks 를 생성합니다.
func NewNetworksDataSource() datasource.DataSource { return &networksDataSource{} }

type networksDataSource struct {
	client *client.Client
}

type networkModel struct {
	ID        types.String `tfsdk:"id"`
	NetworkID types.String `tfsdk:"network_id"`
	Name      types.String `tfsdk:"name"`
	CIDR      types.String `tfsdk:"cidr"`
	GatewayIP types.String `tfsdk:"gateway_ip"`
	Type      types.String `tfsdk:"type"`
	Status    types.String `tfsdk:"status"`
	Shared    types.Bool   `tfsdk:"shared"`
}

type networksDataSourceModel struct {
	NameContains types.String   `tfsdk:"name_contains"`
	Networks     []networkModel `tfsdk:"networks"`
}

func (d *networksDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_networks"
}

func (d *networksDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Provider 데이터 타입 오류",
			fmt.Sprintf("*client.Client 를 기대했으나 %T 를 받았습니다.", req.ProviderData))
		return
	}
	d.client = c
}

func (d *networksDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "사용 가능한 네트워크(Tier) 목록을 조회합니다. 외부 업링크 등 " +
			"일부 항목은 `id`(refId)가 비어 있을 수 있으니, VM에 연결할 땐 `id`가 " +
			"비어있지 않은 항목만 쓰세요.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "이름에 이 문자열이 포함된 네트워크만 반환합니다 (대소문자 무시).",
			},
			"networks": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "조회된 네트워크 목록.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
							MarkdownDescription: "`ktcloud_server.networks.uuid` 에 써야 하는 " +
								"값(refId). `network_id` 와 혼동하지 마세요.",
						},
						"network_id": schema.StringAttribute{
							Computed: true,
							MarkdownDescription: "kt cloud 자체 네트워크 관리 ID. " +
								"**`networks.uuid` 에 이 값을 쓰면 `Network ... could not be " +
								"found` 에러가 납니다** — 콘솔 등에서 대조용으로만 참고하세요.",
						},
						"name":       schema.StringAttribute{Computed: true, MarkdownDescription: "네트워크(Tier) 이름."},
						"cidr":       schema.StringAttribute{Computed: true, MarkdownDescription: "CIDR 대역."},
						"gateway_ip": schema.StringAttribute{Computed: true, MarkdownDescription: "게이트웨이 IP."},
						"type":       schema.StringAttribute{Computed: true, MarkdownDescription: "네트워크 타입."},
						"status":     schema.StringAttribute{Computed: true, MarkdownDescription: "상태 (ACTIVE 등)."},
						"shared":     schema.BoolAttribute{Computed: true, MarkdownDescription: "공유 네트워크 여부."},
					},
				},
			},
		},
	}
}

func (d *networksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg networksDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	networks, err := d.client.ListNetworks(ctx)
	if err != nil {
		resp.Diagnostics.AddError("네트워크 목록 조회 실패", err.Error())
		return
	}

	filter := strings.ToLower(stringValue(cfg.NameContains))

	cfg.Networks = nil
	for _, n := range networks {
		if filter != "" && !strings.Contains(strings.ToLower(n.NetworkName), filter) {
			continue
		}
		cfg.Networks = append(cfg.Networks, networkModel{
			ID:        types.StringValue(n.RefID),
			NetworkID: types.StringValue(n.NetworkID),
			Name:      types.StringValue(n.NetworkName),
			CIDR:      types.StringValue(n.CIDR),
			GatewayIP: types.StringValue(n.GatewayIP),
			Type:      types.StringValue(n.Type),
			Status:    types.StringValue(n.Status),
			Shared:    types.BoolValue(n.Shared),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
