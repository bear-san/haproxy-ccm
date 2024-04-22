package haproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func ListServer(backend string) ([]Server, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v2/services/haproxy/configuration/servers?backend=%s", haproxyBaseUrl, backend), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := ServerResult{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Data, nil
}

func CreateServer(backend string, server Server, transaction *Transaction) error {
	reqBody, _ := json.Marshal(server)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/servers?backend=%s", haproxyBaseUrl, backend)
	if transaction != nil {
		url = fmt.Sprintf("%s&transaction_id=%s", url, transaction.Id)
	}

	req, _ := http.NewRequest("POST", url, &reqBodyBuffer)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	_, _ = client.Do(req)

	return nil
}

func DeleteServer(name string, backend string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/servers/%s?backend=%s", haproxyBaseUrl, name, backend)
	if transaction != nil {
		url = fmt.Sprintf("%s&transaction_id=%s", url, transaction.Id)
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

type ServerResult struct {
	Version int      `json:"_version"`
	Data    []Server `json:"data"`
}

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
