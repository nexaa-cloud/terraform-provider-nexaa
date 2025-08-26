// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/resources"

	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/nexaa-cloud/nexaa-cli/config"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure NexaaProvider satisfies various provider interfaces.
var _ provider.Provider = &NexaaProvider{}
var _ provider.ProviderWithFunctions = &NexaaProvider{}
var _ provider.ProviderWithEphemeralResources = &NexaaProvider{}

// NexaaProvider defines the provider implementation.
type NexaaProvider struct {
	version string
}

// NexaaProviderModel describes the provider data model.
type NexaaProviderModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *NexaaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nexaa"
	resp.Version = p.version
}

func (p *NexaaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username used to log in the API account",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The password used to log in the API account",
			},
		},
	}
}

func (p *NexaaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var conf NexaaProviderModel
	diags := req.Config.Get(ctx, &conf)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if conf.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown username",
			"Missing username for authentication",
		)
	}

	if conf.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown password",
			"Missing password for authentication",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	username := os.Getenv("NEXAA_USERNAME")
	password := os.Getenv("NEXAA_PASSWORD")

	if !conf.Username.IsNull() {
		username = conf.Username.ValueString()
	}

	if !conf.Password.IsNull() {
		password = conf.Password.ValueString()
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown username",
			"Missing username for authentication",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown password",
			"Missing password for authentication",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	config.Initialize()

	if err := config.LoadConfig(); err != nil {
		panic(err)
	}

	err := api.Login(username, password)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to log in",
			"Error: "+err.Error(),
		)
		return
	}

	// Create API client and make it available to resources
	client := api.NewClient()
	resp.ResourceData = client
}

func (p *NexaaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewNamespaceResource,
		resources.NewVolumeResource,
		resources.NewRegistryResource,
		resources.NewContainerResource,
		resources.NewContainerJobResource,
		resources.NewCloudDatabaseClusterResource,
		resources.NewDatabaseResource,
		resources.NewDatabaseUserResource,
	}
}

func (p *NexaaProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *NexaaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *NexaaProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &NexaaProvider{
			version: version,
		}
	}
}
