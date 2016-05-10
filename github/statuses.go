// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package github

import "fmt"

// Status defines a git commit on Github
type Status struct {
	State   string `json:"state"`
	Context string `json:"context"`
}

const (
	statusesListPath = "/repos/%s/%s/commits/%s/statuses"
)

// Statuses lists statuses for a git commit
func Statuses(user, repo, sha, token string) ([]Status, error) {
	url, err := githubURL(GithubAPIURL, token)
	if err != nil {
		return nil, err
	}
	url.Path = fmt.Sprintf(statusesListPath, user, repo, sha)
	var statuses []Status
	if err = Get(url.String(), &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}
