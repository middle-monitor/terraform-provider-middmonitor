package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ provider.Provider = (*middmonitorProvider)(nil)

type middmonitorProvider struct {
	version string
}

func New() provider.Provider {
	return &middmonitorProvider{
		version: "0.1.0",
	}
}

func (p *middmonitorProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "middmonitor"
	resp.Version = p.version
}

func (p *middmonitorProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Configure access to the [Middle Monitor](https://github.com/middle-monitor) dashboard API (JWT).",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the **dashboard API** (e.g. `https://monitor.example.com`), without trailing slash. Falls back to `MIDDLE_MONITOR_BASE_URL`.",
				Optional:            true,
			},
			"org_slug": schema.StringAttribute{
				MarkdownDescription: "Organization slug (URL segment), e.g. `default`. Falls back to `MIDDLE_MONITOR_ORG_SLUG`.",
				Optional:            true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "JWT access token from `POST /api/v1/auth/login`. Falls back to `MIDDLE_MONITOR_ACCESS_TOKEN`.",
				Optional:            true,
				Sensitive:           true,
			},
			"receiver_base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the **receiver** service (agent downloads, install script). Defaults to `base_url` if empty (monolithic deploy).",
				Optional:            true,
			},
		},
	}
}

type providerModel struct {
	BaseURL         types.String `tfsdk:"base_url"`
	OrgSlug         types.String `tfsdk:"org_slug"`
	AccessToken     types.String `tfsdk:"access_token"`
	ReceiverBaseURL types.String `tfsdk:"receiver_base_url"`
}

func (p *middmonitorProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	base := cfg.BaseURL.ValueString()
	if base == "" {
		base = os.Getenv("MIDDLE_MONITOR_BASE_URL")
	}
	token := cfg.AccessToken.ValueString()
	if token == "" {
		token = os.Getenv("MIDDLE_MONITOR_ACCESS_TOKEN")
	}
	org := cfg.OrgSlug.ValueString()
	if org == "" {
		org = os.Getenv("MIDDLE_MONITOR_ORG_SLUG")
	}

	if base == "" || token == "" || org == "" {
		resp.Diagnostics.AddError("Configuration error", "base_url, org_slug and access_token are required (or set MIDDLE_MONITOR_* env vars).")
		return
	}

	c := client.New(base, token, org)

	// Receiver URL for data sources (defaults to base_url for monolithic deploys).
	recv := base
	if !cfg.ReceiverBaseURL.IsNull() && cfg.ReceiverBaseURL.ValueString() != "" {
		recv = cfg.ReceiverBaseURL.ValueString()
	}
	resp.ResourceData = &resourceData{Client: c, ReceiverBaseURL: recv}
	resp.DataSourceData = &resourceData{Client: c, ReceiverBaseURL: recv}
}

// resourceData carries client + receiver URL for resources and data sources.
type resourceData struct {
	Client          *client.Client
	ReceiverBaseURL string
}

func (p *middmonitorProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHostResource,
		NewServiceResource,
		NewInstallTokenResource,
	}
}

func (p *middmonitorProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewAgentInstallDataSource,
	}
}
