package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ resource.Resource = &notificationChannelResource{}
var _ resource.ResourceWithImportState = &notificationChannelResource{}

type notificationChannelResource struct {
	client *client.Client
}

func NewNotificationChannelResource() resource.Resource {
	return &notificationChannelResource{}
}

func (r *notificationChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_channel"
}

func (r *notificationChannelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *notificationChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A destination for alerts: email, Slack, a generic webhook, Jira Service Management or WhatsApp.\n\n" +
			"`config` is a JSON document whose keys depend on `type`. For a webhook: `webhook_url`, " +
			"`secret` (HMAC key), `format` (`slack` by default, `structured` for the typed payload), " +
			"`headers`, and the anti-burst settings `group_by`, `group_wait`, `repeat_interval`. " +
			"It is a JSON string rather than a typed block because the accepted keys differ per channel type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "Channel ID.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Channel name.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "One of `email`, `slack`, `webhook`, `jsm`, `whatsapp`.",
				Required:            true,
			},
			"config": schema.StringAttribute{
				MarkdownDescription: "Channel configuration, as a JSON object. Use `jsonencode({...})`.",
				Required:            true,
				Sensitive:           true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether alerts are delivered here. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

type notificationChannelModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
	Config  types.String `tfsdk:"config"`
	Enabled types.Bool   `tfsdk:"enabled"`
}

func (m notificationChannelModel) toAPI() (client.NotificationChannel, error) {
	channel := client.NotificationChannel{
		Name:    m.Name.ValueString(),
		Type:    m.Type.ValueString(),
		Enabled: m.Enabled.ValueBool(),
		Config:  map[string]any{},
	}
	if raw := m.Config.ValueString(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &channel.Config); err != nil {
			return channel, err
		}
	}
	return channel, nil
}

// mapChannelToModel keeps the configured JSON string rather than re-encoding
// what the API returned: a secret the API redacts would otherwise show up as a
// permanent diff.
func mapChannelToModel(out *client.NotificationChannel, m *notificationChannelModel) {
	m.ID = types.Int64Value(out.ID)
	m.Name = types.StringValue(out.Name)
	m.Type = types.StringValue(out.Type)
	m.Enabled = types.BoolValue(out.Enabled)
}

func (r *notificationChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := plan.toAPI()
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", "config must be a JSON object: "+err.Error())
		return
	}
	out, err := r.client.CreateNotificationChannel(body)
	if err != nil {
		resp.Diagnostics.AddError("Create notification channel failed", err.Error())
		return
	}
	mapChannelToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetNotificationChannel(state.ID.ValueInt64())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Read notification channel failed", err.Error())
		return
	}
	mapChannelToModel(out, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *notificationChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state notificationChannelModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, err := plan.toAPI()
	if err != nil {
		resp.Diagnostics.AddError("Invalid config", "config must be a JSON object: "+err.Error())
		return
	}
	out, err := r.client.UpdateNotificationChannel(state.ID.ValueInt64(), body)
	if err != nil {
		resp.Diagnostics.AddError("Update notification channel failed", err.Error())
		return
	}
	mapChannelToModel(out, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *notificationChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state notificationChannelModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteNotificationChannel(state.ID.ValueInt64()); err != nil {
		resp.Diagnostics.AddError("Delete notification channel failed", err.Error())
	}
}

func (r *notificationChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", "Expected numeric channel id")
		return
	}
	out, err := r.client.GetNotificationChannel(id)
	if err != nil {
		resp.Diagnostics.AddError("Import read failed", err.Error())
		return
	}

	var m notificationChannelModel
	mapChannelToModel(out, &m)
	// The config is not returned verbatim by the API (secrets are redacted), so
	// an import carries what it can and the operator writes the block.
	encoded, err := json.Marshal(out.Config)
	if err == nil {
		m.Config = types.StringValue(string(encoded))
	} else {
		m.Config = types.StringValue("{}")
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
