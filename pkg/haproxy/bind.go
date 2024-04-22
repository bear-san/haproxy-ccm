package haproxy

import (
	"bytes"
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

func CreateBind(frontend string, bind Bind, transaction *Transaction) error {
	reqBody, _ := json.Marshal(bind)
	reqBodyBuffer := bytes.Buffer{}
	reqBodyBuffer.Write(reqBody)

	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/binds?frontend=%s", haproxyBaseUrl, frontend)
	if transaction != nil {
		url = fmt.Sprintf("%s&transaction_id=%s", url, transaction.Id)
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

func DeleteBind(name string, frontend string, transaction *Transaction) error {
	url := fmt.Sprintf("%s/v2/services/haproxy/configuration/binds/%s?frontend=%s", haproxyBaseUrl, name, frontend)
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

type BindResult struct {
	Version int    `json:"_version"`
	Data    []Bind `json:"data"`
}

type Bind struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}
