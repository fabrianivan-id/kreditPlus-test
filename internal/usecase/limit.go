package usecase

import (
	"context"
	"errors"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type LimitUsecase struct {
	repo repository.LimitRepository
}

func NewLimitUsecase(repo repository.LimitRepository) *LimitUsecase {
	return &LimitUsecase{repo: repo}
}

func (uc *LimitUsecase) Set(ctx context.Context, customerID int64, tenor int, amount int64) error {
	if customerID <= 0 || amount <= 0 || !isAllowedTenor(tenor) {
		return ErrInvalidInput
	}

	current, err := uc.repo.GetByCustomerAndTenor(ctx, customerID, tenor)
	if err != nil {
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
	}
	var usedBaseline int64
	if current != nil {
		if amount < current.UsedAmount {
			return ErrConflict
		}
		usedBaseline = current.UsedAmount
	} else {
		usedBaseline, err = uc.repo.GetMaxUsedAtOrBelowTenor(ctx, customerID, tenor)
		if err != nil && !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if amount < usedBaseline {
			return ErrConflict
		}
	}

	return uc.repo.Upsert(ctx, customerID, tenor, amount, usedBaseline)
}

func (uc *LimitUsecase) List(ctx context.Context, customerID int64) ([]entity.Limit, error) {
	if customerID <= 0 {
		return nil, ErrInvalidInput
	}
	return uc.repo.ListByCustomerID(ctx, customerID)
}

func isAllowedTenor(tenor int) bool {
	switch tenor {
	case 1, 2, 3, 6:
		return true
	default:
		return false
	}
}
