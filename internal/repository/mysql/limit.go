package mysql

import (
	"context"
	"database/sql"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type LimitRepository struct {
	db *sql.DB
}

func NewLimitRepository(db *sql.DB) *LimitRepository {
	return &LimitRepository{db: db}
}

func (r *LimitRepository) GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error) {
	query := `SELECT id, customer_id, tenor_months, amount, used_amount, created_at, updated_at
		FROM limits WHERE customer_id = ? AND tenor_months = ?`
	row := r.db.QueryRowContext(ctx, query, customerID, tenor)
	return scanLimit(row)
}

func (r *LimitRepository) GetForUpdate(ctx context.Context, tx repository.Tx, customerID int64, tenor int) (*entity.Limit, error) {
	query := `SELECT id, customer_id, tenor_months, amount, used_amount, created_at, updated_at
		FROM limits WHERE customer_id = ? AND tenor_months = ? FOR UPDATE`
	row := tx.QueryRowContext(ctx, query, customerID, tenor)
	return scanLimit(row)
}

func (r *LimitRepository) ListForUpdateFromTenor(ctx context.Context, tx repository.Tx, customerID int64, tenor int) ([]entity.Limit, error) {
	query := `SELECT id, customer_id, tenor_months, amount, used_amount, created_at, updated_at
		FROM limits WHERE customer_id = ? AND tenor_months >= ? ORDER BY tenor_months FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query, customerID, tenor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var limits []entity.Limit
	for rows.Next() {
		var l entity.Limit
		if err := rows.Scan(&l.ID, &l.CustomerID, &l.TenorMonths, &l.Amount, &l.UsedAmount, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		limits = append(limits, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(limits) == 0 {
		return nil, repository.ErrNotFound
	}
	return limits, nil
}

func (r *LimitRepository) GetMaxUsedAtOrBelowTenor(ctx context.Context, customerID int64, tenor int) (int64, error) {
	query := `SELECT used_amount FROM limits WHERE customer_id = ? AND tenor_months <= ? ORDER BY tenor_months DESC LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, customerID, tenor)
	var usedAmount int64
	if err := row.Scan(&usedAmount); err != nil {
		if err == sql.ErrNoRows {
			return 0, repository.ErrNotFound
		}
		return 0, err
	}
	return usedAmount, nil
}

func (r *LimitRepository) Upsert(ctx context.Context, customerID int64, tenor int, amount int64, usedAmount int64) error {
	query := `INSERT INTO limits (customer_id, tenor_months, amount, used_amount)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE amount = VALUES(amount)`
	_, err := r.db.ExecContext(ctx, query, customerID, tenor, amount, usedAmount)
	return err
}

func (r *LimitRepository) UpdateUsed(ctx context.Context, tx repository.Tx, limitID int64, usedAmount int64) error {
	query := `UPDATE limits SET used_amount = ? WHERE id = ?`
	res, err := tx.ExecContext(ctx, query, usedAmount, limitID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *LimitRepository) ListByCustomerID(ctx context.Context, customerID int64) ([]entity.Limit, error) {
	query := `SELECT id, customer_id, tenor_months, amount, used_amount, created_at, updated_at
		FROM limits WHERE customer_id = ? ORDER BY tenor_months`
	rows, err := r.db.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var limits []entity.Limit
	for rows.Next() {
		var l entity.Limit
		if err := rows.Scan(&l.ID, &l.CustomerID, &l.TenorMonths, &l.Amount, &l.UsedAmount, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		limits = append(limits, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return limits, nil
}

func scanLimit(row *sql.Row) (*entity.Limit, error) {
	var l entity.Limit
	if err := row.Scan(&l.ID, &l.CustomerID, &l.TenorMonths, &l.Amount, &l.UsedAmount, &l.CreatedAt, &l.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}
