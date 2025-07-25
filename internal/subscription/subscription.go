package subscription

import "time"

type SubscriptionEntry struct {
	ID          int
	ServiceName string
	Price       int
	UserID      string
	StartDate   time.Time
	EndDate     time.Time
}
