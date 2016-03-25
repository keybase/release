// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package update

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alecthomas/template"
	"github.com/blang/semver"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	keybase1 "github.com/keybase/client/go/protocol"
	"github.com/keybase/release/version"
)

// Section defines a set of releases
type Section struct {
	Header   string
	Releases []Release
}

// Release defines a release bundle
type Release struct {
	Name       string
	Key        s3.Key
	URL        string
	Version    string
	DateString string
	Date       time.Time
	Commit     string
}

// ByRelease defines how to sort releases
type ByRelease []Release

func (s ByRelease) Len() int {
	return len(s)
}

func (s ByRelease) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByRelease) Less(i, j int) bool {
	// Reverse date order
	return s[j].Date.Before(s[i].Date)
}

// Client knows how to find and save releases
type Client struct {
	s3 *s3.S3
}

// NewClient constructs a Client
func NewClient() (*Client, error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return nil, err
	}
	s3 := s3.New(auth, aws.USEast)
	return &Client{s3: s3}, nil
}

func convertEastern(t time.Time) time.Time {
	locationNewYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("Couldn't load location: %s", err)
	}
	return t.In(locationNewYork)
}

func loadReleases(keys []s3.Key, bucketName string, prefix string, suffix string, truncate int) []Release {
	var releases []Release
	for _, k := range keys {
		if strings.HasSuffix(k.Key, suffix) {
			urlString, name := urlStringForKey(k, bucketName, prefix)
			if name == "index.html" {
				continue
			}
			version, _, date, commit, err := version.Parse(name)
			if err != nil {
				log.Printf("Couldn't get version from name: %s\n", name)
			}
			date = convertEastern(date)
			releases = append(releases,
				Release{
					Name:       name,
					Key:        k,
					URL:        urlString,
					Version:    version,
					Date:       date,
					DateString: date.Format("Mon Jan _2 15:04:05 MST 2006"),
					Commit:     commit,
				})
		}
	}
	// TODO: Should also sanity check that version sort is same as time sort
	// otherwise something got messed up
	sort.Sort(ByRelease(releases))
	if truncate > 0 && len(releases) > truncate {
		releases = releases[0:truncate]
	}
	return releases
}

// WriteHTML creates an html file for releases
func WriteHTML(bucketName string, prefixes string, suffix string, outPath string, uploadDest string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	bucket := client.s3.Bucket(bucketName)
	if bucket == nil {
		return fmt.Errorf("Bucket %s not found", bucketName)
	}

	var sections []Section
	for _, prefix := range strings.Split(prefixes, ",") {
		var resp *s3.ListResp
		resp, err = bucket.List(prefix, "", "", 0)
		if err != nil {
			return err
		}

		releases := loadReleases(resp.Contents, bucketName, prefix, suffix, 50)
		if len(releases) > 0 {
			log.Printf("Found %d release(s) at %s\n", len(releases), prefix)
			// for _, release := range releases {
			// 	log.Printf(" %s %s %s\n", release.Name, release.Version, release.DateString)
			// }
		}
		sections = append(sections, Section{
			Header:   prefix,
			Releases: releases,
		})
	}

	var buf bytes.Buffer
	err = WriteHTMLForLinks(bucketName, sections, &buf)
	if err != nil {
		return err
	}
	if outPath != "" {
		err = makeParentDirs(outPath)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(outPath, buf.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	if uploadDest != "" {
		client, err := NewClient()
		if err != nil {
			return err
		}
		bucket = client.s3.Bucket(bucketName)
		log.Printf("Uploading to %s", urlStringNoEscape(bucketName, uploadDest))
		err = bucket.Put(uploadDest, buf.Bytes(), "text/html", s3.PublicRead, s3.Options{})
		if err != nil {
			return err
		}
	}

	return nil
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

// WriteHTMLForLinks writes a summary document for a set of releases
func WriteHTMLForLinks(title string, sections []Section, writer io.Writer) error {
	vars := map[string]interface{}{
		"Title":    title,
		"Sections": sections,
	}

	t, err := template.New("t").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	return t.Execute(writer, vars)
}

// Platform defines where platform specific files are (in darwin, linux, windows)
type Platform struct {
	Name          string
	Prefix        string
	PrefixSupport string
	Suffix        string
	LatestName    string
}

// CopyLatest copies latest release to a fixed path
func CopyLatest(bucketName string, platform string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	return client.CopyLatest(bucketName, platform)
}

const (
	// PlatformTypeDarwin is platform type for OS X
	PlatformTypeDarwin = "darwin"
	// PlatformTypeLinux is platform type for Linux
	PlatformTypeLinux = "linux"
	// PlatformTypeWindows is platform type for windows
	PlatformTypeWindows = "windows"
)

var platformDarwin = Platform{Name: PlatformTypeDarwin, Prefix: "darwin/", PrefixSupport: "darwin-support/", LatestName: "Keybase.dmg"}
var platformLinuxDeb = Platform{Name: "deb", Prefix: "linux_binaries/deb/", Suffix: "_amd64.deb", LatestName: "keybase_amd64.deb"}
var platformLinuxRPM = Platform{Name: "rpm", Prefix: "linux_binaries/rpm/", Suffix: ".x86_64.rpm", LatestName: "keybase_amd64.rpm"}
var platformWindows = Platform{Name: PlatformTypeWindows, Prefix: "windows/", Suffix: ".386.exe", LatestName: "keybase_setup_386.exe"}

var platformsAll = []Platform{
	platformDarwin,
	platformLinuxDeb,
	platformLinuxRPM,
	platformWindows,
}

// Platforms returns platforms for a name (linux may have multiple platforms) or all platforms is "" is specified
func Platforms(name string) ([]Platform, error) {
	switch name {
	case PlatformTypeDarwin:
		return []Platform{platformDarwin}, nil
	case PlatformTypeLinux:
		return []Platform{platformLinuxDeb, platformLinuxRPM}, nil
	case PlatformTypeWindows:
		return []Platform{platformWindows}, nil
	case "":
		return platformsAll, nil
	default:
		return nil, fmt.Errorf("Invalid platform %s", name)
	}
}

// FindRelease searches for a release matching a predicate
func (p *Platform) FindRelease(bucket s3.Bucket, f func(r Release) bool) (*Release, error) {
	resp, err := bucket.List(p.Prefix, "", "", 0)
	if err != nil {
		return nil, err
	}
	releases := loadReleases(resp.Contents, bucket.Name, p.Prefix, p.Suffix, 0)
	for _, release := range releases {
		k := release.Key
		if !strings.HasSuffix(k.Key, p.Suffix) {
			continue
		}
		if f(release) {
			return &release, nil
		}
	}
	return nil, nil
}

// CopyLatest copies latest release to a fixed path for the Client
func (c *Client) CopyLatest(bucketName string, platform string) error {
	platforms, err := Platforms(platform)
	if err != nil {
		return err
	}
	for _, platform := range platforms {
		var url string
		// Use update json to look for current DMG (for darwin)
		// TODO: Fix for linux, windows
		if platform.Name == PlatformTypeDarwin {
			url, err = c.copyFromUpdate(platform, bucketName)
		} else {
			_, url, err = c.copyFromReleases(platform, bucketName)
		}
		if err != nil {
			return err
		}
		if url == "" {
			continue
		}
		bucket := c.s3.Bucket(bucketName)
		log.Printf("PutCopying %s to %s\n", url, platform.LatestName)
		_, err = bucket.PutCopy(platform.LatestName, s3.PublicRead, s3.CopyOptions{}, url)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) copyFromUpdate(platform Platform, bucketName string) (url string, err error) {
	currentUpdate, path, err := c.CurrentUpdate(bucketName, "", platform.Name, "prod")
	if err != nil {
		return
	}
	if currentUpdate == nil {
		err = fmt.Errorf("No latest for %s at %s", platform.Name, path)
		return
	}
	switch platform.Name {
	case PlatformTypeDarwin:
		url = urlString(bucketName, platform.Prefix, fmt.Sprintf("Keybase-%s.dmg", currentUpdate.Version))
	default:
		err = fmt.Errorf("Unsupported platform for copyFromUpdate")
	}
	return
}

func (c *Client) copyFromReleases(platform Platform, bucketName string) (release *Release, url string, err error) {
	bucket := c.s3.Bucket(bucketName)
	release, err = platform.FindRelease(*bucket, func(r Release) bool { return true })
	if err != nil || release == nil {
		return
	}
	url, _ = urlStringForKey(release.Key, bucketName, platform.Prefix)
	return
}

// CurrentUpdate returns current update for a platform
func (c *Client) CurrentUpdate(bucketName string, channel string, platformName string, env string) (currentUpdate *keybase1.Update, path string, err error) {
	bucket := c.s3.Bucket(bucketName)
	path = updateJSONName(channel, platformName, env)
	data, err := bucket.Get(path)
	if err != nil {
		return
	}
	currentUpdate, err = DecodeJSON(data)
	return
}

func promoteRelease(bucketName string, delay time.Duration, hourEastern int, channel string, platform Platform, env string) (*Release, error) {
	client, err := NewClient()
	if err != nil {
		return nil, err
	}
	return client.PromoteRelease(bucketName, delay, hourEastern, channel, platform, env)
}

func updateJSONName(channel string, platformName string, env string) string {
	if channel == "" {
		return fmt.Sprintf("update-%s-%s.json", platformName, env)
	}
	return fmt.Sprintf("update-%s-%s-%s.json", platformName, env, channel)
}

// PromoteRelease promotes a test release to the public
func (c *Client) PromoteRelease(bucketName string, delay time.Duration, beforeHourEastern int, channel string, platform Platform, env string) (*Release, error) {
	if channel == "" {
		log.Printf("Finding release to promote (%s delay, < %dam)", delay, beforeHourEastern)
	} else {
		log.Printf("Finding release to promote for %s channel (%s delay)", channel, delay)
	}
	bucket := c.s3.Bucket(bucketName)

	release, err := platform.FindRelease(*bucket, func(r Release) bool {
		log.Printf("Checking release date %s", r.Date)
		if delay != 0 && time.Since(r.Date) < delay {
			return false
		}
		hour, _, _ := r.Date.Clock()
		if beforeHourEastern != 0 && hour >= beforeHourEastern {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	if release == nil {
		log.Printf("No matching release found")
		return nil, nil
	}
	log.Printf("Found release %s (%s), %s", release.Name, time.Since(release.Date), release.Version)

	currentUpdate, _, err := c.CurrentUpdate(bucketName, channel, platform.Name, env)
	if err != nil {
		log.Printf("Error looking for current update: %s (%s)", err, platform.Name)
	}
	if currentUpdate != nil {
		log.Printf("Found update: %s", currentUpdate.Version)
		var currentVer semver.Version
		currentVer, err = semver.Make(currentUpdate.Version)
		if err != nil {
			return nil, err
		}
		var releaseVer semver.Version
		releaseVer, err = semver.Make(release.Version)
		if err != nil {
			return nil, err
		}

		if releaseVer.Equals(currentVer) {
			log.Printf("Release unchanged")
			return nil, nil
		} else if releaseVer.LT(currentVer) {
			log.Printf("Release older than current update")
			return nil, nil
		}
	}

	jsonName := updateJSONName(channel, platform.Name, env)
	jsonURL := urlString(bucketName, platform.PrefixSupport, fmt.Sprintf("update-%s-%s-%s.json", platform.Name, env, release.Version))
	log.Printf("PutCopying %s to %s\n", jsonURL, jsonName)
	_, err = bucket.PutCopy(jsonName, s3.PublicRead, s3.CopyOptions{}, jsonURL)
	if err != nil {
		return nil, err
	}
	return release, nil
}

func copyUpdateJSON(bucketName string, channel string, platformName string, env string) error {
	client, err := NewClient()
	if err != nil {
		return err
	}
	jsonNameDest := updateJSONName(channel, platformName, env)
	jsonURLSource := urlString(bucketName, "", updateJSONName("", platformName, env))
	bucket := client.s3.Bucket(bucketName)
	log.Printf("PutCopying %s to %s\n", jsonURLSource, jsonNameDest)
	_, err = bucket.PutCopy(jsonNameDest, s3.PublicRead, s3.CopyOptions{}, jsonURLSource)
	return err
}

func (c *Client) report(tw *tabwriter.Writer, bucketName string, channel string, platformName string) {
	update, _, err := c.CurrentUpdate(bucketName, channel, platformName, "prod")
	if channel == "" {
		channel = "public"
	}
	fmt.Fprintf(tw, fmt.Sprintf("%s\t%s\t", platformName, channel))
	if err != nil {
		fmt.Fprintln(tw, "Error")
	} else if update != nil {
		published := ""
		if update.PublishedAt != nil {
			published = convertEastern(keybase1.FromTime(*update.PublishedAt)).Format(time.UnixDate)
		}
		fmt.Fprintf(tw, "%s\t%s\n", update.Version, published)
	} else {
		fmt.Fprintln(tw, "None")
	}
}

// Report returns a summary of releases
func Report(bucketName string, writer io.Writer) error {
	client, err := NewClient()
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(writer, 5, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "Platform\tType\tVersion\tCreated")
	client.report(tw, bucketName, "test", PlatformTypeDarwin)
	client.report(tw, bucketName, "", PlatformTypeDarwin)
	client.report(tw, bucketName, "test", PlatformTypeLinux)
	client.report(tw, bucketName, "", PlatformTypeLinux)
	client.report(tw, bucketName, "test", PlatformTypeWindows)
	client.report(tw, bucketName, "", PlatformTypeWindows)
	tw.Flush()
	return nil
}

func promoteTestReleaseForDarwin(bucketName string) (*Release, error) {
	return promoteRelease(bucketName, time.Duration(0), 0, "test", platformDarwin, "prod")
}

func promoteTestReleaseForLinux(bucketName string) error {
	return copyUpdateJSON(bucketName, "test", PlatformTypeLinux, "prod")
}

func promoteTestReleaseForWindows(bucketName string) error {
	return copyUpdateJSON(bucketName, "test", PlatformTypeWindows, "prod")
}

// PromoteTestReleases promotes test releases for a platform
func PromoteTestReleases(bucketName string, platform string) error {
	switch platform {
	case PlatformTypeDarwin:
		_, err := promoteTestReleaseForDarwin(bucketName)
		return err
	case PlatformTypeLinux:
		return promoteTestReleaseForLinux(bucketName)
	case PlatformTypeWindows:
		return promoteTestReleaseForWindows(bucketName)
	default:
		return fmt.Errorf("Invalid platform %s", platform)
	}
}

// PromoteReleases promotes releases for a platform
func PromoteReleases(bucketName string, platform string) error {
	switch platform {
	case PlatformTypeDarwin:
		release, err := promoteRelease(bucketName, time.Hour*27, 10, "", platformDarwin, "prod")
		if err != nil {
			return err
		}
		if release != nil {
			log.Printf("Promoted (darwin) release: %s\n", release.Name)
		}
	case PlatformTypeLinux:
		log.Printf("Promoting releases is unsupported for linux")
	case PlatformTypeWindows:
		log.Printf("Promoting releases is unsupported for windows")
	}
	return nil
}
