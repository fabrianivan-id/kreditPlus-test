package entity

import "time"

type Limit struct {
	ID          int64
	CustomerID  int64
	TenorMonths int
	Amount      int64
	UsedAmount  int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (l Limit) Remaining() int64 {
	return l.Amount - l.UsedAmount
}
