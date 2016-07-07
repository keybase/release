// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"
)

// CreateRelease creates a release for a tag
func CreateRelease(token string, repo string, tag string, name string) error {
	params := ReleaseCreate{
		TagName: tag,
		Name:    name,
	}

	payload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("can't encode release creation params, %v", err)
	}
	reader := bytes.NewReader(payload)

	uri := fmt.Sprintf("/repos/keybase/%s/releases", repo)
	resp, err := DoAuthRequest("POST", githubAPIURL+uri, "application/json", token, nil, reader)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return fmt.Errorf("while submitting %v, %v", string(payload), err)
	}
	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == 422 {
			return fmt.Errorf("github returned %v (this is probably because the release already exists)",
				resp.Status)
		}
		return fmt.Errorf("github returned %v", resp.Status)
	}
	return nil
}

// Upload uploads a file to a tagged repo
func Upload(token string, repo string, tag string, name string, file string) error {
	release, err := ReleaseOfTag("keybase", repo, tag, token)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("name", name)
	url := release.CleanUploadURL() + "?" + v.Encode()
	osfile, err := os.Open(file)
	if err != nil {
		return err
	}
	resp, err := DoAuthRequest("POST", url, "application/octet-stream", token, nil, osfile)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == 422 {
			return fmt.Errorf("github returned %v (this is probably because the release already exists)",
				resp.Status)
		}
		return fmt.Errorf("github returned %v", resp.Status)
	}
	return nil
}

// DownloadSource dowloads source from repo tag
func DownloadSource(token string, repo string, tag string) error {
	url := githubAPIURL + fmt.Sprintf("/repos/keybase/%s/tarball/%s", repo, tag)
	name := fmt.Sprintf("%s-%s.tar.gz", repo, tag)
	log.Printf("Url: %s", url)
	return Download(token, url, name)
}

// DownloadAsset downloads an asset from Github that matches name
func DownloadAsset(token string, repo string, tag string, name string) error {
	release, err := ReleaseOfTag("keybase", repo, tag, token)
	if err != nil {
		return err
	}

	assetID := 0
	for _, asset := range release.Assets {
		if asset.Name == name {
			assetID = asset.ID
		}
	}

	if assetID == 0 {
		return fmt.Errorf("could not find asset named %s", name)
	}

	url := githubAPIURL + fmt.Sprintf(assetDownloadURI, "keybase", repo, assetID)
	return Download(token, url, name)
}

// Download from Github
func Download(token string, url string, name string) error {
	resp, err := DoAuthRequest("GET", url, "", token, map[string]string{
		"Accept": "application/octet-stream",
	}, nil)
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		return fmt.Errorf("could not fetch releases, %v", err)
	}

	contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github did not respond with 200 OK but with %v", resp.Status)
	}

	out, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("could not create file %s", name)
	}
	defer func() { _ = out.Close() }()

	n, err := io.Copy(out, resp.Body)
	if n != contentLength {
		return fmt.Errorf("downloaded data did not match content length %d != %d", contentLength, n)
	}
	return err
}

// LatestCommit returns a latest commit for all statuses matching state and contexts
func LatestCommit(token string, repo string, contexts []string) (*Commit, error) {
	commits, err := Commits("keybase", repo, token)
	if err != nil {
		return nil, err
	}

	for _, commit := range commits {
		log.Printf("Checking %s", commit.SHA)
		statuses, err := Statuses("keybase", repo, commit.SHA, token)
		if err != nil {
			return nil, err
		}
		matching := map[string]Status{}
		for _, status := range statuses {
			if stringInSlice(status.Context, contexts) {
				switch status.State {
				case "failure":
					log.Printf("%s (failure)", status.Context)
				case "success":
					log.Printf("%s (success)", status.Context)
					matching[status.Context] = status
				}
			}
		}
		// If we match all contexts then we've found the commit
		if len(contexts) == len(matching) {
			return &commit, nil
		}
	}
	return nil, nil
}

func stringInSlice(str string, list []string) bool {
	for _, s := range list {
		if s == str {
			return true
		}
	}
	return false
}

// WaitForCI waits for commit in repo to pass CI contexts
func WaitForCI(token string, repo string, commit string, contexts []string, delay time.Duration, timeout time.Duration) error {
	start := time.Now()
	re := regexp.MustCompile("(.*)(/label=.*)")
	for time.Since(start) < timeout {
		log.Printf("Checking status for %s, %#v (%s)", repo, contexts, commit)
		statuses, err := Statuses("keybase", repo, commit, token)
		if err != nil {
			return err
		}
		matching := map[string]Status{}
		log.Println("\tStatuses:")
		for _, status := range statuses {
			log.Printf("\t%s (%s)", status.Context, status.State)
		}
		const successStatus = "success"
		const failureStatus = "failure"
		const errorStatus = "error"
		log.Println("\t")
		log.Println("\tMatch:")
		for _, status := range statuses {
			context := re.ReplaceAllString(status.Context, "$1")
			if stringInSlice(context, contexts) {
				switch status.State {
				case failureStatus, errorStatus:
					if matching[context].State != successStatus {
						log.Printf("\t%s (%s)", context, status.State)
						return fmt.Errorf("Failure in CI for %s", context)
					}
					log.Printf("\t%s (ignoring previous failure)", context)
				case successStatus:
					log.Printf("\t%s (success)", context)
					matching[context] = status
				}
			}
		}
		log.Println("\t")
		// If we match all contexts then we've passed
		if len(contexts) == len(matching) {
			return nil
		}

		log.Printf("Waiting %s", delay)
		time.Sleep(delay)
	}
	return fmt.Errorf("Timed out")
}
