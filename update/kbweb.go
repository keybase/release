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
MIIGmzCCBIOgAwIBAgIJAPzhpcIBaOeNMA0GCSqGSIb3DQEBBQUAMIGPMQswCQYD
VQQGEwJVUzELMAkGA1UECBMCTlkxETAPBgNVBAcTCE5ldyBZb3JrMRQwEgYDVQQK
EwtLZXliYXNlIExMQzEXMBUGA1UECxMOQ2VydCBBdXRob3JpdHkxEzARBgNVBAMT
CmtleWJhc2UuaW8xHDAaBgkqhkiG9w0BCQEWDWNhQGtleWJhc2UuaW8wHhcNMTQw
MTAyMTY0MjMzWhcNMjMxMjMxMTY0MjMzWjCBjzELMAkGA1UEBhMCVVMxCzAJBgNV
BAgTAk5ZMREwDwYDVQQHEwhOZXcgWW9yazEUMBIGA1UEChMLS2V5YmFzZSBMTEMx
FzAVBgNVBAsTDkNlcnQgQXV0aG9yaXR5MRMwEQYDVQQDEwprZXliYXNlLmlvMRww
GgYJKoZIhvcNAQkBFg1jYUBrZXliYXNlLmlvMIICIjANBgkqhkiG9w0BAQEFAAOC
Ag8AMIICCgKCAgEA3sLA6ZG8uOvmlFvFLVIOURmcQrZyMFKbVu9/TeDiemls3w3/
JzVTduD+7KiUi9R7QcCW/V1ZpReTfunm7rfACiJ1fpIkjSQrgsvKDLghIzxIS5FM
I8utet5p6QtuJhaAwmmXn8xX05FvqWNbrcXRdpL4goFdigPsFK2xhTUiWatLMste
oShI7+zmrgkx75LeLMD0bL2uOf87JjOzbY8x2sUIZLGwPoATyG8WS38ey6KkJxRj
AhG3p+OTYEjYSrsAtQA6ImbeDpfSHKOB8HF3nVp//Eb4HEiEsWwBRbQXvAWh3DYL
GukFW0wiO0HVCoWY+bHL/Mqa0NdRGOlLsbL4Z4pLrhqKgSDU8umX9YuNRRaB0P5n
TkzyU6axHqzq990Gep/I62bjsBdYYp+DjSPK43mXRrfWJl2NTcl8xKAyfsOW+9hQ
9vwK0tpSicNxfYuUZs0BhfjSZ/Tc6Z1ERdgUYRiXTtohl+SRA2IgZMloHCllVMNj
EjXhguvHgLAOrcuyhVBupiUQGUHQvkMsr1Uz8VPNDFOJedwucRU2AaR881bknnSb
ds9+zNLsvUFV+BK7Qdnt/WkFpYL78rGwY47msi9Ooddx6fPyeg3qkJGM6cwn/boy
w9lQeleYDq8kyJdixIAxtAskNzRPJ4nDu2izTfByQoM8epwAWboc/gNFObMCAwEA
AaOB9zCB9DAdBgNVHQ4EFgQURqpATOw1gVVrzlqqFKbkfaKXvwowgcQGA1UdIwSB
vDCBuYAURqpATOw1gVVrzlqqFKbkfaKXvwqhgZWkgZIwgY8xCzAJBgNVBAYTAlVT
MQswCQYDVQQIEwJOWTERMA8GA1UEBxMITmV3IFlvcmsxFDASBgNVBAoTC0tleWJh
c2UgTExDMRcwFQYDVQQLEw5DZXJ0IEF1dGhvcml0eTETMBEGA1UEAxMKa2V5YmFz
ZS5pbzEcMBoGCSqGSIb3DQEJARYNY2FAa2V5YmFzZS5pb4IJAPzhpcIBaOeNMAwG
A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggIBAA3Z5FIhulYghMuHdcHYTYWc
7xT5WD4hXQ0WALZs4p5Y+b2Af54o6v1wUE1Au97FORq5CsFXX/kGl/JzzTimeucn
YJwGuXMpilrlHCBAL5/lSQjA7qbYIolQ3SB9ON+LYuF1jKB9k8SqNp7qzucxT3tO
b8ZMDEPNsseC7NE2uwNtcW3yrTh6WZnSqg/jwswiWjHYDdG7U8FjMYlRol3wPux2
PizGbSgiR+ztI2OthxtxNWMrT9XKxNQTpcxOXnLuhiSwqH8PoY17ecP8VPpaa0K6
zym0zSkbroqydazaxcXRk3eSlc02Ktk7HzRzuqQQXhRMkxVnHbFHgGsz03L533pm
mlIEgBMggZkHwNvs1LR7f3v2McdKulDH7Mv8yyfguuQ5Jxxt7RJhUuqSudbEhoaM
6jAJwBkMFxsV2YnyFEd3eZ/qBYPf7TYHhyzmHW6WkSypGqSnXd4gYpJ8o7LxSf4F
inLjxRD+H9Xn1UVXWLM0gaBB7zZcXd2zjMpRsWgezf5IR5vyakJsc7fxzgor3Qeq
Ri6LvdEkhhFVl5rHMQBwNOPngySrq8cs/ikTLTfQVTYXXA4Ba1YyiMOlfaR1LhKw
If1AkUV0tfCTNRZ01EotKSK77+o+k214n+BAu+7mO+9B5Kb7lMFQcuWCHXKYB2Md
cT7Yh09F0QpFUd0ymEfv
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
