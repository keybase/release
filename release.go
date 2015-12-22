// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	keybase1 "github.com/keybase/client/go/protocol"
	gh "github.com/keybase/release/github"
	"github.com/keybase/release/s3"
	"github.com/keybase/release/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

func githubToken(required bool) string {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" && required {
		log.Fatal("No GITHUB_TOKEN set")
	}
	return token
}

func tag(version string) string {
	return fmt.Sprintf("v%s", version)
}

var (
	app               = kingpin.New("release", "Release tool for build and release scripts.")
	latestVersionCmd  = app.Command("latest-version", "Get latest version of a Github repo.")
	latestVersionUser = latestVersionCmd.Flag("user", "Github user").Required().String()
	latestVersionRepo = latestVersionCmd.Flag("repo", "Repository name").Required().String()

	platformCmd = app.Command("platform", "Get the OS platform name.")

	urlCmd     = app.Command("url", "Get the github release URL for a repo.")
	urlUser    = urlCmd.Flag("user", "Github user").Required().String()
	urlRepo    = urlCmd.Flag("repo", "Repository name").Required().String()
	urlVersion = urlCmd.Flag("version", "Version").Required().String()

	createCmd     = app.Command("create", "Create a Github release.")
	createRepo    = createCmd.Flag("repo", "Repository name").Required().String()
	createVersion = createCmd.Flag("version", "Version").Required().String()

	uploadCmd     = app.Command("upload", "Upload a file to a Github releaes.")
	uploadRepo    = uploadCmd.Flag("repo", "Repository name").Required().String()
	uploadVersion = uploadCmd.Flag("version", "Version").Required().String()
	uploadSrc     = uploadCmd.Flag("src", "Source file").Required().ExistingFile()
	uploadDest    = uploadCmd.Flag("dest", "Destination file").String()

	downloadCmd     = app.Command("download", "Download a file from a Github release.")
	downloadRepo    = downloadCmd.Flag("repo", "Repository name").Required().String()
	downloadVersion = downloadCmd.Flag("version", "Version").Required().String()
	downloadSrc     = downloadCmd.Flag("src", "Source file").Required().ExistingFile()

	updateJSONCmd     = app.Command("update-json", "Generate update.json file for updater.")
	updateJSONVersion = updateJSONCmd.Flag("version", "Version").Required().String()
	updateJSONSrc     = updateJSONCmd.Flag("src", "Source file").Required().ExistingFile()
	updateJSONURI     = updateJSONCmd.Flag("uri", "URI for location of files").Required().URL()

	indexHTMLCmd        = app.Command("index-html", "Generate index.html for s3 bucket.")
	indexHTMLBucketName = indexHTMLCmd.Flag("bucket-name", "Bucket name to index").Required().String()
	indexHTMLPrefix     = indexHTMLCmd.Flag("prefix", "Prefix of files").Required().String()
	indexHTMLSuffix     = indexHTMLCmd.Flag("suffix", "Suffix of files").String()
	indexHTMLDest       = indexHTMLCmd.Flag("dest", "Destination file").Required().String()

	parseVersionCmd    = app.Command("version-parse", "Parse a sematic version string.")
	parseVersionString = parseVersionCmd.Arg("version", "Semantic version to parse").Required().String()
)

func main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case latestVersionCmd.FullCommand():
		tag, err := gh.LatestTag(*latestVersionUser, *latestVersionRepo, githubToken(false))
		if err != nil {
			log.Fatal(err)
		}
		if strings.HasPrefix(tag.Name, "v") {
			version := tag.Name[1:]
			fmt.Printf("%s", version)
		}
	case platformCmd.FullCommand():
		fmt.Printf("%s", runtime.GOOS)

	case urlCmd.FullCommand():
		release, err := gh.ReleaseOfTag(*urlUser, *urlRepo, tag(*urlVersion), githubToken(false))
		if _, ok := err.(*gh.ErrNotFound); ok {
			// No release
		} else if err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("%s", release.URL)
		}
	case createCmd.FullCommand():
		err := gh.CreateRelease(githubToken(true), *createRepo, tag(*createVersion), tag(*createVersion))
		if err != nil {
			log.Fatal(err)
		}
	case uploadCmd.FullCommand():
		if *uploadDest == "" {
			uploadDest = uploadSrc
		}
		log.Printf("Uploading %s as %s (%s)", *uploadSrc, *uploadDest, tag(*uploadVersion))
		err := gh.Upload(githubToken(true), *uploadRepo, tag(*uploadVersion), *uploadDest, *uploadSrc)
		if err != nil {
			log.Fatal(err)
		}
	case downloadCmd.FullCommand():
		defaultSrc := fmt.Sprintf("keybase-%s-%s.tgz", *downloadVersion, runtime.GOOS)
		if *downloadSrc == "" {
			downloadSrc = &defaultSrc
		}
		log.Printf("Downloading %s (%s)", *downloadSrc, tag(*downloadVersion))
		err := gh.DownloadAsset(githubToken(false), *downloadRepo, tag(*downloadVersion), *downloadSrc)
		if err != nil {
			log.Fatal(err)
		}
	case updateJSONCmd.FullCommand():
		fileName := path.Base(*updateJSONSrc)
		urlString := fmt.Sprintf("%s/%s", *updateJSONURI, url.QueryEscape(fileName))
		update := keybase1.Update{
			Version: *updateJSONVersion,
			Name:    tag(*updateJSONVersion),
			Asset: keybase1.Asset{
				Name: fileName,
				Url:  urlString,
			},
		}
		out, err := json.MarshalIndent(update, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stdout, "%s\n", out)
	case indexHTMLCmd.FullCommand():
		err := s3.WriteHTML(*indexHTMLDest, *indexHTMLBucketName, *indexHTMLPrefix, *indexHTMLSuffix)
		if err != nil {
			log.Fatal(err)
		}
	case parseVersionCmd.FullCommand():
		parsed, err := version.ParseVersion(*parseVersionString)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(strings.Join(parsed, "\n"))
	}
}