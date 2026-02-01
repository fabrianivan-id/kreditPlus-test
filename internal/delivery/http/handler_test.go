package http

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
	"kreditplus-test/internal/usecase"
	"kreditplus-test/pkg/config"
)

type memCustomerRepo struct {
	nextID   int64
	customers map[int64]*entity.Customer
}

func (r *memCustomerRepo) Create(ctx context.Context, customer *entity.Customer) error {
	if r.customers == nil {
		r.customers = make(map[int64]*entity.Customer)
	}
	r.nextID++
	customer.ID = r.nextID
	r.customers[customer.ID] = customer
	return nil
}

func (r *memCustomerRepo) GetByID(ctx context.Context, id int64) (*entity.Customer, error) {
	if customer, ok := r.customers[id]; ok {
		return customer, nil
	}
	return nil, repository.ErrNotFound
}

type memLimitRepo struct {
	limits map[string]*entity.Limit
	nextID int64
}

func (r *memLimitRepo) GetByCustomerAndTenor(ctx context.Context, customerID int64, tenor int) (*entity.Limit, error) {
	key := limitKey(customerID, tenor)
	if limit, ok := r.limits[key]; ok {
		return limit, nil
	}
	return nil, repository.ErrNotFound
}

func (r *memLimitRepo) GetForUpdate(ctx context.Context, tx repository.Tx, customerID int64, tenor int) (*entity.Limit, error) {
	return r.GetByCustomerAndTenor(ctx, customerID, tenor)
}

func (r *memLimitRepo) ListForUpdateFromTenor(ctx context.Context, tx repository.Tx, customerID int64, tenor int) ([]entity.Limit, error) {
	var out []entity.Limit
	for _, limit := range r.limits {
		if limit.CustomerID == customerID && limit.TenorMonths >= tenor {
			out = append(out, *limit)
		}
	}
	if len(out) == 0 {
		return nil, repository.ErrNotFound
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TenorMonths < out[j].TenorMonths })
	return out, nil
}

func (r *memLimitRepo) GetMaxUsedAtOrBelowTenor(ctx context.Context, customerID int64, tenor int) (int64, error) {
	found := false
	maxTenor := 0
	var used int64
	for _, limit := range r.limits {
		if limit.CustomerID == customerID && limit.TenorMonths <= tenor {
			if !found || limit.TenorMonths > maxTenor {
				found = true
				maxTenor = limit.TenorMonths
				used = limit.UsedAmount
			}
		}
	}
	if !found {
		return 0, repository.ErrNotFound
	}
	return used, nil
}

func (r *memLimitRepo) Upsert(ctx context.Context, customerID int64, tenor int, amount int64, usedAmount int64) error {
	if r.limits == nil {
		r.limits = make(map[string]*entity.Limit)
	}
	key := limitKey(customerID, tenor)
	if limit, ok := r.limits[key]; ok {
		limit.Amount = amount
		return nil
	}
	r.nextID++
	r.limits[key] = &entity.Limit{
		ID:          r.nextID,
		CustomerID:  customerID,
		TenorMonths: tenor,
		Amount:      amount,
		UsedAmount:  usedAmount,
	}
	return nil
}

func (r *memLimitRepo) UpdateUsed(ctx context.Context, tx repository.Tx, limitID int64, usedAmount int64) error {
	for _, limit := range r.limits {
		if limit.ID == limitID {
			limit.UsedAmount = usedAmount
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *memLimitRepo) ListByCustomerID(ctx context.Context, customerID int64) ([]entity.Limit, error) {
	var out []entity.Limit
	for _, limit := range r.limits {
		if limit.CustomerID == customerID {
			out = append(out, *limit)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TenorMonths < out[j].TenorMonths })
	return out, nil
}

func limitKey(customerID int64, tenor int) string {
	return fmt.Sprintf("%d:%d", customerID, tenor)
}

type memTransactionRepo struct {
	nextID       int64
	transactions []*entity.Transaction
}

func (r *memTransactionRepo) Create(ctx context.Context, tx repository.Tx, t *entity.Transaction) error {
	r.nextID++
	t.ID = r.nextID
	r.transactions = append(r.transactions, t)
	return nil
}

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

type testEnv struct {
	handler         http.Handler
	customerRepo    *memCustomerRepo
	limitRepo       *memLimitRepo
	transactionRepo *memTransactionRepo
	apiKey          string
}

func newTestEnv() *testEnv {
	customerRepo := &memCustomerRepo{}
	limitRepo := &memLimitRepo{limits: make(map[string]*entity.Limit)}
	transactionRepo := &memTransactionRepo{}

	customerUC := usecase.NewCustomerUsecase(customerRepo)
	limitUC := usecase.NewLimitUsecase(limitRepo)
	transactionUC := usecase.NewTransactionUsecase(stubTxManager{tx: &stubTx{}}, limitRepo, transactionRepo)

	cfg := config.Config{
		APIKey:          "test-key",
		RateLimitPerMin: 1000,
		BodyLimitBytes:  1 << 20,
	}

	logger := log.New(io.Discard, "", 0)
	h := NewHandler(customerUC, limitUC, transactionUC, nil)
	mw := NewMiddleware(cfg, logger)

	return &testEnv{
		handler:         h.Routes(mw),
		customerRepo:    customerRepo,
		limitRepo:       limitRepo,
		transactionRepo: transactionRepo,
		apiKey:          cfg.APIKey,
	}
}

func TestCreateCustomerHandler(t *testing.T) {
	env := newTestEnv()
	payload := `{
  "nik": "1234567890123456",
  "full_name": "Budi",
  "legal_name": "Budi Santoso",
  "birth_place": "Jakarta",
  "birth_date": "1990-01-01",
  "salary": 5000000,
  "ktp_photo_url": "https://example.com/ktp.jpg",
  "selfie_photo_url": "https://example.com/selfie.jpg"
}`

	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.apiKey)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["id"] == nil {
		t.Fatalf("expected id in response")
	}
}

func TestSetLimitHandler(t *testing.T) {
	env := newTestEnv()
	payload := `{"tenor_months": 3, "amount": 500000}`

	req := httptest.NewRequest(http.MethodPost, "/customers/1/limits", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.apiKey)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if resp["tenor_months"].(float64) != 3 {
		t.Fatalf("expected tenor_months 3")
	}
}

func TestListLimitsHandler(t *testing.T) {
	env := newTestEnv()
	_ = env.limitRepo.Upsert(context.Background(), 1, 3, 500_000, 100_000)
	_ = env.limitRepo.Upsert(context.Background(), 1, 6, 700_000, 100_000)

	req := httptest.NewRequest(http.MethodGet, "/customers/1/limits", nil)
	req.Header.Set("X-API-Key", env.apiKey)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 limits, got %d", len(resp))
	}
}

func TestCreateTransactionHandler(t *testing.T) {
	env := newTestEnv()
	_ = env.limitRepo.Upsert(context.Background(), 1, 3, 1_000_000, 0)
	_ = env.limitRepo.Upsert(context.Background(), 1, 6, 1_500_000, 0)

	payload := `{
  "contract_number": "CN-001",
  "customer_id": 1,
  "tenor_months": 3,
  "asset_name": "Motor",
  "channel": "dealer",
  "otr": 200000,
  "admin_fee": 10000,
  "installment_amount": 20000,
  "interest_amount": 5000
}`

	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", env.apiKey)
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	limit3, _ := env.limitRepo.GetByCustomerAndTenor(context.Background(), 1, 3)
	limit6, _ := env.limitRepo.GetByCustomerAndTenor(context.Background(), 1, 6)
	if limit3.UsedAmount == 0 || limit6.UsedAmount != limit3.UsedAmount {
		t.Fatalf("expected used amount to be updated for all tenors")
	}
}

func TestUnauthorizedHandler(t *testing.T) {
	env := newTestEnv()
	req := httptest.NewRequest(http.MethodPost, "/customers", bytes.NewBufferString(`{"nik":"1"}`))
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
