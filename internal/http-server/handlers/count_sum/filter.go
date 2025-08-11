package countsum

import "time"

type SubscriptionFilterSum struct {
	Username    string
	ServiceName *string
	StartDate   time.Time
	EndDate     *time.Time
}

type DummySubscriptionFilterSum struct {
	ServiceName string `json:"service_name,omitempty" validate:"omitempty"`
	StartDate   string `json:"start_date" validate:"required"`
	EndDate     string `json:"end_date,omitempty" validate:"omitempty"`
}
