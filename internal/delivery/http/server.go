package http

import (
	"net/http"

	"kreditplus-test/pkg/config"
)

func NewServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.AppAddr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
