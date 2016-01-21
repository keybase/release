// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package version

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/blang/semver"
)

func Parse(name string) (version string, t time.Time, commit string, err error) {
	if strings.HasSuffix(name, ".deb") || strings.HasSuffix(name, ".rpm") {
		return ParseLinux(name)
	}

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

// The Linux packages have different patterns of dashes and underscores, and
// RPM requires some hacks that break SemVer. Parse these packages with a
// stupid regex.
func ParseLinux(name string) (version string, t time.Time, commit string, err error) {
	versionRegex, _ := regexp.Compile("(\\d+\\.\\d+\\.\\d+)[-.](\\d+)[+.]([[:alnum:]]+)")
	parts := versionRegex.FindAllStringSubmatch(name, -1)
	version = parts[0][1]
	date := parts[0][2]
	commit = parts[0][3]
	t, _ = time.Parse("20060102150405", date)
	return
}

func removeExt(name string) string {
	suffix := filepath.Ext(name)
	if len(suffix) > 0 {
		name = name[0 : len(name)-len(suffix)]
	}
	return name
}
