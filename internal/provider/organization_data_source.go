package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/middle-monitor/terraform-provider-middmonitor/internal/client"
)

var _ datasource.DataSource = &organizationDataSource{}

type organizationDataSource struct {
	client *client.Client
}

func NewOrganizationDataSource() datasource.DataSource {
	return &organizationDataSource{}
}

func (d *organizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *organizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	rd, ok := req.ProviderData.(*resourceData)
	if !ok {
		resp.Diagnostics.AddError("Internal error", "invalid provider data")
		return
	}
	d.client = rd.Client
}

func (d *organizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads the current organization (from JWT + `org_slug` in the provider).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"slug": schema.StringAttribute{
				Computed: true,
			},
			"plan": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

type orgDataModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Slug types.String `tfsdk:"slug"`
	Plan types.String `tfsdk:"plan"`
}

func (d *organizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	o, err := d.client.GetOrganization()
	if err != nil {
		resp.Diagnostics.AddError("Read organization failed", err.Error())
		return
	}
	var m orgDataModel
	m.ID = types.Int64Value(o.ID)
	m.Name = types.StringValue(o.Name)
	m.Slug = types.StringValue(o.Slug)
	m.Plan = types.StringValue(o.Plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
}
