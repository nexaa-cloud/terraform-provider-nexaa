// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"context"

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
