package http

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kreditplus-test/internal/repository"
	"kreditplus-test/internal/usecase"
)

type Handler struct {
	customerUC    *usecase.CustomerUsecase
	limitUC       *usecase.LimitUsecase
	transactionUC *usecase.TransactionUsecase
	db            *sql.DB
}

func NewHandler(customerUC *usecase.CustomerUsecase, limitUC *usecase.LimitUsecase, transactionUC *usecase.TransactionUsecase, db *sql.DB) *Handler {
	return &Handler{
		customerUC:    customerUC,
		limitUC:       limitUC,
		transactionUC: transactionUC,
		db:            db,
	}
}

func (h *Handler) Routes(mw *Middleware) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/readyz", h.readyz)
	mux.HandleFunc("/customers", h.handleCustomers)
	mux.HandleFunc("/customers/", h.handleCustomerSubroutes)
	mux.HandleFunc("/transactions", h.handleTransactions)

	return Chain(
		mux,
		mw.Recover,
		mw.RequestID,
		mw.Logger,
		mw.SecurityHeaders,
		mw.CORS,
		mw.RateLimit,
		mw.BodyLimit,
		mw.APIKeyAuth,
	)
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 2*time.Second)
	defer cancel()
	if err := h.db.PingContext(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, errInternal)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) handleCustomers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createCustomer(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
	}
}

func (h *Handler) handleCustomerSubroutes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "customers" && parts[2] == "limits" {
		customerID, err := parseID(parts[1])
		if err != nil {
			writeError(w, http.StatusBadRequest, usecase.ErrInvalidInput)
			return
		}
		switch r.Method {
		case http.MethodPost:
			h.setLimit(w, r, customerID)
		case http.MethodGet:
			h.listLimits(w, r, customerID)
		default:
			writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		}
		return
	}
	writeError(w, http.StatusNotFound, errNotFound)
}

func (h *Handler) handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createTransaction(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
	}
}

func (h *Handler) createCustomer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NIK            string `json:"nik"`
		FullName       string `json:"full_name"`
		LegalName      string `json:"legal_name"`
		BirthPlace     string `json:"birth_place"`
		BirthDate      string `json:"birth_date"`
		Salary         int64  `json:"salary"`
		KTPPhotoURL    string `json:"ktp_photo_url"`
		SelfiePhotoURL string `json:"selfie_photo_url"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, usecase.ErrInvalidInput)
		return
	}

	birthDate, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, usecase.ErrInvalidInput)
		return
	}

	customer, err := h.customerUC.Create(r.Context(), usecase.CreateCustomerInput{
		NIK:            req.NIK,
		FullName:       req.FullName,
		LegalName:      req.LegalName,
		BirthPlace:     req.BirthPlace,
		BirthDate:      birthDate,
		Salary:         req.Salary,
		KTPPhotoURL:    req.KTPPhotoURL,
		SelfiePhotoURL: req.SelfiePhotoURL,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": customer.ID})
}

func (h *Handler) setLimit(w http.ResponseWriter, r *http.Request, customerID int64) {
	var req struct {
		TenorMonths int   `json:"tenor_months"`
		Amount      int64 `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, usecase.ErrInvalidInput)
		return
	}

	if err := h.limitUC.Set(r.Context(), customerID, req.TenorMonths, req.Amount); err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"customer_id":  customerID,
		"tenor_months": req.TenorMonths,
		"amount":       req.Amount,
	})
}

func (h *Handler) listLimits(w http.ResponseWriter, r *http.Request, customerID int64) {
	limits, err := h.limitUC.List(r.Context(), customerID)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	response := make([]map[string]any, 0, len(limits))
	for _, l := range limits {
		response = append(response, map[string]any{
			"id":           l.ID,
			"tenor_months": l.TenorMonths,
			"amount":       l.Amount,
			"used_amount":  l.UsedAmount,
			"remaining":    l.Remaining(),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContractNumber    string `json:"contract_number"`
		CustomerID        int64  `json:"customer_id"`
		TenorMonths       int    `json:"tenor_months"`
		AssetName         string `json:"asset_name"`
		Channel           string `json:"channel"`
		OTR               int64  `json:"otr"`
		AdminFee          int64  `json:"admin_fee"`
		InstallmentAmount int64  `json:"installment_amount"`
		InterestAmount    int64  `json:"interest_amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, usecase.ErrInvalidInput)
		return
	}

	result, err := h.transactionUC.Create(r.Context(), usecase.CreateTransactionInput{
		ContractNumber:    req.ContractNumber,
		CustomerID:        req.CustomerID,
		TenorMonths:       req.TenorMonths,
		AssetName:         req.AssetName,
		Channel:           req.Channel,
		OTR:               req.OTR,
		AdminFee:          req.AdminFee,
		InstallmentAmount: req.InstallmentAmount,
		InterestAmount:    req.InterestAmount,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"transaction_id":  result.Transaction.ID,
		"contract_number": result.Transaction.ContractNumber,
		"financed_amount": result.Transaction.FinancedAmount,
		"remaining_limit": result.RemainingLimit,
	})
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	case errors.Is(err, usecase.ErrLimitExceeded):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, usecase.ErrConflict), errors.Is(err, repository.ErrDuplicate):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, usecase.ErrNotFound), errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	default:
		writeError(w, http.StatusInternalServerError, errInternal)
	}
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func contextWithTimeout(r *http.Request, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), d)
}
