// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package version

import (
	"fmt"
	"regexp"
	"time"
)

func Parse(name string) (version string, t time.Time, commit string, err error) {
	versionRegex, _ := regexp.Compile("(\\d+\\.\\d+\\.\\d+)[-.](\\d+)[+.]([[:alnum:]]+)")
	parts := versionRegex.FindAllStringSubmatch(name, -1)
	if len(parts) == 0 || len(parts[0]) < 4 {
		err = fmt.Errorf("Unable to parse: %s", name)
		return
	}
	version = parts[0][1]
	date := parts[0][2]
	commit = parts[0][3]
	t, _ = time.Parse("20060102150405", date)
	return
}
