package test

import (
	"context"
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestListFrontend(t *testing.T) {
	t.Run("List Test", func(t *testing.T) {
		ctx := context.Background()
		got, err := haproxy.ListFrontend(ctx)
		if err != nil {
			t.Errorf("ListFrontend() error = %v", err)
			return
		}

		if len(got) == 0 {
			t.Errorf("ListFrontend() got = %v, want > 0", len(got))
		}
	})
}
