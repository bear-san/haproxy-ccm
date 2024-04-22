package haproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func CreateTransaction() (*Transaction, error) {
	v, err := GetVersion()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v2/services/haproxy/transactions?version=%d", haproxyBaseUrl, v), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)

	result := Transaction{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func CommitTransaction(transactionId string) error {
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/v2/services/haproxy/transactions/%s", haproxyBaseUrl, transactionId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	_, _ = client.Do(req)

	return nil
}

type Transaction struct {
	Version string `json:"_version"`
	Id      string `json:"id"`
	Status  string `json:"status"`
}
