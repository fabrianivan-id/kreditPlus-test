package usecase

import (
	"context"
	"strings"
	"time"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type CustomerUsecase struct {
	repo repository.CustomerRepository
}

type CreateCustomerInput struct {
	NIK            string
	FullName       string
	LegalName      string
	BirthPlace     string
	BirthDate      time.Time
	Salary         int64
	KTPPhotoURL    string
	SelfiePhotoURL string
}

func NewCustomerUsecase(repo repository.CustomerRepository) *CustomerUsecase {
	return &CustomerUsecase{repo: repo}
}

func (uc *CustomerUsecase) Create(ctx context.Context, input CreateCustomerInput) (*entity.Customer, error) {
	if strings.TrimSpace(input.NIK) == "" ||
		strings.TrimSpace(input.FullName) == "" ||
		strings.TrimSpace(input.LegalName) == "" ||
		strings.TrimSpace(input.BirthPlace) == "" ||
		input.BirthDate.IsZero() ||
		input.Salary <= 0 ||
		strings.TrimSpace(input.KTPPhotoURL) == "" ||
		strings.TrimSpace(input.SelfiePhotoURL) == "" {
		return nil, ErrInvalidInput
	}

	customer := &entity.Customer{
		NIK:            strings.TrimSpace(input.NIK),
		FullName:       strings.TrimSpace(input.FullName),
		LegalName:      strings.TrimSpace(input.LegalName),
		BirthPlace:     strings.TrimSpace(input.BirthPlace),
		BirthDate:      input.BirthDate,
		Salary:         input.Salary,
		KTPPhotoURL:    strings.TrimSpace(input.KTPPhotoURL),
		SelfiePhotoURL: strings.TrimSpace(input.SelfiePhotoURL),
	}

	if err := uc.repo.Create(ctx, customer); err != nil {
		return nil, err
	}

	return customer, nil
}
