package database

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Transaction struct {
	DB *sql.DB
}

func NewTransaction(db *sql.DB) *Transaction {
	return &Transaction{
		DB: db,
	}
}

func (t *Transaction) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		tx, err := t.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		err = fn(tx)
		if err != nil {
			_ = tx.Rollback()
			if IsRetryable(err) && i < maxRetries-1 {
				continue
			}
			return err
		}
		err = tx.Commit()
		if err != nil {
			if IsRetryable(err) && i < maxRetries-1 {
				continue
			}
			return err
		}

		return nil
	}
	return fmt.Errorf("transaction errors after retries")
}
