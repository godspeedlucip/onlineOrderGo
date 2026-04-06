package domain

import "time"

type OverviewQuery struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	StoreID   int64     `json:"storeId"`
	Timezone  string    `json:"timezone"`
}

type TrendQuery struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Granularity string  `json:"granularity"`
	StoreID   int64     `json:"storeId"`
	Timezone  string    `json:"timezone"`
}

type OrderListQuery struct {
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	StoreID   int64     `json:"storeId"`
	Page      int       `json:"page"`
	PageSize  int       `json:"pageSize"`
	SortBy    string    `json:"sortBy"`
	Desc      bool      `json:"desc"`
}

type OverviewPartial struct {
	OrderCount     int64
	ValidOrderCount int64
	Turnover       int64
	RefundAmount   int64
	UserCount      int64
}

type OverviewReport struct {
	OrderCount      int64 `json:"orderCount"`
	ValidOrderCount int64 `json:"validOrderCount"`
	Turnover        int64 `json:"turnover"`
	RefundAmount    int64 `json:"refundAmount"`
	UserCount       int64 `json:"userCount"`
}

type TrendPoint struct {
	TimeKey string `json:"timeKey"`
	Value   int64  `json:"value"`
}

type TrendReport struct {
	Series []TrendPoint `json:"series"`
}

type OrderRow struct {
	OrderID      int64     `json:"orderId"`
	OrderNumber  string    `json:"orderNumber"`
	Status       int       `json:"status"`
	Amount       int64     `json:"amount"`
	CreatedAt    time.Time `json:"createdAt"`
}

type OrderListResult struct {
	Total int64      `json:"total"`
	List  []OrderRow `json:"list"`
}