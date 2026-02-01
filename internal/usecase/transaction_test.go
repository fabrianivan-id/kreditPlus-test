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
	limits  []entity.Limit
	updated map[int64]int64
}

func (s *stubLimitRepo) GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error) {
	for i := range s.limits {
		if s.limits[i].CustomerID == customerID && s.limits[i].TenorMonths == tenor {
			return &s.limits[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (s *stubLimitRepo) GetForUpdate(ctx context.Context, tx repository.Tx, customerID int64, tenor int) (*entity.Limit, error) {
	return s.GetByCustomerAndTenor(ctx, customerID, tenor)
}

func (s *stubLimitRepo) ListForUpdateFromTenor(ctx context.Context, tx repository.Tx, customerID int64, tenor int) ([]entity.Limit, error) {
	var out []entity.Limit
	for _, l := range s.limits {
		if l.CustomerID == customerID && l.TenorMonths >= tenor {
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return nil, repository.ErrNotFound
	}
	return out, nil
}

func (s *stubLimitRepo) GetMaxUsedAtOrBelowTenor(ctx context.Context, customerID int64, tenor int) (int64, error) {
	var found bool
	var maxTenor int
	var used int64
	for _, l := range s.limits {
		if l.CustomerID == customerID && l.TenorMonths <= tenor && (!found || l.TenorMonths > maxTenor) {
			found = true
			maxTenor = l.TenorMonths
			used = l.UsedAmount
		}
	}
	if !found {
		return 0, repository.ErrNotFound
	}
	return used, nil
}

func (s *stubLimitRepo) Upsert(ctx context.Context, customerID int64, tenor int, amount int64, usedAmount int64) error {
	return nil
}

func (s *stubLimitRepo) UpdateUsed(ctx context.Context, tx repository.Tx, limitID int64, usedAmount int64) error {
	if s.updated == nil {
		s.updated = make(map[int64]int64)
	}
	s.updated[limitID] = usedAmount
	for i := range s.limits {
		if s.limits[i].ID == limitID {
			s.limits[i].UsedAmount = usedAmount
		}
	}
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
	limitRepo := &stubLimitRepo{limits: []entity.Limit{
		{ID: 1, CustomerID: 10, TenorMonths: 3, Amount: 1_000_000, UsedAmount: 100_000},
		{ID: 2, CustomerID: 10, TenorMonths: 6, Amount: 1_500_000, UsedAmount: 100_000},
	}}
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
	if limitRepo.updated[1] != expectedUsed || limitRepo.updated[2] != expectedUsed {
		t.Fatalf("expected used amount %d to be applied to all newer limits", expectedUsed)
	}

	expectedRemaining := int64(1_000_000 - expectedUsed)
	if result.RemainingLimit != expectedRemaining {
		t.Fatalf("expected remaining %d, got %d", expectedRemaining, result.RemainingLimit)
	}
}

func TestCreateTransactionLimitExceeded(t *testing.T) {
	limitRepo := &stubLimitRepo{limits: []entity.Limit{
		{ID: 1, CustomerID: 10, TenorMonths: 3, Amount: 500_000, UsedAmount: 100_000},
		{ID: 2, CustomerID: 10, TenorMonths: 6, Amount: 120_000, UsedAmount: 100_000},
	}}
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

func TestCreateTransactionMissingBaseTenor(t *testing.T) {
	limitRepo := &stubLimitRepo{limits: []entity.Limit{
		{ID: 2, CustomerID: 10, TenorMonths: 6, Amount: 1_000_000, UsedAmount: 0},
	}}
	transactionRepo := &stubTransactionRepo{}

	uc := NewTransactionUsecase(stubTxManager{tx: &stubTx{}}, limitRepo, transactionRepo)

	_, err := uc.Create(context.Background(), CreateTransactionInput{
		ContractNumber:    "CN-003",
		CustomerID:        10,
		TenorMonths:       3,
		AssetName:         "Motor",
		Channel:           "dealer",
		OTR:               20_000,
		AdminFee:          5_000,
		InstallmentAmount: 5_000,
		InterestAmount:    1_000,
	})

	if err == nil || err != ErrNotFound {
		t.Fatalf("expected not found error")
	}
}
