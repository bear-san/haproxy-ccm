package test

import (
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestListFrontend(t *testing.T) {
	t.Run("List Test", func(t *testing.T) {
		got, err := haproxy.ListFrontend()
		if err != nil {
			t.Errorf("ListFrontend() error = %v", err)
			return
		}

		if len(got) == 0 {
			t.Errorf("ListFrontend() got = %v, want > 0", len(got))
		}
	})
}
