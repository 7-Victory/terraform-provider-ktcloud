package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/client"
)

const volumeOpTimeout = 15 * time.Minute

// ---------------------------------------------------------------------------
// ktcloud_volume
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = (*volumeResource)(nil)
	_ resource.ResourceWithConfigure   = (*volumeResource)(nil)
	_ resource.ResourceWithImportState = (*volumeResource)(nil)
)

// NewVolumeResource 는 ktcloud_volume 리소스를 생성합니다.
func NewVolumeResource() resource.Resource { return &volumeResource{} }

type volumeResource struct {
	client *client.Client
}

type volumeResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Size             types.Int64  `tfsdk:"size"`
	VolumeType       types.String `tfsdk:"volume_type"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	SnapshotID       types.String `tfsdk:"snapshot_id"`
	UsagePlanType    types.String `tfsdk:"usage_plan_type"`
	Bootable         types.Bool   `tfsdk:"bootable"`
	Status           types.String `tfsdk:"status"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func (r *volumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume"
}

func (r *volumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *volumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "kt cloud 블록 스토리지 볼륨을 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "볼륨 UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "볼륨 이름.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "볼륨 설명.",
			},
			"size": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "볼륨 크기(GB). 값을 늘리면 온라인 확장(os-extend)을 시도하며, 줄이면 오류가 발생합니다.",
			},
			"volume_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "볼륨 타입.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"availability_zone": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "가용 영역.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapshot_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "스냅샷으로부터 복원할 경우의 스냅샷 ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"usage_plan_type": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "과금 단위. `hourly` 또는 `monthly` (kt cloud 기본값 " +
					"`monthly`). 미검증 옵션입니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bootable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "부팅 가능한 볼륨으로 생성할지 여부. 기본값 `false`. " +
					"조회 시 kt cloud 응답(문자열 `\"true\"`/`\"false\"`)으로 갱신됩니다.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "볼륨 상태 (available, in-use 등).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "생성 시각.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *volumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vol, err := r.client.CreateVolume(ctx, client.CreateVolumeOpts{
		Name:          stringValue(plan.Name),
		Description:   stringValue(plan.Description),
		Size:          plan.Size.ValueInt64(),
		VolumeType:    stringValue(plan.VolumeType),
		Zone:          stringValue(plan.AvailabilityZone),
		SnapshotID:    stringValue(plan.SnapshotID),
		UsagePlanType: stringValue(plan.UsagePlanType),
		Bootable:      boolValue(plan.Bootable),
	})
	if err != nil {
		resp.Diagnostics.AddError("볼륨 생성 실패", err.Error())
		return
	}

	ready, err := r.client.WaitForVolumeStatus(ctx, vol.ID, []string{"available", "in-use"}, volumeOpTimeout)
	if err != nil {
		plan.ID = types.StringValue(vol.ID)
		resp.State.Set(ctx, &plan)
		resp.Diagnostics.AddError("볼륨이 사용 가능 상태가 되지 못했습니다", err.Error())
		return
	}

	applyVolumeToModel(&plan, ready)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vol, err := r.client.GetVolume(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("볼륨 조회 실패", err.Error())
		return
	}

	applyVolumeToModel(&state, vol)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *volumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state volumeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	plan.ID = state.ID

	// 이름/설명 수정
	if stringValue(plan.Name) != stringValue(state.Name) || stringValue(plan.Description) != stringValue(state.Description) {
		if err := r.client.UpdateVolume(ctx, id, stringValue(plan.Name), stringValue(plan.Description)); err != nil {
			resp.Diagnostics.AddError("볼륨 정보 수정 실패", err.Error())
			return
		}
	}

	// 크기 확장
	newSize := plan.Size.ValueInt64()
	oldSize := state.Size.ValueInt64()
	if newSize != oldSize {
		if newSize < oldSize {
			resp.Diagnostics.AddError(
				"볼륨 크기 축소 불가",
				fmt.Sprintf("현재 %dGB 에서 %dGB 로 줄일 수 없습니다. 확장만 가능합니다.", oldSize, newSize),
			)
			return
		}
		if err := r.client.ExtendVolume(ctx, id, newSize); err != nil {
			resp.Diagnostics.AddError("볼륨 확장 실패", err.Error())
			return
		}
		if _, err := r.client.WaitForVolumeStatus(ctx, id, []string{"available", "in-use"}, volumeOpTimeout); err != nil {
			resp.Diagnostics.AddError("볼륨 확장 대기 실패", err.Error())
			return
		}
	}

	vol, err := r.client.GetVolume(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("볼륨 조회 실패", err.Error())
		return
	}

	applyVolumeToModel(&plan, vol)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteVolume(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("볼륨 삭제 실패", err.Error())
	}
}

func (r *volumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyVolumeToModel(m *volumeResourceModel, vol *client.Volume) {
	m.ID = types.StringValue(vol.ID)
	m.Size = types.Int64Value(vol.Size)
	m.Status = types.StringValue(vol.Status)
	m.CreatedAt = types.StringValue(vol.CreatedAt)
	m.VolumeType = types.StringValue(vol.VolumeType)
	m.AvailabilityZone = types.StringValue(vol.Zone)
	// kt cloud 는 조회 응답에서 bootable 을 "true"/"false" 문자열로 돌려줍니다.
	m.Bootable = types.BoolValue(strings.EqualFold(vol.Bootable, "true"))

	if vol.Name != "" {
		m.Name = types.StringValue(vol.Name)
	}
	if vol.Description != "" {
		m.Description = types.StringValue(vol.Description)
	}
}

// ---------------------------------------------------------------------------
// ktcloud_volume_attachment
// ---------------------------------------------------------------------------

var (
	_ resource.Resource                = (*volumeAttachmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*volumeAttachmentResource)(nil)
	_ resource.ResourceWithImportState = (*volumeAttachmentResource)(nil)
)

// NewVolumeAttachmentResource 는 ktcloud_volume_attachment 리소스를 생성합니다.
func NewVolumeAttachmentResource() resource.Resource { return &volumeAttachmentResource{} }

type volumeAttachmentResource struct {
	client *client.Client
}

type volumeAttachmentModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	VolumeID types.String `tfsdk:"volume_id"`
	Device   types.String `tfsdk:"device"`
}

func (r *volumeAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_attachment"
}

func (r *volumeAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *volumeAttachmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "볼륨을 VM 에 연결합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "`<server_id>/<volume_id>` 형식의 복합 ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"server_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "VM UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"volume_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "볼륨 UUID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"device": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "게스트 OS 상의 디바이스 경로 (예: `/dev/vdb`). 생략하면 자동 할당됩니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *volumeAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan volumeAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	volumeID := plan.VolumeID.ValueString()

	att, err := r.client.AttachVolume(ctx, serverID, volumeID, stringValue(plan.Device))
	if err != nil {
		resp.Diagnostics.AddError("볼륨 연결 실패", err.Error())
		return
	}

	if _, err := r.client.WaitForVolumeStatus(ctx, volumeID, []string{"in-use"}, volumeOpTimeout); err != nil {
		resp.Diagnostics.AddError("볼륨 연결 대기 실패", err.Error())
		return
	}

	plan.ID = types.StringValue(serverID + "/" + volumeID)
	plan.Device = types.StringValue(att.Device)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state volumeAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	att, err := r.client.GetVolumeAttachment(ctx, state.ServerID.ValueString(), state.VolumeID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("볼륨 연결 조회 실패", err.Error())
		return
	}

	state.Device = types.StringValue(att.Device)
	state.ID = types.StringValue(state.ServerID.ValueString() + "/" + state.VolumeID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// 모든 필드가 RequiresReplace 이므로 실제로 호출되지 않습니다.
func (r *volumeAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan volumeAttachmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *volumeAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state volumeAttachmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	volumeID := state.VolumeID.ValueString()
	if err := r.client.DetachVolume(ctx, state.ServerID.ValueString(), volumeID); err != nil {
		resp.Diagnostics.AddError("볼륨 연결 해제 실패", err.Error())
		return
	}
	if _, err := r.client.WaitForVolumeStatus(ctx, volumeID, []string{"available"}, volumeOpTimeout); err != nil {
		// 볼륨이 함께 삭제되는 중일 수 있으므로 경고만 남깁니다.
		resp.Diagnostics.AddWarning("볼륨 연결 해제 확인 실패", err.Error())
	}
}

func (r *volumeAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"import ID 형식 오류",
			fmt.Sprintf("`<server_id>/<volume_id>` 형식이어야 합니다. 입력값: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("volume_id"), parts[1])...)
}
