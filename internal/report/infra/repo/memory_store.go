package repo

import (
	"sync"
	"time"
)

type orderRecord struct {
	OrderID     int64
	OrderNumber string
	Status      int
	Amount      int64
	CreatedAt   time.Time
	StoreID     int64
	UserID      int64
	Refunded    bool
}

type memoryStore struct {
	mu     sync.RWMutex
	tables map[string][]orderRecord
}

func newMemoryStoreWithSeed() *memoryStore {
	loc := time.Local
	return &memoryStore{
		tables: map[string][]orderRecord{
			"orders_202604": {
				{OrderID: 1, OrderNumber: "A20260401001", Status: 5, Amount: 6800, CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, loc), StoreID: 1, UserID: 1001, Refunded: false},
				{OrderID: 2, OrderNumber: "A20260401002", Status: 5, Amount: 3200, CreatedAt: time.Date(2026, 4, 1, 11, 0, 0, 0, loc), StoreID: 1, UserID: 1002, Refunded: false},
				{OrderID: 3, OrderNumber: "A20260402001", Status: 6, Amount: 4200, CreatedAt: time.Date(2026, 4, 2, 12, 0, 0, 0, loc), StoreID: 1, UserID: 1001, Refunded: true},
			},
			"orders_202603": {
				{OrderID: 11, OrderNumber: "A20260330001", Status: 5, Amount: 8800, CreatedAt: time.Date(2026, 3, 30, 18, 0, 0, 0, loc), StoreID: 1, UserID: 1003, Refunded: false},
				{OrderID: 12, OrderNumber: "A20260331001", Status: 2, Amount: 2600, CreatedAt: time.Date(2026, 3, 31, 20, 0, 0, 0, loc), StoreID: 2, UserID: 1004, Refunded: false},
			},
		},
	}
}

var defaultStore = newMemoryStoreWithSeed()