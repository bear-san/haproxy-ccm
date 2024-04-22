package haproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func CreateTransaction() (*Transaction, error) {
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v2/services/haproxy/transactions", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(resp.Body)

	result := Transaction{}
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func CommitTransaction(transactionId string) error {
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/v2/services/haproxy/transactions/%s", haproxyBaseUrl, transactionId), nil)
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

type Transaction struct {
	Version string `json:"_version"`
	Id      string `json:"id"`
	Status  string `json:"status"`
}
