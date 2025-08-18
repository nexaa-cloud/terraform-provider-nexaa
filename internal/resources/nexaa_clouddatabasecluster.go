// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nexaa-cloud/nexaa-cli/api"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &cloudDatabaseClusterResource{}
	_ resource.ResourceWithImportState = &cloudDatabaseClusterResource{}
)

func NewCloudDatabaseClusterResource() resource.Resource {
	return &cloudDatabaseClusterResource{}
}

type cloudDatabaseClusterResource struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Namespace   types.String `tfsdk:"namespace"`
	Spec        types.Object `tfsdk:"spec"`
	Plan        types.Object `tfsdk:"plan"`
	Databases   types.List   `tfsdk:"databases"`
	Users       types.List   `tfsdk:"users"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type specResource struct {
	Type    types.String `tfsdk:"type"`
	Version types.String `tfsdk:"version"`
}

type databaseResource struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	State       types.String `tfsdk:"state"`
}

type databaseUserResource struct {
	Name        types.String `tfsdk:"name"`
	Password    types.String `tfsdk:"password"`
	State       types.String `tfsdk:"state"`
	Permissions types.List   `tfsdk:"permissions"`
}

type databaseUserPermissionResource struct {
	Database   types.String `tfsdk:"database"`
	Permission types.String `tfsdk:"permission"`
}

type planResource struct {
	// User-specified inputs for plan selection
	Cpu      types.Int64   `tfsdk:"cpu"`
	Memory   types.Float64 `tfsdk:"memory"`
	Storage  types.Int64   `tfsdk:"storage"`
	Replicas types.Int64   `tfsdk:"replicas"`
	
	// Computed fields
	ID       types.String  `tfsdk:"id"`
	Name     types.String  `tfsdk:"name"`
	Group    types.String  `tfsdk:"group"` // Computed: the actual group name from API
	Price    types.Object  `tfsdk:"price"`
}

type planPriceResource struct {
	Amount   types.Int64  `tfsdk:"amount"`
	Currency types.String `tfsdk:"currency"`
}

// replicasToGroup maps replica count to API group names
func replicasToGroup(replicas int64) string {
	switch replicas {
	case 1:
		return "Single (1 node)"
	case 2:
		return "Redundant (2 nodes)"
	case 3:
		return "Highly available (3 nodes)"
	default:
		return "Single (1 node)" // fallback
	}
}

// findMatchingPlan finds a plan that matches the user's specifications
func findMatchingPlan(client *api.Client, specs planResource) (*api.CloudDatabaseClusterPlan, error) {
	plans, err := client.CloudDatabaseClusterListPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to list available plans: %w", err)
	}

	requiredCpu := int(specs.Cpu.ValueInt64())
	requiredMemory := specs.Memory.ValueFloat64()
	requiredStorage := int(specs.Storage.ValueInt64())
	requiredReplicas := specs.Replicas.ValueInt64()
	requiredGroup := replicasToGroup(requiredReplicas)

	// Find exact matches first
	for _, plan := range plans {
		if plan.Cpu == requiredCpu &&
			plan.Memory == requiredMemory &&
			plan.Storage == requiredStorage &&
			plan.Group == requiredGroup {
			// TODO: Add replica matching once available in API
			return &plan, nil
		}
	}

	// If no exact match, find the smallest plan that meets or exceeds requirements
	var bestPlan *api.CloudDatabaseClusterPlan
	for _, plan := range plans {
		if plan.Cpu >= requiredCpu &&
			plan.Memory >= requiredMemory &&
			plan.Storage >= requiredStorage &&
			plan.Group == requiredGroup {
			if bestPlan == nil ||
				plan.Cpu < bestPlan.Cpu ||
				(plan.Cpu == bestPlan.Cpu && plan.Memory < bestPlan.Memory) ||
				(plan.Cpu == bestPlan.Cpu && plan.Memory == bestPlan.Memory && plan.Storage < bestPlan.Storage) {
				bestPlan = &plan
			}
		}
	}

	if bestPlan == nil {
		// Create a helpful error message with available plans
		var availablePlans []string
		for _, plan := range plans {
			availablePlans = append(availablePlans, fmt.Sprintf("cpu=%d, memory=%.1f, storage=%d, group=%s", 
				plan.Cpu, plan.Memory, plan.Storage, plan.Group))
		}
		return nil, fmt.Errorf("no plan found matching requirements: cpu=%d, memory=%.1f, storage=%d, replicas=%d (group=%s). Available plans: %v", 
			requiredCpu, requiredMemory, requiredStorage, requiredReplicas, requiredGroup, availablePlans)
	}

	return bestPlan, nil
}

func (r *cloudDatabaseClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clouddatabasecluster"
}

func (r *cloudDatabaseClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Cloud Database Cluster resource representing a managed database cluster on Nexaa.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Unique identifier of the cloud database cluster",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the cloud database cluster",
			},
			"namespace": schema.StringAttribute{
				Required:    true,
				Description: "Name of the namespace that the cluster will belong to",
			},
			"spec": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required:    true,
						Description: "Database type (e.g., postgresql, mysql)",
					},
					"version": schema.StringAttribute{
						Required:    true,
						Description: "Database version",
					},
				},
				Required:    true,
				Description: "Database specification including type and version",
			},
			"plan": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// User inputs for plan selection
					"cpu": schema.Int64Attribute{
						Required:    true,
						Description: "Number of CPU cores required",
					},
					"memory": schema.Float64Attribute{
						Required:    true,
						Description: "Memory required in GB",
					},
					"storage": schema.Int64Attribute{
						Required:    true,
						Description: "Storage required in GB",
					},
					"replicas": schema.Int64Attribute{
						Required:    true,
						Description: "Number of replicas/nodes (1 = single node, 2 = redundant, 3 = highly available)",
						Validators: []validator.Int64{
							int64validator.Between(1, 3),
						},
					},
					// Computed fields
					"id": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan ID",
					},
					"name": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan name",
					},
					"group": schema.StringAttribute{
						Computed:    true,
						Description: "Matched plan group (e.g., 'Single (1 node)', 'Redundant (2 nodes)')",
					},
					"price": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"amount": schema.Int64Attribute{
								Computed:    true,
								Description: "Price amount in cents",
							},
							"currency": schema.StringAttribute{
								Computed:    true,
								Description: "Price currency",
							},
						},
						Computed:    true,
						Description: "Matched plan pricing information",
					},
				},
				Required:    true,
				Description: "Database cluster plan specification - provider will find matching plan",
			},
			"databases": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the database",
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Optional description of the database",
						},
						"state": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "State of the database (present/absent)",
							Validators: []validator.String{
								stringvalidator.OneOf("present", "absent"),
							},
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "List of databases to create in the cluster",
			},
			"users": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the database user",
						},
						"password": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "Password for the database user",
						},
						"state": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "State of the user (present/absent)",
							Validators: []validator.String{
								stringvalidator.OneOf("present", "absent"),
							},
						},
						"permissions": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"database": schema.StringAttribute{
										Required:    true,
										Description: "Database name for the permission",
									},
									"permission": schema.StringAttribute{
										Required:    true,
										Description: "Permission type (e.g., read, write, admin)",
									},
								},
							},
							Optional:    true,
							Computed:    true,
							Description: "List of database permissions for the user",
						},
					},
				},
				Optional:    true,
				Computed:    true,
				Description: "List of database users to create in the cluster",
			},
			"last_updated": schema.StringAttribute{
				Description: "Timestamp of the last Terraform update of the cloud database cluster",
				Computed:    true,
			},
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

	var spec specResource
	diags = plan.Spec.As(ctx, &spec, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var planSpecs planResource
	diags = plan.Plan.As(ctx, &planSpecs, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Find matching plan based on user specifications
	client := api.NewClient()
	matchingPlan, err := findMatchingPlan(client, planSpecs)
	if err != nil {
		resp.Diagnostics.AddError("Plan Selection Error", "Could not find a plan matching your specifications: "+err.Error())
		return
	}

	input := api.CloudDatabaseClusterCreateInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
		Spec: api.CloudDatabaseClusterSpecInput{
			Type:    spec.Type.ValueString(),
			Version: spec.Version.ValueString(),
		},
		Plan: matchingPlan.Id,
	}

	if !plan.Databases.IsNull() && !plan.Databases.IsUnknown() {
		var databases []databaseResource
		diags = plan.Databases.ElementsAs(ctx, &databases, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, db := range databases {
			state := api.StatePresent
			if !db.State.IsNull() && !db.State.IsUnknown() {
				state = api.State(db.State.ValueString())
			}

			var description *string
			if !db.Description.IsNull() && !db.Description.IsUnknown() {
				desc := db.Description.ValueString()
				description = &desc
			}

			input.Databases = append(input.Databases, api.DatabaseInput{
				Name:        db.Name.ValueString(),
				Description: description,
				State:       state,
			})
		}
	} else {
		input.Databases = []api.DatabaseInput{}
	}

	if !plan.Users.IsNull() && !plan.Users.IsUnknown() {
		var users []databaseUserResource
		diags = plan.Users.ElementsAs(ctx, &users, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, user := range users {
			state := api.StatePresent
			if !user.State.IsNull() && !user.State.IsUnknown() {
				state = api.State(user.State.ValueString())
			}

			var password *string
			if !user.Password.IsNull() && !user.Password.IsUnknown() {
				pwd := user.Password.ValueString()
				password = &pwd
			}

			userInput := api.DatabaseUserInput{
				Name:     user.Name.ValueString(),
				Password: password,
				State:    state,
			}

			if !user.Permissions.IsNull() && !user.Permissions.IsUnknown() {
				var permissions []databaseUserPermissionResource
				diags = user.Permissions.ElementsAs(ctx, &permissions, false)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				for _, perm := range permissions {
					userInput.Permissions = append(userInput.Permissions, api.DatabaseUserPermissionInput{
						DatabaseName: perm.Database.ValueString(),
						Permission:   api.DatabasePermission(perm.Permission.ValueString()),
						State:        api.StatePresent,
					})
				}
			}

			input.Users = append(input.Users, userInput)
		}
	} else {
		input.Users = []api.DatabaseUserInput{}
	}

	cluster, err := client.CloudDatabaseClusterCreate(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cloud database cluster", "Could not create cluster: "+err.Error())
		return
	}

	plan.ID = types.StringValue(cluster.Id)
	plan.Name = types.StringValue(cluster.Name)
	plan.Namespace = types.StringValue(cluster.Namespace.Name)

	// Set plan object with both user specs and matched plan info
	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Use the user-specified replicas value
	replicas := planSpecs.Replicas

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price":    types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": replicas,
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringValue(matchingPlan.Name), // Use the matched plan's name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)
	plan.Plan = planObj

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	plan.Spec = specObj

	if cluster.Databases != nil {
		databases := make([]attr.Value, len(cluster.Databases))
		for i, db := range cluster.Databases {
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":        types.StringType,
					"description": types.StringType,
					"state":       types.StringType,
				},
				map[string]attr.Value{
					"name":        types.StringValue(db.Name),
					"description": types.StringPointerValue(db.Description),
					"state":       types.StringValue("present"),
				})
			databases[i] = obj
		}
		dbList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":        types.StringType,
				"description": types.StringType,
				"state":       types.StringType,
			},
		}, databases)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Databases = dbList
	}

	if cluster.Users != nil {
		users := make([]attr.Value, len(cluster.Users))
		for i, clusterUser := range cluster.Users {
			permissions := make([]attr.Value, len(clusterUser.Permissions))
			for j, perm := range clusterUser.Permissions {
				permObj := types.ObjectValueMust(
					map[string]attr.Type{
						"database":   types.StringType,
						"permission": types.StringType,
					},
					map[string]attr.Value{
						"database":   types.StringValue(perm.DatabaseName),
						"permission": types.StringValue(string(perm.Permission)),
					})
				permissions[j] = permObj
			}
			permList, diags := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
			}, permissions)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":     types.StringType,
					"password": types.StringType,
					"state":    types.StringType,
					"permissions": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"database":   types.StringType,
						"permission": types.StringType,
					}}},
				},
				map[string]attr.Value{
					"name":        types.StringValue(clusterUser.Name),
					"password":    types.StringNull(),
					"state":       types.StringValue("present"),
					"permissions": permList,
				})
			users[i] = obj
		}
		userList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":     types.StringType,
				"password": types.StringType,
				"state":    types.StringType,
				"permissions": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				}}},
			},
		}, users)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Users = userList
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud database clusters", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == state.Name.ValueString() && c.Namespace.Name == state.Namespace.ValueString() {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = types.StringValue(cluster.Id)
	state.Name = types.StringValue(cluster.Name)
	state.Namespace = types.StringValue(cluster.Namespace.Name)

	// Set plan object for Read - preserve from current state or use API data
	var currentPlan planResource
	if !state.Plan.IsNull() && !state.Plan.IsUnknown() {
		diags = state.Plan.As(ctx, &currentPlan, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Preserve user input values or derive from API group name
	replicas := types.Int64Value(1) // default
	if !currentPlan.Replicas.IsNull() && !currentPlan.Replicas.IsUnknown() {
		replicas = currentPlan.Replicas
	} else {
		// Try to derive replicas from group name if not preserved
		switch cluster.Plan.Group {
		case "Single (1 node)":
			replicas = types.Int64Value(1)
		case "Redundant (2 nodes)":
			replicas = types.Int64Value(2)
		case "Highly available (3 nodes)":
			replicas = types.Int64Value(3)
		}
	}

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price":    types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": replicas,
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringNull(), // Plan result doesn't include name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)
	state.Plan = planObj

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	state.Spec = specObj

	if cluster.Databases != nil {
		databases := make([]attr.Value, len(cluster.Databases))
		for i, db := range cluster.Databases {
			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":        types.StringType,
					"description": types.StringType,
					"state":       types.StringType,
				},
				map[string]attr.Value{
					"name":        types.StringValue(db.Name),
					"description": types.StringPointerValue(db.Description),
					"state":       types.StringValue("present"),
				})
			databases[i] = obj
		}
		dbList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":        types.StringType,
				"description": types.StringType,
				"state":       types.StringType,
			},
		}, databases)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Databases = dbList
	}

	if cluster.Users != nil {
		users := make([]attr.Value, len(cluster.Users))
		for i, user := range cluster.Users {
			permissions := make([]attr.Value, len(user.Permissions))
			for j, perm := range user.Permissions {
				permObj := types.ObjectValueMust(
					map[string]attr.Type{
						"database":   types.StringType,
						"permission": types.StringType,
					},
					map[string]attr.Value{
						"database":   types.StringValue(perm.DatabaseName),
						"permission": types.StringValue(string(perm.Permission)),
					})
				permissions[j] = permObj
			}
			permList, diags := types.ListValue(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				},
			}, permissions)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			obj := types.ObjectValueMust(
				map[string]attr.Type{
					"name":     types.StringType,
					"password": types.StringType,
					"state":    types.StringType,
					"permissions": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
						"database":   types.StringType,
						"permission": types.StringType,
					}}},
				},
				map[string]attr.Value{
					"name":        types.StringValue(user.Name),
					"password":    types.StringNull(),
					"state":       types.StringValue("present"),
					"permissions": permList,
				})
			users[i] = obj
		}
		userList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":     types.StringType,
				"password": types.StringType,
				"state":    types.StringType,
				"permissions": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{
					"database":   types.StringType,
					"permission": types.StringType,
				}}},
			},
		}, users)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Users = userList
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan cloudDatabaseClusterResource
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := api.CloudDatabaseClusterModifyInput{
		Name:      plan.Name.ValueString(),
		Namespace: plan.Namespace.ValueString(),
	}

	if !plan.Databases.IsNull() && !plan.Databases.IsUnknown() {
		var databases []databaseResource
		diags = plan.Databases.ElementsAs(ctx, &databases, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, db := range databases {
			state := api.StatePresent
			if !db.State.IsNull() && !db.State.IsUnknown() {
				state = api.State(db.State.ValueString())
			}

			var description *string
			if !db.Description.IsNull() && !db.Description.IsUnknown() {
				desc := db.Description.ValueString()
				description = &desc
			}

			input.Databases = append(input.Databases, api.DatabaseInput{
				Name:        db.Name.ValueString(),
				Description: description,
				State:       state,
			})
		}
	}

	if !plan.Users.IsNull() && !plan.Users.IsUnknown() {
		var users []databaseUserResource
		diags = plan.Users.ElementsAs(ctx, &users, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, user := range users {
			state := api.StatePresent
			if !user.State.IsNull() && !user.State.IsUnknown() {
				state = api.State(user.State.ValueString())
			}

			var password *string
			if !user.Password.IsNull() && !user.Password.IsUnknown() {
				pwd := user.Password.ValueString()
				password = &pwd
			}

			userInput := api.DatabaseUserInput{
				Name:     user.Name.ValueString(),
				Password: password,
				State:    state,
			}

			if !user.Permissions.IsNull() && !user.Permissions.IsUnknown() {
				var permissions []databaseUserPermissionResource
				diags = user.Permissions.ElementsAs(ctx, &permissions, false)
				resp.Diagnostics.Append(diags...)
				if resp.Diagnostics.HasError() {
					return
				}
				for _, perm := range permissions {
					userInput.Permissions = append(userInput.Permissions, api.DatabaseUserPermissionInput{
						DatabaseName: perm.Database.ValueString(),
						Permission:   api.DatabasePermission(perm.Permission.ValueString()),
						State:        api.StatePresent,
					})
				}
			}

			input.Users = append(input.Users, userInput)
		}
	}

	client := api.NewClient()
	cluster, err := client.CloudDatabaseClusterModify(input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating cloud database cluster", "Could not update cluster: "+err.Error())
		return
	}

	plan.ID = types.StringValue(cluster.Id)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *cloudDatabaseClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudDatabaseClusterResource
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := api.NewClient()
	input := api.CloudDatabaseClusterResourceInput{
		Name:      state.Name.ValueString(),
		Namespace: state.Namespace.ValueString(),
	}

	_, err := client.CloudDatabaseClusterDelete(input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting cloud database cluster",
			fmt.Sprintf("Failed to delete cluster %q: %s", state.Name.ValueString(), err.Error()),
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
	name := parts[1]

	client := api.NewClient()
	clusters, err := client.CloudDatabaseClusterList()
	if err != nil {
		resp.Diagnostics.AddError("Error importing cloud database cluster", "Could not list clusters: "+err.Error())
		return
	}

	var cluster *api.CloudDatabaseClusterResult
	for _, c := range clusters {
		if c.Name == name && c.Namespace.Name == namespace {
			cluster = &c
			break
		}
	}

	if cluster == nil {
		resp.Diagnostics.AddError(
			"Error importing cloud database cluster",
			fmt.Sprintf("Unable to find cluster %q in namespace %q", name, namespace),
		)
		return
	}

	// Set plan object for ImportState
	var amount *int64
	if cluster.Plan.Price.Amount != nil {
		val := int64(*cluster.Plan.Price.Amount)
		amount = &val
	}

	planPriceObj := types.ObjectValueMust(
		map[string]attr.Type{
			"amount":   types.Int64Type,
			"currency": types.StringType,
		},
		map[string]attr.Value{
			"amount":   types.Int64PointerValue(amount),
			"currency": types.StringPointerValue(cluster.Plan.Price.Currency),
		},
	)

	// Derive replicas from group name for imported resources
	var replicas int64 = 1 // default
	switch cluster.Plan.Group {
	case "Single (1 node)":
		replicas = 1
	case "Redundant (2 nodes)":
		replicas = 2
	case "Highly available (3 nodes)":
		replicas = 3
	}

	planObj := types.ObjectValueMust(
		map[string]attr.Type{
			"cpu":      types.Int64Type,
			"memory":   types.Float64Type,
			"storage":  types.Int64Type,
			"replicas": types.Int64Type,
			"id":       types.StringType,
			"name":     types.StringType,
			"group":    types.StringType,
			"price":    types.ObjectType{AttrTypes: map[string]attr.Type{
				"amount":   types.Int64Type,
				"currency": types.StringType,
			}},
		},
		map[string]attr.Value{
			"cpu":      types.Int64Value(int64(cluster.Plan.Cpu)),
			"memory":   types.Float64Value(cluster.Plan.Memory),
			"storage":  types.Int64Value(int64(cluster.Plan.Storage)),
			"replicas": types.Int64Value(replicas),
			"id":       types.StringValue(cluster.Plan.Id),
			"name":     types.StringNull(), // Plan result doesn't include name
			"group":    types.StringValue(cluster.Plan.Group),
			"price":    planPriceObj,
		},
	)

	state := cloudDatabaseClusterResource{
		ID:        types.StringValue(cluster.Id),
		Name:      types.StringValue(cluster.Name),
		Namespace: types.StringValue(cluster.Namespace.Name),
		Plan:      planObj,
	}

	specObj := types.ObjectValueMust(
		map[string]attr.Type{
			"type":    types.StringType,
			"version": types.StringType,
		},
		map[string]attr.Value{
			"type":    types.StringValue(cluster.Spec.Type),
			"version": types.StringValue(cluster.Spec.Version),
		},
	)
	state.Spec = specObj
	state.LastUpdated = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
