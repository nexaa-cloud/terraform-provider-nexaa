// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

type noDuplicateDefaultIngressValidator struct{}

func (v noDuplicateDefaultIngressValidator) Description(_ context.Context) string {
	return "At most one ingress may omit domain_name, as all omitted domain names resolve to the same default."
}

func (v noDuplicateDefaultIngressValidator) MarkdownDescription(_ context.Context) string {
	return "At most one ingress may omit `domain_name`, as all omitted domain names resolve to the same default."
}

func (v noDuplicateDefaultIngressValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var ingresses []ingresResource
	diags := req.ConfigValue.ElementsAs(ctx, &ingresses, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	seen := make(map[string]bool)
	defaultCount := 0
	for _, ing := range ingresses {
		if ing.DomainName.IsUnknown() {
			continue
		}
		if ing.DomainName.IsNull() {
			defaultCount++
			if defaultCount > 1 {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Duplicate default ingress domain",
					fmt.Sprintf(
						"%d ingresses have no domain_name set. All unset domain names resolve to the same default domain, "+
							"which is invalid. Set a unique domain_name for each additional ingress.",
						defaultCount,
					),
				)
				return
			}
			continue
		}
		domain := ing.DomainName.ValueString()
		if seen[domain] {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Duplicate ingress domain_name",
				fmt.Sprintf("domain_name %q is used more than once. Each ingress must have a unique domain_name.", domain),
			)
			return
		}
		seen[domain] = true
	}
}
