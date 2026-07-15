package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &agentInstallDataSource{}

type agentInstallDataSource struct {
	receiverBase string
}

func NewAgentInstallDataSource() datasource.DataSource {
	return &agentInstallDataSource{}
}

func (d *agentInstallDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_install"
}

func (d *agentInstallDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	rd, ok := req.ProviderData.(*resourceData)
	if !ok {
		resp.Diagnostics.AddError("Internal error", "invalid provider data")
		return
	}
	d.receiverBase = strings.TrimRight(rd.ReceiverBaseURL, "/")
}

func (d *agentInstallDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Builds **URLs and shell snippets** to install the Middle Monitor agent using an install token. Does not run commands on remote machines — pass `cloud_init` or `remote-exec` yourself.",
		Attributes: map[string]schema.Attribute{
			"install_token": schema.StringAttribute{
				MarkdownDescription: "Install token (from `middmonitor_install_token` or UI).",
				Required:            true,
				Sensitive:           true,
			},
			"os": schema.StringAttribute{
				MarkdownDescription: "Agent OS for binary URL: `linux` or `darwin`.",
				Optional:            true,
			},
			"arch": schema.StringAttribute{
				MarkdownDescription: "Agent arch: `amd64` or `arm64`.",
				Optional:            true,
			},
			"install_script_url": schema.StringAttribute{
				MarkdownDescription: "GET URL returning the install shell script (token injected).",
				Computed:            true,
			},
			"agent_binary_url": schema.StringAttribute{
				MarkdownDescription: "Direct download URL for the agent binary.",
				Computed:            true,
			},
			"curl_install_command": schema.StringAttribute{
				MarkdownDescription: "One-liner to download and run the install script with `curl`.",
				Computed:            true,
			},
			"export_env_snippet": schema.StringAttribute{
				MarkdownDescription: "Shell snippet setting `MIDDLE_MONITOR_API_URL` and token for manual installs.",
				Computed:            true,
			},
		},
	}
}

type agentInstallModel struct {
	InstallToken       types.String `tfsdk:"install_token"`
	OS                 types.String `tfsdk:"os"`
	Arch               types.String `tfsdk:"arch"`
	InstallScriptURL   types.String `tfsdk:"install_script_url"`
	AgentBinaryURL     types.String `tfsdk:"agent_binary_url"`
	CurlInstallCommand types.String `tfsdk:"curl_install_command"`
	ExportEnvSnippet   types.String `tfsdk:"export_env_snippet"`
}

func (d *agentInstallDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg agentInstallModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	osName := "linux"
	if !cfg.OS.IsNull() && cfg.OS.ValueString() != "" {
		osName = cfg.OS.ValueString()
	}
	arch := "amd64"
	if !cfg.Arch.IsNull() && cfg.Arch.ValueString() != "" {
		arch = cfg.Arch.ValueString()
	}

	token := cfg.InstallToken.ValueString()
	scriptURL := fmt.Sprintf("%s/api/v1/agents/download/install?token=%s", d.receiverBase, url.QueryEscape(token))
	binURL := fmt.Sprintf("%s/api/v1/agents/download/%s/%s", d.receiverBase, osName, arch)

	curl := fmt.Sprintf("curl -fsSL %q | sudo bash", scriptURL)

	env := fmt.Sprintf(`export MIDDLE_MONITOR_API_URL=%q
export X_INSTALL_TOKEN=%q
# Or pass ?token= to the install script URL (already embedded in install_script_url).
`, d.receiverBase, token)

	cfg.InstallScriptURL = types.StringValue(scriptURL)
	cfg.AgentBinaryURL = types.StringValue(binURL)
	cfg.CurlInstallCommand = types.StringValue(curl)
	cfg.ExportEnvSnippet = types.StringValue(env)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
