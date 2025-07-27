package subscription

type SubscriptionEntry struct {
	ServiceName string `json:"service_name" validator:"required,alphanum"`
	Price       int    `json:"price" validator:"required,numeric"`
	UserID      string `json:"user_id" validator:"required,uuid"`
	StartDate   string `json:"start_date" validator:"required,datetime=01-2006"`
	EndDate     string `json:"end_date,omitempty" validator:"omitempty,datetime=01-2006"`
}
