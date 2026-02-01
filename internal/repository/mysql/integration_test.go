//go:build integration
// +build integration

package mysql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"kreditplus-test/internal/entity"
)

func openTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func resetTables(t *testing.T, db *sql.DB) {
	_, err := db.Exec("SET FOREIGN_KEY_CHECKS=0")
	if err != nil {
		t.Fatalf("disable fk checks: %v", err)
	}
	for _, table := range []string{"transactions", "limits", "customers"} {
		if _, err := db.Exec("TRUNCATE TABLE " + table); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS=1")
	if err != nil {
		t.Fatalf("enable fk checks: %v", err)
	}
}

func seedCustomer(t *testing.T, repo *CustomerRepository) *entity.Customer {
	customer := &entity.Customer{
		NIK:            "1234567890123456",
		FullName:       "Budi",
		LegalName:      "Budi Santoso",
		BirthPlace:     "Jakarta",
		BirthDate:      time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
		Salary:         5_000_000,
		KTPPhotoURL:    "https://example.com/ktp.jpg",
		SelfiePhotoURL: "https://example.com/selfie.jpg",
	}
	if err := repo.Create(context.Background(), customer); err != nil {
		t.Fatalf("create customer: %v", err)
	}
	return customer
}

func TestCustomerRepositoryCreateAndGet(t *testing.T) {
	db := openTestDB(t)
	resetTables(t, db)

	repo := NewCustomerRepository(db)
	customer := seedCustomer(t, repo)

	got, err := repo.GetByID(context.Background(), customer.ID)
	if err != nil {
		t.Fatalf("get customer: %v", err)
	}
	if got.NIK != customer.NIK {
		t.Fatalf("expected nik %s, got %s", customer.NIK, got.NIK)
	}
}

func TestLimitRepositoryUpsertAndGet(t *testing.T) {
	db := openTestDB(t)
	resetTables(t, db)

	customerRepo := NewCustomerRepository(db)
	customer := seedCustomer(t, customerRepo)
	limitRepo := NewLimitRepository(db)

	if err := limitRepo.Upsert(context.Background(), customer.ID, 3, 500_000, 100_000); err != nil {
		t.Fatalf("upsert limit: %v", err)
	}

	limit, err := limitRepo.GetByCustomerAndTenor(context.Background(), customer.ID, 3)
	if err != nil {
		t.Fatalf("get limit: %v", err)
	}
	if limit.UsedAmount != 100_000 {
		t.Fatalf("expected used_amount 100000, got %d", limit.UsedAmount)
	}

	if err := limitRepo.Upsert(context.Background(), customer.ID, 3, 600_000, 0); err != nil {
		t.Fatalf("upsert limit update: %v", err)
	}

	limit, err = limitRepo.GetByCustomerAndTenor(context.Background(), customer.ID, 3)
	if err != nil {
		t.Fatalf("get limit after update: %v", err)
	}
	if limit.Amount != 600_000 {
		t.Fatalf("expected amount 600000, got %d", limit.Amount)
	}
	if limit.UsedAmount != 100_000 {
		t.Fatalf("expected used_amount unchanged, got %d", limit.UsedAmount)
	}
}

func TestLimitRepositoryGetMaxUsedAtOrBelowTenor(t *testing.T) {
	db := openTestDB(t)
	resetTables(t, db)

	customerRepo := NewCustomerRepository(db)
	customer := seedCustomer(t, customerRepo)
	limitRepo := NewLimitRepository(db)

	_ = limitRepo.Upsert(context.Background(), customer.ID, 1, 100_000, 10_000)
	_ = limitRepo.Upsert(context.Background(), customer.ID, 3, 300_000, 50_000)
	_ = limitRepo.Upsert(context.Background(), customer.ID, 6, 600_000, 100_000)

	used, err := limitRepo.GetMaxUsedAtOrBelowTenor(context.Background(), customer.ID, 3)
	if err != nil {
		t.Fatalf("get max used: %v", err)
	}
	if used != 50_000 {
		t.Fatalf("expected used 50000, got %d", used)
	}
}

func TestLimitRepositoryListForUpdateAndUpdateUsed(t *testing.T) {
	db := openTestDB(t)
	resetTables(t, db)

	customerRepo := NewCustomerRepository(db)
	customer := seedCustomer(t, customerRepo)
	limitRepo := NewLimitRepository(db)

	_ = limitRepo.Upsert(context.Background(), customer.ID, 3, 300_000, 10_000)
	_ = limitRepo.Upsert(context.Background(), customer.ID, 6, 600_000, 10_000)

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	limits, err := limitRepo.ListForUpdateFromTenor(context.Background(), tx, customer.ID, 3)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("list for update: %v", err)
	}
	for _, l := range limits {
		if err := limitRepo.UpdateUsed(context.Background(), tx, l.ID, l.UsedAmount+5_000); err != nil {
			_ = tx.Rollback()
			t.Fatalf("update used: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	limit, err := limitRepo.GetByCustomerAndTenor(context.Background(), customer.ID, 3)
	if err != nil {
		t.Fatalf("get limit after update: %v", err)
	}
	if limit.UsedAmount != 15_000 {
		t.Fatalf("expected used_amount 15000, got %d", limit.UsedAmount)
	}
}

func TestTransactionRepositoryCreate(t *testing.T) {
	db := openTestDB(t)
	resetTables(t, db)

	customerRepo := NewCustomerRepository(db)
	customer := seedCustomer(t, customerRepo)
	transactionRepo := NewTransactionRepository()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	transaction := &entity.Transaction{
		ContractNumber:    "CN-TEST",
		CustomerID:        customer.ID,
		TenorMonths:       3,
		AssetName:         "Motor",
		Channel:           "dealer",
		OTR:               200_000,
		AdminFee:          10_000,
		InstallmentAmount: 20_000,
		InterestAmount:    5_000,
		FinancedAmount:    210_000,
	}

	if err := transactionRepo.Create(context.Background(), tx, transaction); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create transaction: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	row := db.QueryRow("SELECT contract_number FROM transactions WHERE id = ?", transaction.ID)
	var contract string
	if err := row.Scan(&contract); err != nil {
		t.Fatalf("query transaction: %v", err)
	}
	if contract != transaction.ContractNumber {
		t.Fatalf("expected contract %s, got %s", transaction.ContractNumber, contract)
	}
}
