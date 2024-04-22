package test

import (
	"fmt"
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestGetVersion(t *testing.T) {
	t.Run("Version Test", func(t *testing.T) {
		got, err := haproxy.GetVersion()
		if err != nil {
			t.Errorf("ListServer() error = %v", err)
			return
		}

		fmt.Println("Version: ", *got)
	})
}
