// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// --- Shared logic ---

func immutableDescription() string {
	return "Errors if this attribute is changed after creation."
}

// --- String ---

type immutableStringModifier struct{}

func ImmutableString() planmodifier.String {
	return immutableStringModifier{}
}

func (m immutableStringModifier) Description(_ context.Context) string { return immutableDescription() }
func (m immutableStringModifier) MarkdownDescription(_ context.Context) string {
	return immutableDescription()
}

func (m immutableStringModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Attribute is immutable",
			fmt.Sprintf("Cannot change from %q to %q.", req.StateValue.ValueString(), req.PlanValue.ValueString()),
		)
	}
}

// --- Object ---

type immutableObjectModifier struct{}

func ImmutableObject() planmodifier.Object {
	return immutableObjectModifier{}
}

func (m immutableObjectModifier) Description(_ context.Context) string { return immutableDescription() }
func (m immutableObjectModifier) MarkdownDescription(_ context.Context) string {
	return immutableDescription()
}

func (m immutableObjectModifier) PlanModifyObject(ctx context.Context, req planmodifier.ObjectRequest, resp *planmodifier.ObjectResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Attribute is immutable",
			fmt.Sprintf("Cannot change attribute after creation. Current value is %s, new value is %s.",
				req.StateValue.String(),
				req.PlanValue.String(),
			),
		)
	}
}

// --- List ---

type immutableListModifier struct{}

func ImmutableList() planmodifier.List {
	return immutableListModifier{}
}

func (m immutableListModifier) Description(_ context.Context) string { return immutableDescription() }
func (m immutableListModifier) MarkdownDescription(_ context.Context) string {
	return immutableDescription()
}

func (m immutableListModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Attribute is immutable",
			fmt.Sprintf("Cannot change attribute after creation. Current value is %s, new value is %s.",
				req.StateValue.String(),
				req.PlanValue.String(),
			),
		)
	}
}
