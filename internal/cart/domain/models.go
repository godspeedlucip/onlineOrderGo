package domain

import "time"

type ItemType string

const (
	ItemTypeDish    ItemType = "dish"
	ItemTypeSetmeal ItemType = "setmeal"
)

type CartItem struct {
	ID            int64
	UserID        int64
	ItemType      ItemType
	ItemID        int64
	Flavor        string
	Name          string
	Image         string
	UnitPrice     int64
	Quantity      int
	Amount        int64
	Version       int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CartItemKey struct {
	ItemType ItemType
	ItemID   int64
	Flavor   string
}

type ItemSnapshot struct {
	ItemID      int64
	Name        string
	Image       string
	Price       int64
	SaleEnabled bool
}

type AddCartItemCmd struct {
	ItemType ItemType `json:"itemType"`
	ItemID   int64    `json:"itemId"`
	Flavor   string   `json:"flavor"`
	Count    int      `json:"count"`
}

type SubCartItemCmd struct {
	ItemType ItemType `json:"itemType"`
	ItemID   int64    `json:"itemId"`
	Flavor   string   `json:"flavor"`
	Count    int      `json:"count"`
}

type UpdateCartQtyCmd struct {
	ItemType ItemType `json:"itemType"`
	ItemID   int64    `json:"itemId"`
	Flavor   string   `json:"flavor"`
	Count    int      `json:"count"`
}

type CartItemVO struct {
	ID        int64  `json:"id"`
	ItemType  string `json:"itemType"`
	ItemID    int64  `json:"itemId"`
	Flavor    string `json:"flavor"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	UnitPrice int64  `json:"unitPrice"`
	Quantity  int    `json:"quantity"`
	Amount    int64  `json:"amount"`
}