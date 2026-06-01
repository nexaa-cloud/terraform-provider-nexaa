// Copyright Tilaa B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package resources

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsCannotBeDeletedErr_nil(t *testing.T) {
	assert.False(t, isCannotBeDeletedErr(nil))
}

func Test_IsCannotBeDeletedErr_unrelated_error(t *testing.T) {
	assert.False(t, isCannotBeDeletedErr(errors.New("some other error")))
}

func Test_IsCannotBeDeletedErr_matching_lowercase(t *testing.T) {
	assert.True(t, isCannotBeDeletedErr(errors.New("namespace cannot be deleted: has active resources")))
}

func Test_IsCannotBeDeletedErr_matching_mixed_case(t *testing.T) {
	assert.True(t, isCannotBeDeletedErr(errors.New("Cannot Be Deleted")))
}
