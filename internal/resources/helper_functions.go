package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func toStringArray(ctx context.Context, listInput types.List) []string {
	var rawList []types.String
	result := []string{}
	if listInput.IsNull() {
		return result
	}

	if listInput.IsUnknown() {
		return result
	}

	_ = listInput.ElementsAs(ctx, &rawList, false)
	for _, element := range rawList {
		result = append(result, element.ValueString())
	}
	return result
}
