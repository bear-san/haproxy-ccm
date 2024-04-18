package haproxy

import "os"

var haproxyBaseUrl string
var auth string

func init() {
	haproxyBaseUrl = os.Getenv("HAPROXY_BASE_URL")
	auth = os.Getenv("HAPROXY_AUTH")
}
