package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &serviceResource{}
var _ resource.ResourceWithImportState = &serviceResource{}

type serviceResource struct {
	client *client.Client
}

func NewServiceResource() resource.Resource {
	return &serviceResource{}
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *serviceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A health check or monitored service attached to a `middmonitor_host`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"host_id": schema.Int64Attribute{
				MarkdownDescription: "Parent host ID.",
				Required:            true,
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Check type: `http`, `ping`, `sql`, `certificate`, `snmp`, etc.",
				Required:              true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Target for the check (IP/hostname); often matches the parent host’s address.",
				Required:            true,
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "Logical app/service label (same meaning as on the host).",
				Required:            true,
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "HTTP path (for `http` checks).",
				Optional:            true,
			},
			"credentials": schema.StringAttribute{
				MarkdownDescription: "Optional JSON credentials (e.g. SQL, SNMP, HTTP auth). **Sensitive.**",
				Optional:            true,
				Sensitive:           true,
			},
			"service_interval": schema.Int64Attribute{
				MarkdownDescription: "Interval between checks in seconds.",
				Optional:            true,
				Computed:            true,
			},
			"max_attempts": schema.Int64Attribute{
				Optional: true,
				Computed: true,
			},
			"failure_threshold": schema.Float64Attribute{
				MarkdownDescription: "Optional threshold (e.g. max latency ms for HTTP).",
				Optional:            true,
				Computed:            true,
			},
			"expected_status_code": schema.Int64Attribute{
				MarkdownDescription: "For `http` checks: exact HTTP status code to treat as success. When unset, any 2xx response is a success (default).",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type serviceModel struct {
	ID                types.Int64   `tfsdk:"id"`
	HostID            types.Int64   `tfsdk:"host_id"`
	Name              types.String  `tfsdk:"name"`
	Type              types.String  `tfsdk:"type"`
	Hostname          types.String  `tfsdk:"hostname"`
	Service           types.String  `tfsdk:"service"`
	Path              types.String  `tfsdk:"path"`
	Credentials       types.String  `tfsdk:"credentials"`
	ServiceInterval    types.Int64   `tfsdk:"service_interval"`
	MaxAttempts        types.Int64   `tfsdk:"max_attempts"`
	FailureThreshold   types.Float64 `tfsdk:"failure_threshold"`
	ExpectedStatusCode types.Int64   `tfsdk:"expected_status_code"`
	CreatedAt          types.String  `tfsdk:"created_at"`
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hid := plan.HostID.ValueInt64()
	s := client.Service{
		HostID:          &hid,
		Name:            plan.Name.ValueString(),
		Type:            plan.Type.ValueString(),
		Host:            plan.Hostname.ValueString(),
		Service:         plan.Service.ValueString(),
		ServiceInterval: 60,
		MaxAttempts:     3,
	}
	if !plan.Path.IsNull() && plan.Path.ValueString() != "" {
		p := plan.Path.ValueString()
		s.Path = &p
	}
	if !plan.Credentials.IsNull() && plan.Credentials.ValueString() != "" {
		c := plan.Credentials.ValueString()
		s.Credentials = &c
	}
	if !plan.ServiceInterval.IsNull() {
		s.ServiceInterval = int(plan.ServiceInterval.ValueInt64())
	}
	if !plan.MaxAttempts.IsNull() {
		s.MaxAttempts = int(plan.MaxAttempts.ValueInt64())
	}
	if !plan.FailureThreshold.IsNull() {
		v := plan.FailureThreshold.ValueFloat64()
		s.FailureThreshold = &v
	}
	if !plan.ExpectedStatusCode.IsNull() {
		v := int(plan.ExpectedStatusCode.ValueInt64())
		s.ExpectedStatusCode = &v
	}

	out, err := r.client.CreateService(s)
	if err != nil {
		resp.Diagnostics.AddError("Create service failed", err.Error())
		return
	}
	mapServiceToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetService(state.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Read service failed", err.Error())
		return
	}
	mapServiceToModel(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state serviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hid := plan.HostID.ValueInt64()
	s := client.Service{
		HostID:          &hid,
		Name:            plan.Name.ValueString(),
		Type:            plan.Type.ValueString(),
		Host:            plan.Hostname.ValueString(),
		Service:         plan.Service.ValueString(),
		ServiceInterval: int(plan.ServiceInterval.ValueInt64()),
		MaxAttempts:     int(plan.MaxAttempts.ValueInt64()),
	}
	if !plan.Path.IsNull() && plan.Path.ValueString() != "" {
		p := plan.Path.ValueString()
		s.Path = &p
	}
	if !plan.Credentials.IsNull() && plan.Credentials.ValueString() != "" {
		c := plan.Credentials.ValueString()
		s.Credentials = &c
	}
	if !plan.FailureThreshold.IsNull() {
		v := plan.FailureThreshold.ValueFloat64()
		s.FailureThreshold = &v
	}
	if !plan.ExpectedStatusCode.IsNull() {
		v := int(plan.ExpectedStatusCode.ValueInt64())
		s.ExpectedStatusCode = &v
	}

	out, err := r.client.UpdateService(state.ID.ValueInt64(), s)
	if err != nil {
		resp.Diagnostics.AddError("Update service failed", err.Error())
		return
	}
	mapServiceToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteService(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete service failed", err.Error())
	}
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric service id")
		return
	}
	out, err := r.client.GetService(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}
	var m serviceModel
	mapServiceToModel(out, &m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}

func mapServiceToModel(out *client.Service, m *serviceModel) {
	m.ID = types.Int64Value(out.ID)
	if out.HostID != nil {
		m.HostID = types.Int64Value(*out.HostID)
	}
	m.Name = types.StringValue(out.Name)
	m.Type = types.StringValue(out.Type)
	m.Hostname = types.StringValue(out.Host)
	m.Service = types.StringValue(out.Service)
	if out.Path != nil {
		m.Path = types.StringValue(*out.Path)
	} else {
		m.Path = types.StringNull()
	}
	if out.Credentials != nil {
		m.Credentials = types.StringValue(*out.Credentials)
	} else {
		m.Credentials = types.StringNull()
	}
	m.ServiceInterval = types.Int64Value(int64(out.ServiceInterval))
	m.MaxAttempts = types.Int64Value(int64(out.MaxAttempts))
	if out.FailureThreshold != nil {
		m.FailureThreshold = types.Float64Value(*out.FailureThreshold)
	} else {
		m.FailureThreshold = types.Float64Null()
	}
	if out.ExpectedStatusCode != nil {
		m.ExpectedStatusCode = types.Int64Value(int64(*out.ExpectedStatusCode))
	} else {
		m.ExpectedStatusCode = types.Int64Null()
	}
	m.CreatedAt = types.StringValue(out.CreatedAt)
}
