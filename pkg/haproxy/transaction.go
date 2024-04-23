package haproxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func CreateTransaction() (*Transaction, error) {
	v, err := GetVersion()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/v2/services/haproxy/transactions?version=%d", haproxyBaseUrl, *v), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusCreated {
		errMsg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create transaction %s", string(errMsg))
	}

	resultText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := Transaction{}
	err = json.Unmarshal(resultText, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func CommitTransaction(transactionId string) error {
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/v2/services/haproxy/transactions/%s", haproxyBaseUrl, transactionId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to commit transaction %s %s", transactionId, string(errMsg))
	}

	return nil
}

func ListTransactions() ([]Transaction, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v2/services/haproxy/transactions", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)

	var result []Transaction
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func DeleteTransaction(transactionId string) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v2/services/haproxy/transactions/%s", haproxyBaseUrl, transactionId), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		errMsg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete transaction %s %s", transactionId, string(errMsg))
	}

	return nil
}

type Transaction struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}
