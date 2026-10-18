package db_postgres

import "testing"

func TestPasswordHistoryWriterRequiresATransaction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil transaction accepted")
		}
	}()
	PasswordHistoryWriterInTx(nil)
}
