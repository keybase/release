package winbuild

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	buildNumAPIUrl = "https://keybase.io/_/api/1.0/pkg/build_number.json"
)

type buildNoResponse struct {
	Status struct {
		Code int    `json:"code"`
		Name string `json:"name"`
	} `json:"status"`
	BuildNumber int `json:"build_number"`
}

func GetNextBuildNumber(keybaseToken string, version string) error {

	form := url.Values{}
	form.Set("version", version)
	form.Add("bot_id", "1")
	form.Add("platform", "1")
	req, err := http.NewRequest("POST", buildNumAPIUrl, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return fmt.Errorf("newrequest failed, %v", err)
	}
	req.Header.Add("X-keybase-admin-token", keybaseToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed, %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("body err, %v", err)
	}

	var reply buildNoResponse
	if err := json.Unmarshal(body, &reply); err != nil {
		return fmt.Errorf("json reply err, %v", err)
	}

	if reply.Status.Code != 0 {
		return fmt.Errorf("Server returned failure, %s", body)
	}

	fmt.Printf("%d\n", reply.BuildNumber)
	return nil
}
