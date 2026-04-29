// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterResource{}
	_ resource.ResourceWithIdentity    = &cloudDatabaseClusterResource{}
)

func NewCloudDatabaseClusterResource() resource.Resource {
	return &cloudDatabaseClusterResource{}
}

type cloudDatabaseClusterResource struct {
	ID                 types.String   `tfsdk:"id"`
	Cluster            ClusterRef     `tfsdk:"cluster"`
	Spec               Spec           `tfsdk:"spec"`
	Plan               types.String   `tfsdk:"plan"`
	Hostname           types.String   `tfsdk:"hostname"`
	ExternalConnection types.Object   `tfsdk:"external_connection"`
	State              types.String   `tfsdk:"state"`
	LastUpdated        types.String   `tfsdk:"last_updated"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

type cloudDatabaseClusterExternalConnectionResource struct {
	Ipv6  types.String `tfsdk:"ipv6"`
	Ipv4  types.String `tfsdk:"ipv4"`
	Ports types.Object `tfsdk:"ports"`
}

type cloudDatabaseClusterExternalConnectionPortsResource struct {
	ExternalPort types.Int64 `tfsdk:"external_port"`
	Allowlist    types.List  `tfsdk:"allowlist"`
}

func (r *cloudDatabaseClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_database_cluster"
}

func (r *cloudDatabaseClusterResource) IdentitySchema(ctx context.Context, request resource.IdentitySchemaRequest, response *resource.IdentitySchemaResponse) {
	response.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"name": identityschema.StringAttribute{
				Description:       "The name of the cloud database cluster.",
				RequiredForImport: true,
			},
			"namespace": identityschema.StringAttribute{
				Description:       "The namespace where the cloud database cluster belongs to.",
				RequiredForImport: true,
			},
		},
	}
}

func (r *cloudDatabaseClusterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cloud Database Cluster resource representing a managed database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the cloud database cluster",
			},
			"cluster": schema.ObjectAttribute{
				Required:       true,
				Description:    "Cloud database cluster",
				CustomType:     NewClusterRefType(),
				AttributeTypes: ClusterRefAttributes(),
			},
			"spec": schema.ObjectAttribute{
				Required:       true,
				Description:    "Database specification including type and version",
				CustomType:     NewSpecType(),
				AttributeTypes: SpecAttributes(),
			},
			"plan": schema.StringAttribute{
				Required:    true,
				Description: "Plan for the cloud database cluster.",
			},
			"hostname": schema.StringAttribute{
				Computed:    true,
				Description: "Hostname of the cloud database cluster",
			},
			"external_connection": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"ipv4": schema.StringAttribute{
						Computed:    true,
						Description: "The ipv4 address that can be used in combination with the external port to connect to your cluster",
					},
					"ipv6": schema.StringAttribute{
						Computed:    true,
						Description: "The ipv6 address that can be used in combination with the external port to connect to your cluster",
					},
					"ports": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"external_port": schema.Int64Attribute{
								Computed:    true,
								Description: "The port that is used in combination with your ipv4 or ipv6 address to connect to your database cluster",
							},
							"allowlist": schema.ListAttribute{
								ElementType: types.StringType,
								Optional:    true,
								Computed:    true,
								Description: "A list with the IP's that can access the database cluster through the external connection, can be in ipv4 and/or ipv6 format. Defaults to 0.0.0.0/0 and ::/0, which means that the database cluster can be accessed from any IP address.",
								Default: listdefault.StaticValue(
									types.ListValueMust(types.StringType, []attr.Value{
										types.StringValue("0.0.0.0/0"),
										types.StringValue("::/0"),
									}),
								),
								PlanModifiers: []planmodifier.List{
									listplanmodifier.UseStateForUnknown(),
								},
								Validators: []validator.List{
									noEmptyAllowlistValidator{},
								},
							},
						},
						Required:    true,
						Description: "Used to define the connection parts of the external connection",
					},
				},
				Optional:    true,
				Description: "An external connection that can used to connect to a cloud database cluster",
			},
			"state": schema.StringAttribute{
				Description: "Current state of the cloud database cluster",
				Computed:    true,
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the cloud database cluster",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *cloudDatabaseClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudDatabaseClusterResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	client := api.NewClient()

	input := api.CloudDatabaseClusterCreateInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
		Spec: api.CloudDatabaseClusterSpecInput{
			Type:    plan.Spec.Type.ValueString(),
			Version: plan.Spec.Version.ValueString(),
		},
		ExternalConnection: buildExternalConnectionInputCloudDb(ctx, plan, nil),
		Plan:               plan.Plan.ValueString(),
		Databases:          []api.DatabaseInput{},
		Users:              []api.DatabaseUserInput{},
	}

	_, err := client.CloudDatabaseClusterCreate(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating cluster",
			"Could not create cluster: "+err.Error(),
		)
		return
	}

	err = waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", "Could not reach a unlocked state: "+err.Error())
	}

	cluster, err := client.CloudDatabaseClusterGet(api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster, cluster not found",
			err.Error(),
		)
		return
	}

	plan, diags = translateApiToCloudDatabaseClusterResource(ctx, cluster, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//Set identity
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      plan.Cluster.Name,
		Namespace: plan.Cluster.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *cloudDatabaseClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	var plan cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}
	cluster, err := client.CloudDatabaseClusterGet(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster, cluster not found",
			err.Error(),
		)
		return
	}

	plan, diags = translateApiToCloudDatabaseClusterResource(ctx, cluster, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//Set identity
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      plan.Cluster.Name,
		Namespace: plan.Cluster.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

// Omitting is not fully supported for this resource. So we write the current state back unchanged and only change the external connection.
func (r *cloudDatabaseClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No in-place updates supported; preserve current plan.
	var plan cloudDatabaseClusterResource
	diags := req.Plan.Get(ctx, &plan)
	var state cloudDatabaseClusterResource
	diags = req.State.Get(ctx, &state)

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	client := api.NewClient()

	//Set up the modify input

	input := api.CloudDatabaseClusterModifyInput{
		Name:               plan.Cluster.Name.ValueString(),
		Namespace:          plan.Cluster.Namespace.ValueString(),
		ExternalConnection: buildExternalConnectionInputCloudDb(ctx, plan, &state),
	}

	_, err := client.CloudDatabaseClusterModify(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating cluster",
			"Could not update cluster: "+err.Error(),
		)
		return
	}

	err = waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating cluster", "Could not reach a unlocked state: "+err.Error())
		return
	}

	cluster, err := client.CloudDatabaseClusterGet(api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading cluster after update",
			err.Error(),
		)
		return
	}

	plan, diags = translateApiToCloudDatabaseClusterResource(ctx, cluster, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//Set identity
	identity := struct {
		Name      types.String `tfsdk:"name"`
		Namespace types.String `tfsdk:"namespace"`
	}{
		Name:      plan.Cluster.Name,
		Namespace: plan.Cluster.Namespace,
	}
	resp.Diagnostics.Append(resp.Identity.Set(ctx, identity)...)
}

func (r *cloudDatabaseClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, diags := plan.Timeouts.Delete(ctx, 2*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	client := api.NewClient()
	err := waitForUnlocked(ctx, cloudDatabaseClusterLocked(), *client, plan.Cluster.Namespace.ValueString(), plan.Cluster.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting cluster", "Could not reach a unlocked state: "+err.Error())
		return
	}

	input := api.CloudDatabaseClusterResourceInput{
		Name:      plan.Cluster.Name.ValueString(),
		Namespace: plan.Cluster.Namespace.ValueString(),
	}
	_, err = client.CloudDatabaseClusterDelete(input)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cluster",
			"Could not delete cluster: "+err.Error(),
		)
		return
	}

}

func (r *cloudDatabaseClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the format \"<namespace>/<cluster_name>\", got: "+req.ID,
		)
		return
	}
	namespace := parts[0]
	clusterName := parts[1]

	client := api.NewClient()
	clusterResourceInput := api.CloudDatabaseClusterResourceInput{
		Namespace: namespace,
		Name:      clusterName,
	}
	cluster, err := client.CloudDatabaseClusterGet(clusterResourceInput)
	if err != nil {
		resp.Diagnostics.AddError("Error importing database", "Could not list clusters: "+err.Error())
		return
	}

	var plan cloudDatabaseClusterResource

	plan.Timeouts = timeouts.Value{
		Object: types.ObjectValueMust(
			map[string]attr.Type{
				"create": types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
			map[string]attr.Value{
				"create": types.StringValue("2m"),
				"update": types.StringValue("2m"),
				"delete": types.StringValue("2m"),
			},
		),
	}

	plan, diags := translateApiToCloudDatabaseClusterResource(ctx, cluster, plan.Timeouts)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}
