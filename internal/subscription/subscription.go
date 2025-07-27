package subscription

import "time"

type SubscriptionEntry struct {
	ServiceName string     `json:"service_name" validator:"required,alphanum"`
	Price       int        `json:"price" validator:"required,numeric"`
	UserID      string     `json:"user_id" validator:"required,uuid"`
	StartDate   time.Time  `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     *time.Time `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}

type DummySubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"required,numeric"`
	UserID      string `json:"user_id" validator:"required,uuid"`
	StartDate   string `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}
