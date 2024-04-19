package haproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListBind(frontend string) ([]Bind, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v2/services/haproxy/configuration/binds?frontend=%s", haproxyBaseUrl, frontend), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	result := BindResult{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

type BindResult struct {
	Version int    `json:"_version"`
	Data    []Bind `json:"data"`
}

type Bind struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
