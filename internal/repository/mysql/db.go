package mysql

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"kreditplus-test/internal/repository"
	"kreditplus-test/pkg/config"
)

type DB struct {
	*sql.DB
}

type sqlTx struct {
	*sql.Tx
}

func NewDB(cfg config.Config) (*DB, error) {
	db, err := sql.Open("mysql", cfg.DBDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLife)
	db.SetConnMaxIdleTime(5 * time.Minute)

	return &DB{DB: db}, nil
}

func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (repository.Tx, error) {
	tx, err := d.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sqlTx{Tx: tx}, nil
}
