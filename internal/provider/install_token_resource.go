package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &installTokenResource{}
var _ resource.ResourceWithImportState = &installTokenResource{}

type installTokenResource struct {
	client *client.Client
}

func NewInstallTokenResource() resource.Resource {
	return &installTokenResource{}
}

func (r *installTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_install_token"
}

func (r *installTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	rd, ok := req.ProviderData.(*resourceData)
	if !ok {
		resp.Diagnostics.AddError("Internal error", "invalid provider data")
		return
	}
	r.client = rd.Client
}

func (r *installTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates an **install token** for the agent (receiver API). The secret is shown once; store it in Terraform state or a secret manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Label for this token in the UI.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"expires_at": schema.StringAttribute{
				MarkdownDescription: "Optional RFC3339 expiry (e.g. `2026-12-31T23:59:59Z`).",
				Optional:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "Secret token (only returned on create).",
				Computed:            true,
				Sensitive:           true,
			},
			"token_prefix": schema.StringAttribute{
				MarkdownDescription: "Short prefix for display (if returned by API).",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type installTokenModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
	Token       types.String `tfsdk:"token"`
	TokenPrefix types.String `tfsdk:"token_prefix"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (r *installTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan installTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var exp *string
	if !plan.ExpiresAt.IsNull() && plan.ExpiresAt.ValueString() != "" {
		s := plan.ExpiresAt.ValueString()
		exp = &s
	}

	out, err := r.client.CreateInstallToken(plan.Name.ValueString(), exp)
	if err != nil {
		resp.Diagnostics.AddError("Create install token failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(out.ID)
	plan.Token = types.StringValue(out.Token)
	if out.TokenPrefix != "" {
		plan.TokenPrefix = types.StringValue(out.TokenPrefix)
	} else {
		plan.TokenPrefix = types.StringNull()
	}
	plan.CreatedAt = types.StringValue(out.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *installTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// API does not return full token on list/read — keep state as-is.
	var state installTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// No GET by id for tokens — refresh only updates non-secret metadata if we had an endpoint.
	// Mark token as unknown on read if we re-fetch; simplest: no-op read keeps state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *installTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan installTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *installTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state installTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInstallToken(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete install token failed", err.Error())
	}
}

func (r *installTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric install token id")
		return
	}
	var m installTokenModel
	m.ID = types.Int64Value(id)
	m.Name = types.StringValue("imported")
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
