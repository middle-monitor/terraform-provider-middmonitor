package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// schemaOf builds one resource's schema the way Terraform does.
func schemaOf(t *testing.T, r resource.Resource) fwschema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func typeNameOf(t *testing.T, r resource.Resource) string {
	t.Helper()
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "middmonitor"}, resp)
	return resp.TypeName
}

// Every resource the provider advertises must have a schema Terraform accepts.
// A malformed one is a panic at plan time on the user's machine, not here.
func TestEveryResourceSchemaIsValid(t *testing.T) {
	for _, factory := range (&middmonitorProvider{}).Resources(context.Background()) {
		r := factory()
		name := typeNameOf(t, r)
		schema := schemaOf(t, r)
		if diags := schema.ValidateImplementation(context.Background()); diags.HasError() {
			t.Fatalf("%s: %v", name, diags)
		}
		if _, hasID := schema.Attributes["id"]; !hasID {
			t.Fatalf("%s has no id attribute, so it cannot be imported", name)
		}
	}
}

// The resources the migration needs have to be registered, or a configuration
// referring to them fails with "invalid resource type".
func TestTheProviderRegistersEveryResource(t *testing.T) {
	registered := map[string]bool{}
	for _, factory := range (&middmonitorProvider{}).Resources(context.Background()) {
		registered[typeNameOf(t, factory())] = true
	}

	for _, want := range []string{
		"middmonitor_host",
		"middmonitor_host_group",
		"middmonitor_service",
		"middmonitor_install_token",
		"middmonitor_alert_rule",
		"middmonitor_notification_channel",
		"middmonitor_maintenance_window",
	} {
		if !registered[want] {
			t.Fatalf("%s is not registered: %v", want, registered)
		}
	}
}

// Every resource must be importable, or an existing installation cannot be
// adopted into Terraform without recreating it.
func TestEveryResourceSupportsImport(t *testing.T) {
	for _, factory := range (&middmonitorProvider{}).Resources(context.Background()) {
		r := factory()
		if _, ok := r.(resource.ResourceWithImportState); !ok {
			t.Fatalf("%s does not implement ImportState", typeNameOf(t, r))
		}
	}
}

// The tfsdk model and the schema are matched by name at runtime; a mismatch is
// an error the user sees, so it is checked here instead.
func TestModelsMatchTheirSchema(t *testing.T) {
	cases := []struct {
		resource resource.Resource
		model    any
	}{
		{NewHostResource(), hostModel{}},
		{NewHostGroupResource(), hostGroupModel{}},
		{NewServiceResource(), serviceModel{}},
		{NewAlertRuleResource(), alertRuleModel{}},
		{NewNotificationChannelResource(), notificationChannelModel{}},
		{NewMaintenanceWindowResource(), maintenanceWindowModel{}},
	}

	for _, c := range cases {
		name := typeNameOf(t, c.resource)
		schema := schemaOf(t, c.resource)

		fields := map[string]bool{}
		value := reflect.TypeOf(c.model)
		for i := 0; i < value.NumField(); i++ {
			tag := strings.Split(value.Field(i).Tag.Get("tfsdk"), ",")[0]
			if tag != "" {
				fields[tag] = true
			}
		}

		for attribute := range schema.Attributes {
			if !fields[attribute] {
				t.Fatalf("%s: schema declares %q with no matching model field", name, attribute)
			}
		}
		for field := range fields {
			if _, declared := schema.Attributes[field]; !declared {
				t.Fatalf("%s: model has %q with no matching schema attribute", name, field)
			}
		}
	}
}

// The two-level thresholds are what an operator sets on a check; only the
// legacy single one used to be reachable from Terraform.
func TestServiceExposesBothThresholds(t *testing.T) {
	schema := schemaOf(t, NewServiceResource())
	for _, attribute := range []string{"failure_threshold", "warning_threshold", "critical_threshold"} {
		if _, ok := schema.Attributes[attribute]; !ok {
			t.Fatalf("middmonitor_service has no %s", attribute)
		}
	}
}

// Alerting on a scraped series goes through custom_metric; a rule limited to
// the built-in signals cannot watch anything the agent pulls.
func TestAlertRuleReachesCustomMetrics(t *testing.T) {
	schema := schemaOf(t, NewAlertRuleResource())
	for _, attribute := range []string{"custom_metric", "custom_labels", "warning_threshold", "critical_threshold", "channels"} {
		if _, ok := schema.Attributes[attribute]; !ok {
			t.Fatalf("middmonitor_alert_rule has no %s", attribute)
		}
	}
}
