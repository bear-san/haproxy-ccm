package test

import (
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestListBacktend(t *testing.T) {
	t.Run("List Test", func(t *testing.T) {
		got, err := haproxy.ListBackend()
		if err != nil {
			t.Errorf("ListBackend() error = %v", err)
			return
		}

		if len(got) == 0 {
			t.Errorf("ListBackend() got = %v, want > 0", len(got))
		}
	})
}
