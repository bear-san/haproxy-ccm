package haproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListBackend() ([]Backend, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/backends", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	result := []Backend{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetBackend(name string) (*Backend, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s", haproxyBaseUrl, name), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := BackendResult{}
	err := json.NewDecoder(resp.Body).Decode(&result)

	if err != nil {
		return nil, err
	}

	return &result.Data, nil

}

func CreateBackend(backend Backend, transaction *Transaction) error {
	reqBody, _ := json.Marshal(backend)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/backends", haproxyBaseUrl)
	if transaction != nil {
		url = fmt.Sprintf("%s?transaction_id=%s", url, transaction.Id)
	}

	req, _ := http.NewRequest("POST", url, &reqBodyBuffer)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create backend %s", string(errMsg))
	}

	return nil
}

func DeleteBackend(name string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s", haproxyBaseUrl, name)
	if transaction != nil {
		url = fmt.Sprintf("%s?transaction_id=%s", url, transaction.Id)
	}
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusAccepted {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete backend %s", string(errMsg))
	}

	return nil

}

type BackendResult struct {
	Version int     `json:"_version"`
	Data    Backend `json:"data"`
}

type BackendListResult struct {
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
