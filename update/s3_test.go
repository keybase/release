// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package update

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindRelease(t *testing.T) {
	first := func(r Release) bool { return true }
	release, err := platformDarwin.FindRelease("prerelease.keybase.io", first)
	require.NoError(t, err)
	t.Logf("Release: %#v", release)
	assert.NotEqual(t, "", release.URL)
}
