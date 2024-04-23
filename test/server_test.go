package test

import (
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestListServer(t *testing.T) {
	t.Run("List Test", func(t *testing.T) {
		_, err := haproxy.ListServer("nginx-ingress-controller-http")
		if err != nil {
			t.Errorf("ListServer() error = %v", err)
			return
		}
	})
}
