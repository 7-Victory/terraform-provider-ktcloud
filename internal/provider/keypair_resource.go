package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/7-Victory/terraform-provider-ktcloud/internal/client"
)

var (
	_ resource.Resource                = (*keypairResource)(nil)
	_ resource.ResourceWithConfigure   = (*keypairResource)(nil)
	_ resource.ResourceWithImportState = (*keypairResource)(nil)
)

// NewKeypairResource 는 ktcloud_keypair 리소스를 생성합니다.
func NewKeypairResource() resource.Resource { return &keypairResource{} }

type keypairResource struct {
	client *client.Client
}

type keypairResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	PrivateKey  types.String `tfsdk:"private_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func (r *keypairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keypair"
}

func (r *keypairResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *keypairResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "kt cloud SSH 키페어를 관리합니다.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "키페어 이름과 동일합니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "키페어 이름.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "등록할 공개키(OpenSSH 형식). 생략하면 kt cloud 가 새 키를 생성하고 " +
					"`private_key` 에 개인키가 담깁니다.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				MarkdownDescription: "새로 생성된 경우에만 값이 있습니다. **state 에 평문 저장되므로 " +
					"state 파일 보안에 주의하세요.**",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "키 지문.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *keypairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keypairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	kp, err := r.client.CreateKeypair(ctx, plan.Name.ValueString(), stringValue(plan.PublicKey))
	if err != nil {
		resp.Diagnostics.AddError("키페어 생성 실패", err.Error())
		return
	}

	plan.ID = types.StringValue(kp.Name)
	plan.Name = types.StringValue(kp.Name)
	plan.PublicKey = types.StringValue(kp.PublicKey)
	plan.PrivateKey = types.StringValue(kp.PrivateKey)
	plan.Fingerprint = types.StringValue(kp.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keypairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := state.Name.ValueString()
	if name == "" {
		name = state.ID.ValueString()
	}

	kp, err := r.client.GetKeypair(ctx, name)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("키페어 조회 실패", err.Error())
		return
	}

	state.ID = types.StringValue(kp.Name)
	state.Name = types.StringValue(kp.Name)
	state.PublicKey = types.StringValue(kp.PublicKey)
	state.Fingerprint = types.StringValue(kp.Fingerprint)
	if state.PrivateKey.IsNull() || state.PrivateKey.IsUnknown() {
		state.PrivateKey = types.StringValue("")
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// 모든 수정 가능 필드가 RequiresReplace 이므로 Update 는 호출되지 않습니다.
func (r *keypairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan keypairResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *keypairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keypairResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteKeypair(ctx, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("키페어 삭제 실패", err.Error())
	}
}

func (r *keypairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// terraform import ktcloud_keypair.this <키페어이름>
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
