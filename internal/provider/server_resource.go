package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/client"
)

const (
	serverCreateTimeout = 30 * time.Minute
	serverDeleteTimeout = 20 * time.Minute
	serverResizeTimeout = 15 * time.Minute
)

var (
	_ resource.Resource                = (*serverResource)(nil)
	_ resource.ResourceWithConfigure   = (*serverResource)(nil)
	_ resource.ResourceWithImportState = (*serverResource)(nil)
)

// NewServerResource 는 ktcloud_server 리소스를 생성합니다.
func NewServerResource() resource.Resource { return &serverResource{} }

type serverResource struct {
	client *client.Client
}

type serverNetworkModel struct {
	UUID    types.String `tfsdk:"uuid"`
	FixedIP types.String `tfsdk:"fixed_ip"`
}

var serverNetworkAttrTypes = map[string]attr.Type{
	"uuid":     types.StringType,
	"fixed_ip": types.StringType,
}

type serverResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	FlavorID         types.String `tfsdk:"flavor_id"`
	ImageID          types.String `tfsdk:"image_id"`
	KeypairName      types.String `tfsdk:"keypair_name"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	UserData         types.String `tfsdk:"user_data"`
	RootVolumeSize   types.Int64  `tfsdk:"root_volume_size"`
	RootVolumeType   types.String `tfsdk:"root_volume_type"`
	Networks         types.List   `tfsdk:"networks"`
	Metadata         types.Map    `tfsdk:"metadata"`
	Status           types.String `tfsdk:"status"`
	PrivateIP        types.String `tfsdk:"private_ip"`
	PublicIP         types.String `tfsdk:"public_ip"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Provider 데이터 타입 오류",
			fmt.Sprintf("*client.Client 를 기대했으나 %T 를 받았습니다.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "kt cloud 가상 서버(VM) 를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "VM 이름. 변경 시 재생성 없이 이름만 수정됩니다.",
			},
			"flavor_id": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "VM 스펙 ID. `data.ktcloud_flavors` 로 조회할 수 있습니다. " +
					"변경 시 재생성 없이 resize API 로 처리됩니다 (kt cloud 쪽 사정으로 재부팅이 " +
					"발생할 수 있습니다).",
			},
			"image_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "OS 이미지 ID. `data.ktcloud_images` 로 조회할 수 있습니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keypair_name": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "SSH 키페어 이름. kt cloud API 상 필수값입니다 " +
					"(Null 불가).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "가용 영역.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_data": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "cloud-init 스크립트. **평문으로 입력**하면 Provider 가 base64 로 인코딩해 전송합니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"root_volume_size": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "루트 볼륨 크기(GB). 지정하면 이미지를 볼륨으로 부팅합니다" +
					"(block_device_mapping_v2). 미지정 시 이미지 기본값을 사용합니다.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"root_volume_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "루트 볼륨 타입. `root_volume_size` 와 함께 사용합니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"metadata": schema.MapAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "VM 메타데이터 key/value.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "VM 상태 (ACTIVE, SHUTOFF 등).",
			},
			"private_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "사설 IP.",
			},
			"public_ip": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "공인(floating) IP. 할당된 경우에만 값이 있습니다.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "생성 시각.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"networks": schema.ListNestedBlock{
				MarkdownDescription: "연결할 네트워크(Tier) 목록. 순서가 NIC 순서를 결정합니다.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "네트워크(Tier) UUID.",
						},
						"fixed_ip": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "지정할 사설 IP. 생략하면 자동 할당됩니다.",
						},
					},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := client.CreateServerOpts{
		Name:             plan.Name.ValueString(),
		FlavorRef:        plan.FlavorID.ValueString(),
		KeyName:          stringValue(plan.KeypairName),
		AvailabilityZone: stringValue(plan.AvailabilityZone),
	}

	if ud := stringValue(plan.UserData); ud != "" {
		opts.UserData = base64.StdEncoding.EncodeToString([]byte(ud))
	}

	// 네트워크
	if !plan.Networks.IsNull() && !plan.Networks.IsUnknown() {
		var nets []serverNetworkModel
		resp.Diagnostics.Append(plan.Networks.ElementsAs(ctx, &nets, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, n := range nets {
			opts.Networks = append(opts.Networks, client.NetworkOpts{
				UUID:    n.UUID.ValueString(),
				FixedIP: stringValue(n.FixedIP),
			})
		}
	}

	// 메타데이터
	if !plan.Metadata.IsNull() && !plan.Metadata.IsUnknown() {
		meta := map[string]string{}
		resp.Diagnostics.Append(plan.Metadata.ElementsAs(ctx, &meta, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts.Metadata = meta
	}

	// 루트 볼륨 크기를 지정하면 block_device_mapping_v2 로 부팅합니다.
	if !plan.RootVolumeSize.IsNull() && plan.RootVolumeSize.ValueInt64() > 0 {
		opts.BlockDevices = []client.BlockDevice{{
			BootIndex:           "0",
			UUID:                plan.ImageID.ValueString(),
			SourceType:          "image",
			DestinationType:     "volume",
			VolumeSize:          plan.RootVolumeSize.ValueInt64(),
			DeleteOnTermination: true,
			VolumeType:          stringValue(plan.RootVolumeType),
		}}
	} else {
		opts.ImageRef = plan.ImageID.ValueString()
	}

	created, err := r.client.CreateServer(ctx, opts)
	if err != nil {
		resp.Diagnostics.AddError("VM 생성 실패", err.Error())
		return
	}

	srv, err := r.client.WaitForServerStatus(ctx, created.ID, "ACTIVE", serverCreateTimeout)
	if err != nil {
		// ID 는 저장해야 사용자가 이후 destroy 로 정리할 수 있습니다.
		plan.ID = types.StringValue(created.ID)
		resp.State.Set(ctx, &plan)
		resp.Diagnostics.AddError("VM 이 ACTIVE 상태가 되지 못했습니다", err.Error())
		return
	}

	r.applyServerToModel(&plan, srv)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	srv, err := r.client.GetServer(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("VM 조회 실패", err.Error())
		return
	}

	r.applyServerToModel(&state, srv)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	if plan.Name.ValueString() != state.Name.ValueString() {
		if err := r.client.RenameServer(ctx, plan.ID.ValueString(), plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("VM 이름 변경 실패", err.Error())
			return
		}
	}

	if plan.FlavorID.ValueString() != state.FlavorID.ValueString() {
		if err := r.resizeServer(ctx, plan.ID.ValueString(), plan.FlavorID.ValueString()); err != nil {
			resp.Diagnostics.AddError("VM 리사이즈 실패", err.Error())
			return
		}
	}

	srv, err := r.client.GetServer(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("VM 조회 실패", err.Error())
		return
	}

	r.applyServerToModel(&plan, srv)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// resizeServer 는 flavor 변경을 kt cloud resize API 로 처리합니다.
// resize → VERIFY_RESIZE 대기 → confirmResize → ACTIVE 대기 순서입니다.
// confirmResize 를 안 하면 kt cloud 가 일정 시간 후 자동 confirm 하거나 되돌릴 수 있어
// 명시적으로 확정합니다.
func (r *serverResource) resizeServer(ctx context.Context, id, flavorID string) error {
	body := map[string]any{"resize": map[string]string{"flavorRef": flavorID}}
	if err := r.client.ServerAction(ctx, id, body); err != nil {
		return fmt.Errorf("resize 요청 실패: %w", err)
	}
	if _, err := r.client.WaitForServerStatus(ctx, id, "VERIFY_RESIZE", serverResizeTimeout); err != nil {
		return fmt.Errorf("VERIFY_RESIZE 상태 대기 실패: %w", err)
	}
	if err := r.client.ServerAction(ctx, id, map[string]any{"confirmResize": nil}); err != nil {
		return fmt.Errorf("resize 확정(confirmResize) 실패: %w", err)
	}
	if _, err := r.client.WaitForServerStatus(ctx, id, "ACTIVE", serverResizeTimeout); err != nil {
		return fmt.Errorf("리사이즈 후 ACTIVE 상태 대기 실패: %w", err)
	}
	return nil
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serverResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.DeleteServer(ctx, id); err != nil {
		resp.Diagnostics.AddError("VM 삭제 실패", err.Error())
		return
	}
	if err := r.client.WaitForServerDeleted(ctx, id, serverDeleteTimeout); err != nil {
		resp.Diagnostics.AddError("VM 삭제 대기 실패", err.Error())
	}
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyServerToModel 은 API 응답을 상태 모델에 반영합니다.
// networks / user_data 등 API 가 그대로 돌려주지 않는 값은 건드리지 않습니다.
func (r *serverResource) applyServerToModel(m *serverResourceModel, srv *client.Server) {
	m.ID = types.StringValue(srv.ID)
	m.Name = types.StringValue(srv.Name)
	m.Status = types.StringValue(srv.Status)
	m.CreatedAt = types.StringValue(srv.Created)
	m.PrivateIP = types.StringValue(srv.FixedIP())
	m.PublicIP = types.StringValue(srv.FloatingIP())

	if srv.Flavor.ID != "" {
		m.FlavorID = types.StringValue(srv.Flavor.ID)
	}
	if srv.Image.ID != "" {
		m.ImageID = types.StringValue(srv.Image.ID)
	}
	if srv.KeyName != "" {
		m.KeypairName = types.StringValue(srv.KeyName)
	}

	// import 직후 networks 가 null 이면 빈 목록으로 초기화해 둡니다.
	if m.Networks.IsUnknown() {
		m.Networks = types.ListNull(types.ObjectType{AttrTypes: serverNetworkAttrTypes})
	}
}
