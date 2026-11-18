// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	data_sources "github.com/nexaa-cloud/terraform-provider-nexaa/internal/data-sources"
	"github.com/nexaa-cloud/terraform-provider-nexaa/internal/resources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
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
	ApiKey   types.String `tfsdk:"apikey"`
}

func (p *NexaaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "nexaa"
	resp.Version = p.version
}

func (p *NexaaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{

		MarkdownDescription: `
This is the official provider for Nexaa, an application platform build for developers. We as Nexaa believe that you should focus
on delivering and not on managing your infrastructure. That is why we made it easy for every developer to deploy their applications on Nexaa.
`,
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "The username used to log in the API account",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The password used to log in the API account",
			},
			"apikey": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The API Key used to execute API requests",
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

	if conf.ApiKey.IsNull() || conf.ApiKey.IsUnknown() {
		if (conf.Password.IsUnknown() || conf.Password.IsNull()) &&
			(conf.Username.IsNull() || conf.Username.IsUnknown()) {
			resp.Diagnostics.AddError(
				"Missing Credentials",
				"Error: Missing Username and/or Password. Or missing API key",
			)
		}

		resp.Diagnostics.AddWarning(
			"Deprecated Login",
			"The username password login is deprecated and will be removed in the future. \nAt the moment it can still be used, we recommend switching to an API key instead.",
		)

		p.CredentialLogin(ctx, req, resp)
		return
	}

	p.ApiKeyLogin(ctx, req, resp)
}

func (p *NexaaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewNamespaceResource,
		resources.NewVolumeResource,
		resources.NewRegistryResource,
		resources.NewContainerResource,
		resources.NewStarterContainerResource,
		resources.NewContainerJobResource,
		resources.NewCloudDatabaseClusterResource,
		resources.NewCloudDatabaseClusterDatabaseResource,
		resources.NewDatabaseUserResource,
		resources.NewMessageQueueResource,
	}
}

func (p *NexaaProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *NexaaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		data_sources.NewCloudDatabaseClusterPlans,
		data_sources.NewContainerResources,
		data_sources.NewMessageQueuePlans,
	}
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
