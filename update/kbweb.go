// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package update

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	kbwebAPIUrl = "https://api-0.core.keybaseapi.com"
)

const apiCa = `-----BEGIN CERTIFICATE-----
MIIGIzCCBAugAwIBAgIJAPzhpcIBaOeNMA0GCSqGSIb3DQEBCwUAMIGMMQswCQYD
VQQGEwJVUzELMAkGA1UECBMCTlkxETAPBgNVBAcTCE5ldyBZb3JrMRQwEgYDVQQK
EwtLZXliYXNlIExMQzEXMBUGA1UECxMOQ2VydCBBdXRob3JpdHkxLjAsBgNVBAMM
JWtleWJhc2UuaW8vZW1haWxBZGRyZXNzPWNhQGtleWJhc2UuaW8wIBcNMjMxMjMx
MTkwMzE5WhgPNjAyMzEyMzExOTAzMTlaMIGMMQswCQYDVQQGEwJVUzELMAkGA1UE
CBMCTlkxETAPBgNVBAcTCE5ldyBZb3JrMRQwEgYDVQQKEwtLZXliYXNlIExMQzEX
MBUGA1UECxMOQ2VydCBBdXRob3JpdHkxLjAsBgNVBAMMJWtleWJhc2UuaW8vZW1h
aWxBZGRyZXNzPWNhQGtleWJhc2UuaW8wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAw
ggIKAoICAQDewsDpkby46+aUW8UtUg5RGZxCtnIwUptW739N4OJ6aWzfDf8nNVN2
4P7sqJSL1HtBwJb9XVmlF5N+6ebut8AKInV+kiSNJCuCy8oMuCEjPEhLkUwjy616
3mnpC24mFoDCaZefzFfTkW+pY1utxdF2kviCgV2KA+wUrbGFNSJZq0syy16hKEjv
7OauCTHvkt4swPRsva45/zsmM7NtjzHaxQhksbA+gBPIbxZLfx7LoqQnFGMCEben
45NgSNhKuwC1ADoiZt4Ol9Ico4HwcXedWn/8RvgcSISxbAFFtBe8BaHcNgsa6QVb
TCI7QdUKhZj5scv8yprQ11EY6UuxsvhnikuuGoqBINTy6Zf1i41FFoHQ/mdOTPJT
prEerOr33QZ6n8jrZuOwF1hin4ONI8rjeZdGt9YmXY1NyXzEoDJ+w5b72FD2/ArS
2lKJw3F9i5RmzQGF+NJn9NzpnURF2BRhGJdO2iGX5JEDYiBkyWgcKWVUw2MSNeGC
68eAsA6ty7KFUG6mJRAZQdC+QyyvVTPxU80MU4l53C5xFTYBpHzzVuSedJt2z37M
0uy9QVX4ErtB2e39aQWlgvvysbBjjuayL06h13Hp8/J6DeqQkYzpzCf9ujLD2VB6
V5gOryTIl2LEgDG0CyQ3NE8nicO7aLNN8HJCgzx6nABZuhz+A0U5swIDAQABo4GD
MIGAMA4GA1UdDwEB/wQEAwIChDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYBBQUH
AwEwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQURqpATOw1gVVrzlqqFKbkfaKX
vwowHwYDVR0jBBgwFoAURqpATOw1gVVrzlqqFKbkfaKXvwowDQYJKoZIhvcNAQEL
BQADggIBALjuBecPwt0XJ6rpygOt9r1O6Oyj6WshzD2OvsK/RoHCJLjI32V8xYt3
YubUdFucy5m6dUEeTo6LwDd/7UpX+7NImQdRssHk6GynJJ7Sd2Jqvzlh6t+xFJHG
WqRt/u48T9Bm7pw1Z79QAXXi1L9DnPz8nMzu6gVTS2dzG4FAjXwzKsYV6mLoQW0L
adLKQELboM5hCauSILncD9ujWBZduFr7o4eHrRaZ4FiZ/46nGn/lqDhFTtgvSL53
+thrAiQCVv7sGkg8Niu3WTuJtIDlXzjGFuGli/l9KI9Dnr+RBe1kileQ99VZmayA
PgVFzkicAEd5ZzGnADGWAW0nSA8tOxAyo3qnnJ6Z1e2mNflmGv6+cryIkksfDu7A
oQuFQW0E3wEDmBXFHAGWgNKZQ05nxPY6zDm3FQCzS3v6CZuyJ8iwpDTKZYp4azPb
WLef0IJCGB62/+6YwD3bUunFq6jUR/vCgc5WRrLQd4LAbrrrP8SaLNPIlapZkYIU
Ba88Cg+nfTa7s0ETEJDNV+UyEoZbAMhcjCbua+aMx66WA+iinmZ++ilXxlBPNyFM
XNpVqc8i9YuN5ASXKwR0nna/vFyr2sFYhV/Q+QIBUh6bwZEFF9f3qtgxi908ZSEC
ip88muP7dUJ5jR/XrBLdYqrnMFym5dyHN7AjBdTwjSkTtFKHjAxb
-----END CERTIFICATE-----`

type kbwebClient struct {
	http *http.Client
}

type APIResponseWrapper interface {
	StatusCode() int
}

type AppResponseBase struct {
	Status struct {
		Code int
		Desc string
	}
}

func (s *AppResponseBase) StatusCode() int {
	return s.Status.Code
}

// newKbwebClient constructs a Client
func newKbwebClient() (*kbwebClient, error) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(apiCa))
	if !ok {
		return nil, fmt.Errorf("Could not read CA for keybase.io")
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: certPool},
		},
	}
	return &kbwebClient{http: client}, nil
}

func (client *kbwebClient) post(keybaseToken string, path string, data []byte, response APIResponseWrapper) error {
	req, err := http.NewRequest("POST", kbwebAPIUrl+path, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("newrequest failed, %v", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-keybase-admin-token", keybaseToken)
	resp, err := client.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed, %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body err, %v", err)
	}

	if response == nil {
		response = new(AppResponseBase)
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("json reply err, %v", err)
	}

	if response.StatusCode() != 0 {
		return fmt.Errorf("Server returned failure, %s", body)
	}

	fmt.Printf("Success.\n")
	return nil
}

type announceBuildArgs struct {
	VersionA string `json:"version_a"`
	VersionB string `json:"version_b"`
	Platform string `json:"platform"`
}

// AnnounceBuild tells the API server about the existence of a new build.
// It does not enroll it in smoke testing.
func AnnounceBuild(keybaseToken string, buildA string, buildB string, platform string) error {
	client, err := newKbwebClient()
	if err != nil {
		return fmt.Errorf("client create failed, %v", err)
	}
	args := &announceBuildArgs{
		VersionA: buildA,
		VersionB: buildB,
		Platform: platform,
	}
	jsonStr, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("json marshal err, %v", err)
	}
	var data = jsonStr
	return client.post(keybaseToken, "/_/api/1.0/pkg/add_build.json", data, nil)
}

type promoteBuildArgs struct {
	VersionA string `json:"version_a"`
	Platform string `json:"platform"`
}

type promoteBuildResponse struct {
	AppResponseBase
	ReleaseTimeMs int64 `json:"release_time"`
}

// KBWebPromote tells the API server that a new build is promoted.
func KBWebPromote(keybaseToken string, buildA string, platform string, dryRun bool) (releaseTime time.Time, err error) {
	client, err := newKbwebClient()
	if err != nil {
		return releaseTime, fmt.Errorf("client create failed, %v", err)
	}
	args := &promoteBuildArgs{
		VersionA: buildA,
		Platform: platform,
	}
	jsonStr, err := json.Marshal(args)
	if err != nil {
		return releaseTime, fmt.Errorf("json marshal err, %v", err)
	}
	var data = jsonStr
	var response promoteBuildResponse
	if dryRun {
		log.Printf("DRYRUN: Would post %s\n", data)
		return releaseTime, nil
	}
	err = client.post(keybaseToken, "/_/api/1.0/pkg/set_released.json", data, &response)
	if err != nil {
		return releaseTime, err
	}
	releaseTime = time.Unix(0, response.ReleaseTimeMs*int64(time.Millisecond))
	log.Printf("Release time set to %v for build %v", releaseTime, buildA)
	return releaseTime, nil
}

type setBuildInTestingArgs struct {
	VersionA   string `json:"version_a"`
	Platform   string `json:"platform"`
	InTesting  string `json:"in_testing"`
	MaxTesters int    `json:"max_testers"`
}

// SetBuildInTesting tells the API server to enroll or unenroll a build in smoke testing.
func SetBuildInTesting(keybaseToken string, buildA string, platform string, inTesting string, maxTesters int) error {
	client, err := newKbwebClient()
	if err != nil {
		return fmt.Errorf("client create failed, %v", err)
	}
	args := &setBuildInTestingArgs{
		VersionA:   buildA,
		Platform:   platform,
		InTesting:  inTesting,
		MaxTesters: maxTesters,
	}
	jsonStr, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("json marshal err: %v", err)
	}
	var data = jsonStr
	return client.post(keybaseToken, "/_/api/1.0/pkg/set_in_testing.json", data, nil)
}
