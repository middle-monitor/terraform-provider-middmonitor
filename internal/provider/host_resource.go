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

var _ resource.Resource = &hostResource{}
var _ resource.ResourceWithImportState = &hostResource{}

type hostResource struct {
	client *client.Client
}

func NewHostResource() resource.Resource {
	return &hostResource{}
}

func (r *hostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host"
}

func (r *hostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A monitored host (inventory + agent target). `name` is immutable after creation (API).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Host ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Technical host name (unique per org). **Cannot be changed** after create.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Resolvable address or IP used for checks (maps to API field `host`). Changing this recreates the host (API only allows editing the display name).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "Logical application / stack name (defaults to `name` if omitted).",
				Optional:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable label in the UI.",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp (RFC3339).",
				Computed:            true,
			},
		},
	}
}

type hostModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Hostname    types.String `tfsdk:"hostname"`
	Service     types.String `tfsdk:"service"`
	DisplayName types.String `tfsdk:"display_name"`
	CreatedAt   types.String `tfsdk:"created_at"`
}

func (r *hostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	h := client.Host{
		Name: plan.Name.ValueString(),
		Host: plan.Hostname.ValueString(),
	}
	if !plan.Service.IsNull() && plan.Service.ValueString() != "" {
		h.Service = plan.Service.ValueString()
	}
	if !plan.DisplayName.IsNull() && plan.DisplayName.ValueString() != "" {
		s := plan.DisplayName.ValueString()
		h.DisplayName = &s
	}

	out, err := r.client.CreateHost(h)
	if err != nil {
		resp.Diagnostics.AddError("Create host failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(out.ID)
	plan.CreatedAt = types.StringValue(out.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetHost(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Read host failed", err.Error())
		return
	}
	mapHostToModel(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state hostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// API only updates display_name on hosts.
	h := client.Host{}
	if !plan.DisplayName.IsNull() {
		s := plan.DisplayName.ValueString()
		if s != "" {
			h.DisplayName = &s
		}
	}

	out, err := r.client.UpdateHost(state.ID.ValueInt64(), h)
	if err != nil {
		resp.Diagnostics.AddError("Update host failed", err.Error())
		return
	}
	state.DisplayName = plan.DisplayName
	state.CreatedAt = types.StringValue(out.CreatedAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteHost(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete host failed", err.Error())
	}
}

func (r *hostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric host id")
		return
	}
	out, err := r.client.GetHost(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}
	var m hostModel
	mapHostToModel(out, &m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func mapHostToModel(out *client.Host, m *hostModel) {
	m.ID = types.Int64Value(out.ID)
	m.Name = types.StringValue(out.Name)
	m.Hostname = types.StringValue(out.Host)
	m.Service = types.StringValue(out.Service)
	if out.DisplayName != nil {
		m.DisplayName = types.StringValue(*out.DisplayName)
	} else {
		m.DisplayName = types.StringNull()
	}
	m.CreatedAt = types.StringValue(out.CreatedAt)
}
