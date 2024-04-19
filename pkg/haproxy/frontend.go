package haproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListFrontend() ([]Frontend, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v2/services/haproxy/configuration/frontends", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	result := FrontendResult{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

type FrontendResult struct {
	Version int        `json:"_version"`
	Data    []Frontend `json:"data"`
}

type Frontend struct {
	DefaultBackend string `json:"default_backend"`
	Mode           string `json:"mode"`
	Name           string `json:"name"`
	Tcplog         bool   `json:"tcplog"`
}
