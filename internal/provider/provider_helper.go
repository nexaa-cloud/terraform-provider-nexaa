// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/nexaa-cloud/nexaa-cli/api"
	"github.com/nexaa-cloud/nexaa-cli/config"
	nexaaclient "github.com/nexaa-cloud/terraform-provider-nexaa/internal/client"
)

func (p *NexaaProvider) CredentialLogin(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
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
		resp.Diagnostics.AddError(
			"Unable to load Nexaa configuration",
			"Error: "+err.Error(),
		)
		return
	}

	err := api.Login(username, password)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to log in",
			"Error: "+err.Error(),
		)
		return
	}

	// Create API client and make it available to resources.
	// NexaaClient wraps the client with a shared MutexKV that serializes
	// concurrent Create calls for the same resource name.
	client := nexaaclient.New(api.NewClient())
	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *NexaaProvider) ApiKeyLogin(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var conf NexaaProviderModel
	diags := req.Config.Get(ctx, &conf)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if conf.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Unknown apikey",
			"Missing apikey for authentication",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("NEXAA_APIKEY")

	if !conf.ApiKey.IsNull() {
		apiKey = conf.ApiKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("apikey"),
			"Unknown apikey",
			"Missing apikey for authentication",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	config.Initialize()

	if err := config.LoadApiConfig(); err != nil {
		resp.Diagnostics.AddError(
			"Unable to load Nexaa configuration",
			"Error: "+err.Error(),
		)
		return
	}

	err := api.LoginApiKey(apiKey)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to log in",
			"Error: "+err.Error(),
		)
		return
	}

	// Create API client and make it available to resources.
	// NexaaClient wraps the client with a shared MutexKV that serializes
	// concurrent Create calls for the same resource name.
	client := nexaaclient.New(api.BuildClient())
	resp.ResourceData = client
	resp.DataSourceData = client
}
