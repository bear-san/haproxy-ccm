package test

import (
	"github.com/bear-san/haproxy-ccm/pkg/haproxy"
	"testing"
)

func TestDeleteTransaction(t *testing.T) {
	t.Run("Delete Transaction Test", func(t *testing.T) {
		transaction, err := haproxy.CreateTransaction()
		if err != nil {
			t.Errorf("CreateTransaction() error = %v", err)
			return
		}

		err = haproxy.DeleteTransaction(transaction.Id)
		if err != nil {
			t.Errorf("DeleteTransaction() error = %v", err)
			return
		}
	})
}
