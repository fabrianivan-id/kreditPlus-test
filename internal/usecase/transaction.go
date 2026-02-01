package usecase

import (
	"context"
	"database/sql"
	"strings"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type TransactionUsecase struct {
	txManager       repository.TxManager
	limitRepo       repository.LimitRepository
	transactionRepo repository.TransactionRepository
}

type CreateTransactionInput struct {
	ContractNumber    string
	CustomerID        int64
	TenorMonths       int
	AssetName         string
	Channel           string
	OTR               int64
	AdminFee          int64
	InstallmentAmount int64
	InterestAmount    int64
}

type CreateTransactionResult struct {
	Transaction   *entity.Transaction
	RemainingLimit int64
}

func NewTransactionUsecase(txManager repository.TxManager, limitRepo repository.LimitRepository, transactionRepo repository.TransactionRepository) *TransactionUsecase {
	return &TransactionUsecase{
		txManager:       txManager,
		limitRepo:       limitRepo,
		transactionRepo: transactionRepo,
	}
}

func (uc *TransactionUsecase) Create(ctx context.Context, input CreateTransactionInput) (*CreateTransactionResult, error) {
	if strings.TrimSpace(input.ContractNumber) == "" ||
		input.CustomerID <= 0 ||
		!isAllowedTenor(input.TenorMonths) ||
		strings.TrimSpace(input.AssetName) == "" ||
		!isAllowedChannel(input.Channel) ||
		input.OTR <= 0 ||
		input.AdminFee < 0 ||
		input.InstallmentAmount <= 0 ||
		input.InterestAmount < 0 {
		return nil, ErrInvalidInput
	}

	financed := input.OTR + input.AdminFee
	if financed <= 0 {
		return nil, ErrInvalidInput
	}

	tx, err := uc.txManager.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	limit, err := uc.limitRepo.GetForUpdate(ctx, tx, input.CustomerID, input.TenorMonths)
	if err != nil {
		return nil, err
	}
	if limit == nil {
		return nil, ErrNotFound
	}

	if limit.Remaining() < financed {
		return nil, ErrLimitExceeded
	}

	transaction := &entity.Transaction{
		ContractNumber:    strings.TrimSpace(input.ContractNumber),
		CustomerID:        input.CustomerID,
		TenorMonths:       input.TenorMonths,
		AssetName:         strings.TrimSpace(input.AssetName),
		Channel:           normalizeChannel(input.Channel),
		OTR:               input.OTR,
		AdminFee:          input.AdminFee,
		InstallmentAmount: input.InstallmentAmount,
		InterestAmount:    input.InterestAmount,
		FinancedAmount:    financed,
	}

	if err := uc.transactionRepo.Create(ctx, tx, transaction); err != nil {
		return nil, err
	}

	newUsed := limit.UsedAmount + financed
	if err := uc.limitRepo.UpdateUsed(ctx, tx, limit.ID, newUsed); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true

	return &CreateTransactionResult{
		Transaction:   transaction,
		RemainingLimit: limit.Amount - newUsed,
	}, nil
}

func isAllowedChannel(channel string) bool {
	switch normalizeChannel(channel) {
	case "ecommerce", "web", "dealer":
		return true
	default:
		return false
	}
}

func normalizeChannel(channel string) string {
	return strings.ToLower(strings.TrimSpace(channel))
}
