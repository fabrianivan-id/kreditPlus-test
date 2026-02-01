package entity

import "time"

type Customer struct {
	ID             int64
	NIK            string
	FullName       string
	LegalName      string
	BirthPlace     string
	BirthDate      time.Time
	Salary         int64
	KTPPhotoURL    string
	SelfiePhotoURL string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
