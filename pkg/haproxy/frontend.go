package haproxy

import (
	"bytes"
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

func CreateFrontend(frontend Frontend, transaction *Transaction) error {
	reqBody, _ := json.Marshal(frontend)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/frontends", haproxyBaseUrl)
	if transaction != nil {
		url = fmt.Sprintf("%s?transaction_id=%s", url, transaction.Id)
	}

	req, _ := http.NewRequest("POST", url, &reqBodyBuffer)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	return nil
}

func DeleteFrontend(name string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/frontends/%s", haproxyBaseUrl, name)
	if transaction != nil {
		url = fmt.Sprintf("%s?transaction_id=%s", url, transaction.Id)
	}

	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	return nil

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
