package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &alertRuleResource{}
var _ resource.ResourceWithImportState = &alertRuleResource{}

type alertRuleResource struct {
	client *client.Client
}

func NewAlertRuleResource() resource.Resource {
	return &alertRuleResource{}
}

func (r *alertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *alertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A threshold rule evaluated every minute. Set `metric` for a built-in signal " +
			"(`cpu`, `ram`, `disk`, `latency`, `error_count`, `failure_rate`) or `custom_metric` to watch " +
			"a scraped series, which is what an agent scrape produces.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Alert rule ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Rule name. It becomes the incident title.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Free-text description.",
				Optional:            true,
			},
			"metric": schema.StringAttribute{
				MarkdownDescription: "Built-in signal: `cpu`, `ram`, `disk`, `latency`, `error_count`, `failure_rate`. Leave unset when using `custom_metric`.",
				Optional:            true,
			},
			"custom_metric": schema.StringAttribute{
				MarkdownDescription: "Name of a scraped or SDK-reported series, e.g. `node_load1`.",
				Optional:            true,
			},
			"custom_labels": schema.MapAttribute{
				MarkdownDescription: "Label equality constraints narrowing the custom metric series.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"target_type": schema.StringAttribute{
				MarkdownDescription: "`any` (default), `service` or `host`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("any"),
			},
			"target_id": schema.Int64Attribute{
				MarkdownDescription: "The service or host the rule is scoped to, when `target_type` is not `any`.",
				Optional:            true,
			},
			"aggregation": schema.StringAttribute{
				MarkdownDescription: "`avg` (default), `min`, `max`, `sum`, `p50`, `p75`, `p90`, `p95`, `p99`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("avg"),
			},
			"operator": schema.StringAttribute{
				MarkdownDescription: "`gt`, `gte`, `lt`, `lte` or `eq`.",
				Required:            true,
			},
			"threshold": schema.Float64Attribute{
				MarkdownDescription: "Single threshold. Ignored when `warning_threshold` or `critical_threshold` is set.",
				Optional:            true,
				Computed:            true,
			},
			"warning_threshold": schema.Float64Attribute{
				MarkdownDescription: "Raises a warning when crossed.",
				Optional:            true,
			},
			"critical_threshold": schema.Float64Attribute{
				MarkdownDescription: "Raises a critical alert when crossed. Wins over the warning threshold.",
				Optional:            true,
			},
			"recovery_threshold": schema.Float64Attribute{
				MarkdownDescription: "Hysteresis: the incident resolves only once the value crosses back past this.",
				Optional:            true,
			},
			"duration": schema.Int64Attribute{
				MarkdownDescription: "Evaluation window in seconds. Defaults to 300.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(300),
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Severity of the single-threshold form: `warning` or `critical`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("critical"),
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the rule is evaluated. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"channels": schema.ListAttribute{
				MarkdownDescription: "Notification channel IDs. Empty means every enabled channel of the organization.",
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"notify_warning": schema.BoolAttribute{
				MarkdownDescription: "Deliver warning-severity alerts on these channels. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"notify_critical": schema.BoolAttribute{
				MarkdownDescription: "Deliver critical-severity alerts on these channels. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

type alertRuleModel struct {
	ID                types.Int64   `tfsdk:"id"`
	Name              types.String  `tfsdk:"name"`
	Description       types.String  `tfsdk:"description"`
	Metric            types.String  `tfsdk:"metric"`
	CustomMetric      types.String  `tfsdk:"custom_metric"`
	CustomLabels      types.Map     `tfsdk:"custom_labels"`
	TargetType        types.String  `tfsdk:"target_type"`
	TargetID          types.Int64   `tfsdk:"target_id"`
	Aggregation       types.String  `tfsdk:"aggregation"`
	Operator          types.String  `tfsdk:"operator"`
	Threshold         types.Float64 `tfsdk:"threshold"`
	WarningThreshold  types.Float64 `tfsdk:"warning_threshold"`
	CriticalThreshold types.Float64 `tfsdk:"critical_threshold"`
	RecoveryThreshold types.Float64 `tfsdk:"recovery_threshold"`
	Duration          types.Int64   `tfsdk:"duration"`
	Severity          types.String  `tfsdk:"severity"`
	Enabled           types.Bool    `tfsdk:"enabled"`
	Channels          types.List    `tfsdk:"channels"`
	NotifyWarning     types.Bool    `tfsdk:"notify_warning"`
	NotifyCritical    types.Bool    `tfsdk:"notify_critical"`
}

func optionalFloat(value types.Float64) *float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueFloat64()
	return &v
}

func (m alertRuleModel) toAPI(ctx context.Context) (client.AlertRule, error) {
	rule := client.AlertRule{
		Name:              m.Name.ValueString(),
		TargetType:        m.TargetType.ValueString(),
		Metric:            m.Metric.ValueString(),
		Aggregation:       m.Aggregation.ValueString(),
		Operator:          m.Operator.ValueString(),
		Threshold:         m.Threshold.ValueFloat64(),
		WarningThreshold:  optionalFloat(m.WarningThreshold),
		CriticalThreshold: optionalFloat(m.CriticalThreshold),
		RecoveryThreshold: optionalFloat(m.RecoveryThreshold),
		Duration:          int(m.Duration.ValueInt64()),
		Severity:          m.Severity.ValueString(),
		Enabled:           m.Enabled.ValueBool(),
		NotifyWarning:     m.NotifyWarning.ValueBool(),
		NotifyCritical:    m.NotifyCritical.ValueBool(),
	}

	if !m.Description.IsNull() && m.Description.ValueString() != "" {
		value := m.Description.ValueString()
		rule.Description = &value
	}
	if !m.CustomMetric.IsNull() && m.CustomMetric.ValueString() != "" {
		value := m.CustomMetric.ValueString()
		rule.CustomMetric = &value
	}
	if !m.TargetID.IsNull() {
		value := m.TargetID.ValueInt64()
		rule.TargetID = &value
	}

	if !m.CustomLabels.IsNull() {
		labels := map[string]string{}
		if diags := m.CustomLabels.ElementsAs(ctx, &labels, false); diags.HasError() {
			return rule, errors.New("custom_labels must be a map of strings")
		}
		for key, value := range labels {
			rule.CustomLabels = append(rule.CustomLabels, client.MetricLabel{Key: key, Value: value})
		}
	}

	if !m.Channels.IsNull() {
		var channels []int64
		if diags := m.Channels.ElementsAs(ctx, &channels, false); diags.HasError() {
			return rule, errors.New("channels must be a list of numbers")
		}
		rule.Channels = channels
	}
	return rule, nil
}

// mapAlertRuleToModel copies back only what the API owns. The optional inputs
// keep their configured value, so an unset field does not come back as a diff.
func mapAlertRuleToModel(out *client.AlertRule, m *alertRuleModel) {
	m.ID = types.Int64Value(out.ID)
	m.Name = types.StringValue(out.Name)
	m.Operator = types.StringValue(out.Operator)
	m.Threshold = types.Float64Value(out.Threshold)
	m.Duration = types.Int64Value(int64(out.Duration))
	m.Enabled = types.BoolValue(out.Enabled)
	m.NotifyWarning = types.BoolValue(out.NotifyWarning)
	m.NotifyCritical = types.BoolValue(out.NotifyCritical)
	if out.Aggregation != "" {
		m.Aggregation = types.StringValue(out.Aggregation)
	}
	if out.Severity != "" {
		m.Severity = types.StringValue(out.Severity)
	}
	if out.TargetType != "" {
		m.TargetType = types.StringValue(out.TargetType)
	}
}

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := plan.toAPI(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert rule", err.Error())
		return
	}
	out, err := r.client.CreateAlertRule(body)
	if err != nil {
		resp.Diagnostics.AddError("Create alert rule failed", err.Error())
		return
	}
	mapAlertRuleToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetAlertRule(state.ID.ValueInt64())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read alert rule failed", err.Error())
		return
	}
	mapAlertRuleToModel(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state alertRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := plan.toAPI(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert rule", err.Error())
		return
	}
	out, err := r.client.UpdateAlertRule(state.ID.ValueInt64(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update alert rule failed", err.Error())
		return
	}
	mapAlertRuleToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlertRule(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete alert rule failed", err.Error())
	}
}

func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric alert rule id")
		return
	}
	out, err := r.client.GetAlertRule(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}
	var m alertRuleModel
	mapAlertRuleToModel(out, &m)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
