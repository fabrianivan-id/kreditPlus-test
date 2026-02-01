package usecase

import (
	"context"
	"testing"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type stubLimitOnlyRepo struct {
	limit          *entity.Limit
	errGet         error
	upsertCalled   bool
	upsertCustomer int64
	upsertTenor    int
	upsertAmount   int64
}

func (s *stubLimitOnlyRepo) GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error) {
	if s.errGet != nil {
		return nil, s.errGet
	}
	return s.limit, nil
}

func (s *stubLimitOnlyRepo) GetForUpdate(ctx context.Context, tx repository.Tx, customerID int64, tenor int) (*entity.Limit, error) {
	return s.limit, nil
}

func (s *stubLimitOnlyRepo) Upsert(ctx context.Context, customerID int64, tenor int, amount int64) error {
	s.upsertCalled = true
	s.upsertCustomer = customerID
	s.upsertTenor = tenor
	s.upsertAmount = amount
	return nil
}

func (s *stubLimitOnlyRepo) UpdateUsed(ctx context.Context, tx repository.Tx, limitID int64, usedAmount int64) error {
	return nil
}

func (s *stubLimitOnlyRepo) ListByCustomerID(ctx context.Context, customerID int64) ([]entity.Limit, error) {
	return nil, nil
}

func TestSetLimitRejectsWhenBelowUsed(t *testing.T) {
	repo := &stubLimitOnlyRepo{limit: &entity.Limit{ID: 1, CustomerID: 10, TenorMonths: 3, Amount: 100_000, UsedAmount: 90_000}}
	uc := NewLimitUsecase(repo)

	err := uc.Set(context.Background(), 10, 3, 80_000)
	if err == nil || err != ErrConflict {
		t.Fatalf("expected conflict error")
	}
}

func TestSetLimitAllowsWhenNotFound(t *testing.T) {
	repo := &stubLimitOnlyRepo{errGet: repository.ErrNotFound}
	uc := NewLimitUsecase(repo)

	if err := uc.Set(context.Background(), 10, 3, 500_000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.upsertCalled {
		t.Fatalf("expected upsert to be called")
	}
}
