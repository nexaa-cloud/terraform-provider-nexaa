// Copyright IBM Corp. 2021, 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/nexaa-cloud/nexaa-cli/api"
)

type noEmptyAllowlistValidator struct{}

func (v noEmptyAllowlistValidator) Description(_ context.Context) string {
	return "Allowlist must not be empty, omit the field to use defaults."
}

func (v noEmptyAllowlistValidator) MarkdownDescription(_ context.Context) string {
	return "Allowlist must not be empty, omit the field to use defaults."
}

func (v noEmptyAllowlistValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if !req.ConfigValue.IsNull() && len(req.ConfigValue.Elements()) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid allowlist",
			"Allowlist must not be empty. Omit the field to use the defaults (0.0.0.0/0 and ::/0).",
		)
	}
}

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

	sort.Strings(result)

	return result
}

func toTypesStringList(ctx context.Context, stringArray []string) (types.List, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, types.StringType, stringArray)
	if diags.HasError() {
		return types.ListNull(types.StringType), diags
	}

	return list, nil
}

func buildAllowlistInput(ctx context.Context, oldAllowlist *types.List, newAllowlist types.List) []api.AllowListInput {
	newAllowlistArray := toStringArray(ctx, newAllowlist)
	var oldAllowlistArray []string

	if oldAllowlist != nil && !oldAllowlist.IsNull() && !oldAllowlist.IsUnknown() {
		oldAllowlistArray = toStringArray(ctx, *oldAllowlist)

	}

	var allowlist []api.AllowListInput
	plannedList := map[string]struct{}{}
	for _, ip := range newAllowlistArray {
		plannedList[ip] = struct{}{}
		allowlist = append(allowlist, api.AllowListInput{
			Ip:    ip,
			State: api.StatePresent,
		})
	}

	for _, ip := range oldAllowlistArray {
		if _, exists := plannedList[ip]; !exists {
			allowlist = append(allowlist, api.AllowListInput{
				Ip:    ip,
				State: api.StateAbsent,
			})
		}
	}

	return allowlist
}

/*



 */
