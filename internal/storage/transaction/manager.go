package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ai-product-assistant/internal/usecase"
)

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) (*Manager, error) {
	if db == nil {
		return nil, fmt.Errorf("transaction manager: db is required")
	}

	return &Manager{db: db}, nil
}

func (m *Manager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.WithinTransactionWithOptions(ctx, usecase.TxOptions{}, fn)
}

func (m *Manager) WithinTransactionWithOptions(ctx context.Context, opts usecase.TxOptions, fn func(ctx context.Context) error) error {
	var txOpts *sql.TxOptions
	if opts.Isolation != 0 || opts.ReadOnly {
		txOpts = &sql.TxOptions{
			Isolation: opts.Isolation,
			ReadOnly:  opts.ReadOnly,
		}
	}

	tx, err := m.db.BeginTx(ctx, txOpts)
	if err != nil {
		return fmt.Errorf("transaction manager: begin tx: %w", err)
	}

	txCtx := withTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("transaction manager: rollback tx: %w", rollbackErr))
		}

		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transaction manager: commit tx: %w", err)
	}

	return nil
}
