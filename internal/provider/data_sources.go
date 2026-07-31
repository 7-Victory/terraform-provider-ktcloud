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
