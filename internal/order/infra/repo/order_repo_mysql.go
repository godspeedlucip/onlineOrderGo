package repo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-baseline-skeleton/internal/order/domain"
)

// TODO: replace in-memory maps with MySQL + monthly sharding router in production.
type MySQLOrderRepo struct {
	mu         sync.RWMutex
	nextID     int64
	nextNoSeq  int64
	data       map[int64]*domain.Order
	itemsByOID map[int64][]domain.OrderItem
}

func NewMySQLOrderRepo() *MySQLOrderRepo {
	return &MySQLOrderRepo{
		nextID:     1000,
		nextNoSeq:  1,
		data:       make(map[int64]*domain.Order),
		itemsByOID: make(map[int64][]domain.OrderItem),
	}
}

func (r *MySQLOrderRepo) NextOrderNo(ctx context.Context) (string, error) {
	_ = ctx
	r.mu.Lock()
	seq := r.nextNoSeq
	r.nextNoSeq++
	r.mu.Unlock()
	return fmt.Sprintf("O%s%04d", time.Now().Format("20060102150405"), seq%10000), nil
}

func (r *MySQLOrderRepo) SaveOrder(ctx context.Context, order *domain.Order, items []domain.OrderItem) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	order.OrderID = r.nextID
	cp := *order
	r.data[order.OrderID] = &cp
	if len(items) > 0 {
		rows := make([]domain.OrderItem, 0, len(items))
		for _, it := range items {
			it.OrderID = order.OrderID
			rows = append(rows, it)
		}
		r.itemsByOID[order.OrderID] = rows
	}
	return nil
}

func (r *MySQLOrderRepo) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.data[orderID]
	if !ok {
		return nil, nil
	}
	cp := *v
	return &cp, nil
}

func (r *MySQLOrderRepo) UpdateWithVersion(ctx context.Context, order *domain.Order, expectVersion int64) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	cur, ok := r.data[order.OrderID]
	if !ok {
		return false, nil
	}
	if cur.Version != expectVersion {
		return false, nil
	}
	cp := *order
	r.data[order.OrderID] = &cp
	return true, nil
}
