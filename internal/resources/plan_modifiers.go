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

func isImmutableChangeAllowed[T interface {
	IsNull() bool
	IsUnknown() bool
	Equal(T) bool
}](state, plan T) bool {
	return state.IsNull() || state.IsUnknown()
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

// --- Int64 ---

type immutableInt64Modifier struct{}

func ImmutableInt64() planmodifier.Int64 {
	return immutableInt64Modifier{}
}

func (m immutableInt64Modifier) Description(_ context.Context) string { return immutableDescription() }
func (m immutableInt64Modifier) MarkdownDescription(_ context.Context) string {
	return immutableDescription()
}

func (m immutableInt64Modifier) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Attribute is immutable",
			fmt.Sprintf("Cannot change from %d to %d.", req.StateValue.ValueInt64(), req.PlanValue.ValueInt64()),
		)
	}
}

// --- Bool ---

type immutableBoolModifier struct{}

func ImmutableBool() planmodifier.Bool {
	return immutableBoolModifier{}
}

func (m immutableBoolModifier) Description(_ context.Context) string { return immutableDescription() }
func (m immutableBoolModifier) MarkdownDescription(_ context.Context) string {
	return immutableDescription()
}

func (m immutableBoolModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if !req.PlanValue.Equal(req.StateValue) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Attribute is immutable",
			fmt.Sprintf("Cannot change from %t to %t.", req.StateValue.ValueBool(), req.PlanValue.ValueBool()),
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
			fmt.Sprintf("Cannot change spec after creation. Current value is %s, new value is %s.",
				req.StateValue.String(),
				req.PlanValue.String(),
			),
		)
	}
}
