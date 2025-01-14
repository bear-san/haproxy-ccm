package haproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListServer(backend string) ([]Server, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers", haproxyBaseUrl, backend), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := []Server{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetServer(name string, backend string) (*Server, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers/%s", haproxyBaseUrl, backend, name), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := Server{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func CreateServer(backend string, server Server, transaction *Transaction) error {
	reqBody, _ := json.Marshal(server)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers", haproxyBaseUrl, backend)
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
		return fmt.Errorf("failed to create server %s", string(errMsg))
	}

	return nil
}

func DeleteServer(name string, backend string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/backends/%s/servers/%s", haproxyBaseUrl, backend, name)
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
		return fmt.Errorf("failed to delete server %s", string(errMsg))
	}

	return nil

}

type ServerListResult struct {
	Version int      `json:"_version"`
	Data    []Server `json:"data"`
}

type ServerResult struct {
	Version int    `json:"_version"`
	Data    Server `json:"data"`
}

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
