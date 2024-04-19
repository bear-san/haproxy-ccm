package haproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListBackend() ([]Backend, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v2/services/haproxy/configuration/backends", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	result := BackendResult{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

type BackendResult struct {
	Version int       `json:"_version"`
	Data    []Backend `json:"data"`
}

type Backend struct {
	Balance struct {
		Algorithm string `json:"algorithm"`
	} `json:"balance"`
	Mode string `json:"mode"`
	Name string `json:"name"`
}