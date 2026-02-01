package mysql

import (
	"context"
	"database/sql"

	"github.com/go-sql-driver/mysql"

	"kreditplus-test/internal/entity"
	"kreditplus-test/internal/repository"
)

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(ctx context.Context, customer *entity.Customer) error {
	query := `INSERT INTO customers
		(nik, full_name, legal_name, birth_place, birth_date, salary, ktp_photo_url, selfie_photo_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query,
		customer.NIK,
		customer.FullName,
		customer.LegalName,
		customer.BirthPlace,
		customer.BirthDate,
		customer.Salary,
		customer.KTPPhotoURL,
		customer.SelfiePhotoURL,
	)
	if err != nil {
		if isDuplicate(err) {
			return repository.ErrDuplicate
		}
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	customer.ID = id
	return nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id int64) (*entity.Customer, error) {
	query := `SELECT id, nik, full_name, legal_name, birth_place, birth_date, salary, ktp_photo_url, selfie_photo_url, created_at, updated_at
		FROM customers WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	var customer entity.Customer
	if err := row.Scan(
		&customer.ID,
		&customer.NIK,
		&customer.FullName,
		&customer.LegalName,
		&customer.BirthPlace,
		&customer.BirthDate,
		&customer.Salary,
		&customer.KTPPhotoURL,
		&customer.SelfiePhotoURL,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &customer, nil
}

func isDuplicate(err error) bool {
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == 1062
	}
	return false
}
