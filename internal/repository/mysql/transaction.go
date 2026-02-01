package mysql

import (
	"context"

	"github.com/go-sql-driver/mysql"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type TransactionRepository struct{}

func NewTransactionRepository() *TransactionRepository {
	return &TransactionRepository{}
}

func (r *TransactionRepository) Create(ctx context.Context, tx repository.Tx, t *entity.Transaction) error {
	query := `INSERT INTO transactions
		(contract_number, customer_id, tenor_months, asset_name, channel, otr, admin_fee, installment_amount, interest_amount, financed_amount)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(ctx, query,
		t.ContractNumber,
		t.CustomerID,
		t.TenorMonths,
		t.AssetName,
		t.Channel,
		t.OTR,
		t.AdminFee,
		t.InstallmentAmount,
		t.InterestAmount,
		t.FinancedAmount,
	)
	if err != nil {
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			return repository.ErrDuplicate
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return nil
}
