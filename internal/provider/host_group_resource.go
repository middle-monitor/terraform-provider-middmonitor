package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &hostGroupResource{}
var _ resource.ResourceWithImportState = &hostGroupResource{}

type hostGroupResource struct {
	client *client.Client
}

func NewHostGroupResource() resource.Resource {
	return &hostGroupResource{}
}

func (r *hostGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_host_group"
}

func (r *hostGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *hostGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A group of hosts. Alert correlation is scoped to a host and its group, so the grouping is structural rather than cosmetic.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Host group ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Technical name, unique per organization.",
				Required:            true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable label shown in the UI.",
				Optional:            true,
			},
			"is_default": schema.BoolAttribute{
				MarkdownDescription: "Whether this is the organization's default group.",
				Computed:            true,
			},
		},
	}
}

type hostGroupModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (m hostGroupModel) toAPI() client.HostGroup {
	group := client.HostGroup{Name: m.Name.ValueString()}
	if !m.DisplayName.IsNull() && m.DisplayName.ValueString() != "" {
		value := m.DisplayName.ValueString()
		group.DisplayName = &value
	}
	return group
}

func mapHostGroupToModel(out *client.HostGroup, m *hostGroupModel) {
	m.ID = types.Int64Value(out.ID)
	m.Name = types.StringValue(out.Name)
	if out.DisplayName != nil && *out.DisplayName != "" {
		m.DisplayName = types.StringValue(*out.DisplayName)
	} else {
		m.DisplayName = types.StringNull()
	}
	m.IsDefault = types.BoolValue(out.IsDefault)
}

func (r *hostGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hostGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateHostGroup(plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create host group failed", err.Error())
		return
	}
	mapHostGroupToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state hostGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetHostGroup(state.ID.ValueInt64())
	if errors.Is(err, client.ErrNotFound) {
		// Removed outside Terraform: drop it from state so the next plan
		// recreates it instead of failing forever.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read host group failed", err.Error())
		return
	}
	mapHostGroupToModel(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *hostGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state hostGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateHostGroup(state.ID.ValueInt64(), plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Update host group failed", err.Error())
		return
	}
	mapHostGroupToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *hostGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hostGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteHostGroup(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete host group failed", err.Error())
	}
}

func (r *hostGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric host group id")
		return
	}
	out, err := r.client.GetHostGroup(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}
	var m hostGroupModel
	mapHostGroupToModel(out, &m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
