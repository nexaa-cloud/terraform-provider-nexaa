// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterUserResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterUserResource{}
)

func NewDatabaseUserResource() resource.Resource {
	return &cloudDatabaseClusterUserResource{}
}

type cloudDatabaseClusterUserResource struct {
	ID          types.String   `tfsdk:"id"`
	Cluster     ClusterRef     `tfsdk:"cluster"`
	Name        types.String   `tfsdk:"name"`
	Password    types.String   `tfsdk:"password"`
	Permissions PermissionType `tfsdk:"permissions"`
	LastUpdated types.String   `tfsdk:"last_updated"`
	timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func (r *cloudDatabaseClusterUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_database_cluster_user"
}

func (r *cloudDatabaseClusterUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Database User resource representing a database user within a cloud database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the database user",
			},
			"cluster": schema.ObjectAttribute{
				Required:       true,
				Description:    "Cloud database cluster this database belongs to.",
				CustomType:     NewClusterRefType(),
				AttributeTypes: ClusterRefAttributes(),
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the database user",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password for the database user",
			},
			"permissions": schema.ListAttribute{
				CustomType:  NewPermissionListType(),
				Optional:    true,
				Computed:    true,
				Description: "Permissions for the database user",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the database user",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *cloudDatabaseClusterUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudDatabaseClusterDatabaseResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

}

func (r *cloudDatabaseClusterUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

}

func (r *cloudDatabaseClusterUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

}

func (r *cloudDatabaseClusterUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {

}

func (r *cloudDatabaseClusterUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
}
