package subscription

import (
	"time"
)

type SubscriptionEntry struct {
	ServiceName string     `json:"service_name" validator:"required,alphanum"`
	Price       int        `json:"price" validator:"required,numeric"`
	UserID      string     `json:"user_id" validator:"required,uuid"`
	StartDate   time.Time  `json:"start_date" validator:"required,datetime=2006-01"`
	EndDate     *time.Time `json:"end_date,omitempty" validator:"omitempty,datetime=2006-01"`
}
