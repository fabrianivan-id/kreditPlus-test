package usecase

import (
	"context"
	"database/sql"
	"testing"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type stubTx struct{}

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 1, nil }
func (stubResult) RowsAffected() (int64, error) { return 1, nil }

func (s *stubTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return stubResult{}, nil
}

func (s *stubTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return &sql.Row{}
}

func (s *stubTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (s *stubTx) Commit() error   { return nil }
func (s *stubTx) Rollback() error { return nil }

type stubTxManager struct {
	tx repository.Tx
}

func (m stubTxManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (repository.Tx, error) {
	return m.tx, nil
}

type stubLimitRepo struct {
	limit       *entity.Limit
	updatedUsed int64
}

func (s *stubLimitRepo) GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error) {
	return s.limit, nil
}

func (s *stubLimitRepo) GetForUpdate(ctx context.Context, tx repository.Tx, customerID int64, tenor int) (*entity.Limit, error) {
	return s.limit, nil
}

func (s *stubLimitRepo) Upsert(ctx context.Context, customerID int64, tenor int, amount int64) error {
	return nil
}

func (s *stubLimitRepo) UpdateUsed(ctx context.Context, tx repository.Tx, limitID int64, usedAmount int64) error {
	s.updatedUsed = usedAmount
	s.limit.UsedAmount = usedAmount
	return nil
}

func (s *stubLimitRepo) ListByCustomerID(ctx context.Context, customerID int64) ([]entity.Limit, error) {
	return nil, nil
}

type stubTransactionRepo struct {
	created *entity.Transaction
}

func (s *stubTransactionRepo) Create(ctx context.Context, tx repository.Tx, t *entity.Transaction) error {
	t.ID = 99
	s.created = t
	return nil
}

func TestCreateTransactionSuccess(t *testing.T) {
	limit := &entity.Limit{ID: 1, CustomerID: 10, TenorMonths: 3, Amount: 1_000_000, UsedAmount: 100_000}
	limitRepo := &stubLimitRepo{limit: limit}
	transactionRepo := &stubTransactionRepo{}

	uc := NewTransactionUsecase(stubTxManager{tx: &stubTx{}}, limitRepo, transactionRepo)

	result, err := uc.Create(context.Background(), CreateTransactionInput{
		ContractNumber:    "CN-001",
		CustomerID:        10,
		TenorMonths:       3,
		AssetName:         "Motor",
		Channel:           "dealer",
		OTR:               200_000,
		AdminFee:          10_000,
		InstallmentAmount: 20_000,
		InterestAmount:    5_000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Transaction.ID != 99 {
		t.Fatalf("expected transaction id set")
	}

	expectedUsed := int64(100_000 + 210_000)
	if limitRepo.updatedUsed != expectedUsed {
		t.Fatalf("expected used amount %d, got %d", expectedUsed, limitRepo.updatedUsed)
	}

	expectedRemaining := limit.Amount - expectedUsed
	if result.RemainingLimit != expectedRemaining {
		t.Fatalf("expected remaining %d, got %d", expectedRemaining, result.RemainingLimit)
	}
}

func TestCreateTransactionLimitExceeded(t *testing.T) {
	limit := &entity.Limit{ID: 1, CustomerID: 10, TenorMonths: 3, Amount: 100_000, UsedAmount: 90_000}
	limitRepo := &stubLimitRepo{limit: limit}
	transactionRepo := &stubTransactionRepo{}

	uc := NewTransactionUsecase(stubTxManager{tx: &stubTx{}}, limitRepo, transactionRepo)

	_, err := uc.Create(context.Background(), CreateTransactionInput{
		ContractNumber:    "CN-002",
		CustomerID:        10,
		TenorMonths:       3,
		AssetName:         "Motor",
		Channel:           "dealer",
		OTR:               20_000,
		AdminFee:          5_000,
		InstallmentAmount: 5_000,
		InterestAmount:    1_000,
	})

	if err == nil || err != ErrLimitExceeded {
		t.Fatalf("expected limit exceeded error")
	}
}
