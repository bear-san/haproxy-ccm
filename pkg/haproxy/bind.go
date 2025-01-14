package haproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListBind(frontend string) ([]Bind, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/binds", haproxyBaseUrl, frontend), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := []Bind{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func GetBind(name string, frontend string) (*Bind, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/binds/%s", haproxyBaseUrl, frontend, name), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := Bind{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func CreateBind(frontend string, bind Bind, transaction *Transaction) error {
	reqBody, _ := json.Marshal(bind)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/binds", haproxyBaseUrl, frontend)
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
		return fmt.Errorf("failed to create bind %s", string(errMsg))
	}

	return nil
}

func DeleteBind(name string, frontend string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v3/services/haproxy/configuration/frontends/%s/binds/%s", haproxyBaseUrl, frontend, name)
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
		return fmt.Errorf("failed to delete bind %s", string(errMsg))
	}

	return nil
}

type BindResult struct {
	Version int  `json:"_version"`
	Data    Bind `json:"data"`
}

type BindListResult struct {
	Version int    `json:"_version"`
	Data    []Bind `json:"data"`
}

type Bind struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
