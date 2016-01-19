// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package version

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
)

func Parse(name string) (version string, t time.Time, commit string, err error) {
	t = time.Unix(0, 0)

	start := strings.IndexAny(name, "123456789")
	if start == -1 {
		return
	}
	name = name[start:len(name)]
	verstr := removeExt(name)
	sversion, err := semver.Make(verstr)
	if err != nil {
		return
	}
	version = fmt.Sprintf("%d.%d.%d", sversion.Major, sversion.Minor, sversion.Patch)

	if len(sversion.Pre) != 1 {
		err = fmt.Errorf("Invalid prerelease")
		return
	}

	commit = strings.Join(sversion.Build, " ")
	// Detect if really sha commit
	if len(commit) != 7 {
		commit = ""
	}

	d := fmt.Sprintf("%d", sversion.Pre[0].VersionNum)
	t, err = time.Parse("20060102150405", d)
	if err != nil {
		return
	}

	return
}

func removeExt(name string) string {
	suffix := filepath.Ext(name)
	if len(suffix) > 0 {
		name = name[0 : len(name)-len(suffix)]
	}
	return name
}
