package entity

import "time"

type Transaction struct {
	ID                int64
	ContractNumber    string
	CustomerID        int64
	TenorMonths       int
	AssetName         string
	Channel           string
	OTR               int64
	AdminFee          int64
	InstallmentAmount int64
	InterestAmount    int64
	FinancedAmount    int64
	CreatedAt         time.Time
}
