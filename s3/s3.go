// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package s3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/keybase/release/version"
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

func NewClient() (client *s3.S3, err error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return
	}
	client = s3.New(auth, aws.USEast)
	return
}

func WriteHTML(path string, bucketName string, prefixes string, suffix string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
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
		keys := reverseKey(resp.Contents, 20)

		var releases []Release
		for _, k := range keys {
			if strings.HasSuffix(k.Key, suffix) {
				key := k.Key
				name := key[len(prefix):]
				urlString := fmt.Sprintf("https://s3.amazonaws.com/%s/%s%s", bucketName, prefix, url.QueryEscape(name))
				version, date, commit, err := version.Parse(name)
				if err != nil {
					log.Printf("Couldn't get version from name: %s\n", name)
				}

				// Convert to Eastern
				locationNewYork, err := time.LoadLocation("America/New_York")
				if err != nil {
					log.Printf("Couldn't load location: %s", err)
				}
				date = date.In(locationNewYork)

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
			Releases: releases,
		})
	}

	return WriteHTMLForLinks(path, bucketName, sections)
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
		err := makeParentDirs(path)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(path, data.Bytes(), 0644)
	}
	return nil
}

type Link struct {
	Prefix string
	Suffix string
	Name   string
}

func CopyLatest(bucketName string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	bucket := client.Bucket(bucketName)

	linksForPrefix := []Link{
		Link{Prefix: "darwin/", Name: "Keybase.dmg"},
		Link{Prefix: "linux_binaries/deb/", Suffix: "_amd64.deb", Name: "keybase_amd64.deb"},
		Link{Prefix: "linux_binaries/rpm/", Suffix: ".x86_64.rpm", Name: "keybase_amd64.rpm"},
	}

	for _, link := range linksForPrefix {
		resp, err := bucket.List(link.Prefix, "", "", 0)
		if err != nil {
			return err
		}
		keys := reverseKey(resp.Contents, 0)
		for _, k := range keys {
			if !strings.HasSuffix(k.Key, link.Suffix) {
				continue
			}

			url := urlString(k, bucketName, link.Prefix)
			// Instead of linking, we're making copies. S3 linking has some issues.
			// headers := map[string][]string{
			// 	"x-amz-website-redirect-location": []string{url},
			// }
			//err = bucket.PutHeader(name, []byte{}, headers, s3.PublicRead)
			log.Printf("Copying %s from %s (latest)\n", link.Name, k.Key)
			_, err = bucket.PutCopy(link.Name, s3.PublicRead, s3.CopyOptions{}, url)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func urlString(k s3.Key, bucketName string, prefix string) string {
	key := k.Key
	name := key[len(prefix):]
	return fmt.Sprintf("https://s3.amazonaws.com/%s/%s%s", bucketName, prefix, url.QueryEscape(name))
}

func reverseKey(a []s3.Key, truncate int) []s3.Key {
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	if truncate > 0 && len(a) > truncate {
		a = a[0:truncate]
	}
	return a
}

func makeParentDirs(filename string) error {
	dir, _ := filepath.Split(filename)
	exists, err := fileExists(dir)
	if err != nil {
		return err
	}

	if !exists {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
