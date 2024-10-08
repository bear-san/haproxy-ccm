package haproxy

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func GetVersion() (*int, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/v3/services/haproxy/configuration/version", haproxyBaseUrl), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", auth))

	client := &http.Client{}
	resp, _ := client.Do(req)

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	versionString := strings.TrimRight(string(result), "\n")
	version, err := strconv.Atoi(versionString)
	if err != nil {
		return nil, err
	}

	return &version, nil
}
