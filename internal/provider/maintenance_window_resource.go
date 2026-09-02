package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &maintenanceWindowResource{}
var _ resource.ResourceWithImportState = &maintenanceWindowResource{}

type maintenanceWindowResource struct {
	client *client.Client
}

func NewMaintenanceWindowResource() resource.Resource {
	return &maintenanceWindowResource{}
}

func (r *maintenanceWindowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_maintenance_window"
}

func (r *maintenanceWindowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *maintenanceWindowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A period during which alerts on one target are suppressed. " +
			"The API has no update for these, so any change replaces the window.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Maintenance window ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Why the window exists, shown on the suppressed alert.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_type": schema.StringAttribute{
				MarkdownDescription: "`service` or `host`.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_id": schema.Int64Attribute{
				MarkdownDescription: "ID of the service or host whose alerts are suppressed.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"starts_at": schema.StringAttribute{
				MarkdownDescription: "Start of the window, RFC3339.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"ends_at": schema.StringAttribute{
				MarkdownDescription: "End of the window, RFC3339.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

type maintenanceWindowModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	TargetType types.String `tfsdk:"target_type"`
	TargetID   types.Int64  `tfsdk:"target_id"`
	StartsAt   types.String `tfsdk:"starts_at"`
	EndsAt     types.String `tfsdk:"ends_at"`
}

func (m maintenanceWindowModel) toAPI() client.MaintenanceWindow {
	return client.MaintenanceWindow{
		Name:       m.Name.ValueString(),
		TargetType: m.TargetType.ValueString(),
		TargetID:   m.TargetID.ValueInt64(),
		StartsAt:   m.StartsAt.ValueString(),
		EndsAt:     m.EndsAt.ValueString(),
	}
}

func (r *maintenanceWindowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan maintenanceWindowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateMaintenanceWindow(plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Create maintenance window failed", err.Error())
		return
	}
	plan.ID = types.Int64Value(out.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *maintenanceWindowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetMaintenanceWindow(state.ID.ValueInt64())
	if errors.Is(err, client.ErrNotFound) {
		// A window that has been deleted (or expired and cleaned up) must leave
		// the state, or every later plan fails on a row that no longer exists.
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read maintenance window failed", err.Error())
		return
	}
	state.ID = types.Int64Value(out.ID)
	state.Name = types.StringValue(out.Name)
	state.TargetType = types.StringValue(out.TargetType)
	state.TargetID = types.Int64Value(out.TargetID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update cannot happen: every attribute forces a replacement.
func (r *maintenanceWindowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan maintenanceWindowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *maintenanceWindowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state maintenanceWindowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMaintenanceWindow(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete maintenance window failed", err.Error())
	}
}

func (r *maintenanceWindowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric maintenance window id")
		return
	}
	out, err := r.client.GetMaintenanceWindow(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}
	m := maintenanceWindowModel{
		ID:         types.Int64Value(out.ID),
		Name:       types.StringValue(out.Name),
		TargetType: types.StringValue(out.TargetType),
		TargetID:   types.Int64Value(out.TargetID),
		StartsAt:   types.StringValue(out.StartsAt),
		EndsAt:     types.StringValue(out.EndsAt),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
