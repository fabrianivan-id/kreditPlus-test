package main

import (
	"log"
	"net/http"

	httpdelivery "kreditplus-test/internal/delivery/http"
	"kreditplus-test/internal/repository/mysql"
	"kreditplus-test/internal/usecase"
	"kreditplus-test/pkg/config"
	"kreditplus-test/pkg/logger"
)

func main() {
	cfg := config.Load()
	appLogger := logger.New()

	db, err := mysql.NewDB(cfg)
	if err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	customerRepo := mysql.NewCustomerRepository(db.DB)
	limitRepo := mysql.NewLimitRepository(db.DB)
	transactionRepo := mysql.NewTransactionRepository()

	customerUC := usecase.NewCustomerUsecase(customerRepo)
	limitUC := usecase.NewLimitUsecase(limitRepo)
	transactionUC := usecase.NewTransactionUsecase(db, limitRepo, transactionRepo)

	handler := httpdelivery.NewHandler(customerUC, limitUC, transactionUC, db.DB)
	middleware := httpdelivery.NewMiddleware(cfg, appLogger)
	server := httpdelivery.NewServer(cfg, handler.Routes(middleware))

	appLogger.Printf("listening on %s", cfg.AppAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Fatalf("server error: %v", err)
	}
}
