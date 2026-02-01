package repository

import (
	"context"
	"database/sql"

	"kreditplus-test/internal/entity"
)

type Tx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	Commit() error
	Rollback() error
}

type TxManager interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
}

type CustomerRepository interface {
	Create(ctx context.Context, customer *entity.Customer) error
	GetByID(ctx context.Context, id int64) (*entity.Customer, error)
}

type LimitRepository interface {
	GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error)
	GetForUpdate(ctx context.Context, tx Tx, customerID int64, tenor int) (*entity.Limit, error)
	Upsert(ctx context.Context, customerID int64, tenor int, amount int64) error
	UpdateUsed(ctx context.Context, tx Tx, limitID int64, usedAmount int64) error
	ListByCustomerID(ctx context.Context, customerID int64) ([]entity.Limit, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, tx Tx, t *entity.Transaction) error
}
