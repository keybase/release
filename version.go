// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package main

import (
	"fmt"

	"github.com/blang/semver"
)

func ParseVersion(s string) ([]string, error) {
	version, err := semver.Make(s)
	if err != nil {
		return nil, err
	}

	parsed := []string{
		fmt.Sprintf("%d", version.Major),
		fmt.Sprintf("%d", version.Minor),
		fmt.Sprintf("%d", version.Patch),
	}

	if len(version.Pre) > 1 {
		return nil, fmt.Errorf("Multiple prerelease not supported")
	}

	if len(version.Pre) == 1 {
		if version.Pre[0].IsNum {
			parsed = append(parsed, fmt.Sprintf("%d", version.Pre[0].VersionNum))
		} else {
			parsed = append(parsed, version.Pre[0].VersionStr)
		}
	} else {
		parsed = append(parsed, "")
	}

	if len(version.Build) > 1 {
		return nil, fmt.Errorf("Multiple comments not supported")
	}

	if len(version.Build) == 1 {
		parsed = append(parsed, version.Build[0])
	} else {
		parsed = append(parsed, "")
	}

	return parsed, nil
}
