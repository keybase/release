// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package s3

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/alecthomas/template"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

type Link struct {
	Name string
	URL  string
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
			links = append(links, Link{Name: name, URL: urlString})
		}
	}

	return WriteHTMLForLinks(path, bucketName, bucketName, links)
}

var htmlTemplate = `
<!doctype html>
<html lang="en">
<head>
  <title>{{ .Title }}</title>
</head>
<body>
	<h3>{{ .Header }}</h3>
  {{ range $index, $value := .Links }}
    <li><a href="{{ $value.URL }}">{{ $value.Name }}</a></li>
  {{ end }}
</body>
</html>
`

func WriteHTMLForLinks(path string, title string, header string, links []Link) error {
	vars := map[string]interface{}{
		"Title":  title,
		"Header": header,
		"Links":  links,
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
