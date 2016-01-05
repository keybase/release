// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package s3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/blang/semver"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

type Section struct {
	Header   string
	Releases []Release
}

type Release struct {
	Name    string
	URL     string
	Version string
	Date    string
	Commit  string
}

func WriteHTML(path string, bucketName string, prefixes string, suffix string) error {
	auth, err := aws.EnvAuth()
	if err != nil {
		return err
	}
	client := s3.New(auth, aws.USEast)
	bucket := client.Bucket(bucketName)
	if bucket == nil {
		return fmt.Errorf("Bucket %s not found", bucketName)
	}

	var sections []Section
	for _, prefix := range strings.Split(prefixes, ",") {
		resp, err := bucket.List(prefix, "", "", 0)
		if err != nil {
			return err
		}

		var releases []Release
		for _, k := range resp.Contents {
			if strings.HasSuffix(k.Key, suffix) {
				key := k.Key
				name := key[len(prefix):]
				urlString := fmt.Sprintf("https://s3.amazonaws.com/%s/%s%s", bucketName, prefix, url.QueryEscape(name))
				version, date, commit, err := parseName(name, prefix)
				if err != nil {
					log.Printf("Couldn't get version from name: %s\n", name)
				}
				releases = append(releases,
					Release{
						Name:    name,
						URL:     urlString,
						Version: version,
						Date:    date.Format("Mon Jan _2 15:04:05 MST 2006"),
						Commit:  commit,
					})
			}
		}
		sections = append(sections, Section{
			Header:   prefix,
			Releases: reverse(releases),
		})
	}

	return WriteHTMLForLinks(path, bucketName, sections)
}

func parseName(name string, prefix string) (version string, t time.Time, commit string, err error) {
	t = time.Unix(0, 0)

	start := strings.IndexAny(name, "123456789")
	if start == -1 {
		return
	}
	suffix := filepath.Ext(name)

	verstr := name[start : len(name)-len(suffix)]
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

	d := fmt.Sprintf("%d", sversion.Pre[0].VersionNum)
	t, err = time.Parse("20060102150405", d)
	if err != nil {
		return
	}

	return
}

func reverse(a []Release) []Release {
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	return a
}

var htmlTemplate = `
<!doctype html>
<html lang="en">
<head>
  <title>{{ .Title }}</title>
	<style>
  body { font-family: monospace; }
  </style>
</head>
<body>
	{{ range $index, $sec := .Sections }}
		<h3>{{ $sec.Header }}</h3>
		<ul>
		{{ range $index2, $rel := $sec.Releases }}
		<li><a href="{{ $rel.URL }}">{{ $rel.Name }}</a> <strong>{{ $rel.Version }}</strong> <em>{{ $rel.Date }}</em> <a href="https://github.com/keybase/client/commit/{{ $rel.Commit }}"">{{ $rel.Commit }}</a></li>
		{{ end }}
		</ul>
	{{ end }}
</body>
</html>
`

func WriteHTMLForLinks(path string, title string, sections []Section) error {
	vars := map[string]interface{}{
		"Title":    title,
		"Sections": sections,
	}

	t, err := template.New("t").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	if path != "" {
		var data bytes.Buffer
		err = t.Execute(&data, vars)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(path, data.Bytes(), 0644)
	}
	return nil
}
