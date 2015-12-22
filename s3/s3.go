// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package s3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

type Link struct {
	Name         string
	URL          string
	LastModified string
}

func WriteHTML(path string, bucketName string, prefix string, suffix string) error {
	auth, err := aws.EnvAuth()
	if err != nil {
		return err
	}
	client := s3.New(auth, aws.USEast)
	bucket := client.Bucket(bucketName)
	if bucket == nil {
		return fmt.Errorf("Bucket %s not found", bucketName)
	}
	resp, err := bucket.List(prefix, "", "", 0)
	if err != nil {
		return err
	}

	var links []Link
	for _, k := range resp.Contents {
		if strings.HasSuffix(k.Key, suffix) {
			name := k.Key
			urlString := fmt.Sprintf("https://s3.amazonaws.com/%s/%s", bucketName, url.QueryEscape(k.Key))

			lastModfified, err := time.Parse(time.RFC3339, k.LastModified)
			if err != nil {
				return err
			}

			links = append(links, Link{Name: name, URL: urlString, LastModified: lastModfified.Format("Mon Jan _2 15:04:05 2006")})
		}
	}

	return WriteHTMLForLinks(path, bucketName, bucketName, links)
}

func reverse(a []Link) []Link {
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
	<h3>{{ .Header }}</h3>
  {{ range $index, $value := .Links }}
    <li><a href="{{ $value.URL }}">{{ $value.Name }}</a> {{ $value.LastModified }}</li>
  {{ end }}
</body>
</html>
`

func WriteHTMLForLinks(path string, title string, header string, links []Link) error {
	vars := map[string]interface{}{
		"Title":  title,
		"Header": header,
		"Links":  reverse(links),
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
