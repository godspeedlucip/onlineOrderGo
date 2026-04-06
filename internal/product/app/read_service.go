package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type ReadDeps struct {
	Repo  domain.ProductReadRepository
	Cache domain.ProductReadCache
	Tx    domain.TxManager

	// Optional cross-domain dependencies. Keep injected for future expansion.
	CacheInfra domain.CachePort
	MQ         domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	CategoryTTL time.Duration
	DishTTL     time.Duration
	SetmealTTL  time.Duration
}

type ReadService struct {
	deps ReadDeps

	mu       sync.Mutex
	inflight map[string]*sync.Mutex
}

func NewReadService(deps ReadDeps) *ReadService {
	if deps.CategoryTTL <= 0 {
		deps.CategoryTTL = 5 * time.Minute
	}
	if deps.DishTTL <= 0 {
		deps.DishTTL = 5 * time.Minute
	}
	if deps.SetmealTTL <= 0 {
		deps.SetmealTTL = 5 * time.Minute
	}
	return &ReadService{deps: deps, inflight: make(map[string]*sync.Mutex)}
}

func (s *ReadService) ListCategories(ctx context.Context, q domain.CategoryQuery) ([]domain.CategoryVO, error) {
	if s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}
	q = normalizeCategoryQuery(q)
	key := buildCategoryCacheKey(q)

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetCategories(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	unlock := s.lockKey(key)
	defer unlock()

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetCategories(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	items, err := s.deps.Repo.ListCategories(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CategoryVO, 0, len(items))
	for _, item := range items {
		out = append(out, domain.CategoryVO{ID: item.ID, Name: item.Name, Type: item.Type})
	}

	if s.deps.Cache != nil {
		_ = s.deps.Cache.SetCategories(ctx, key, out, s.deps.CategoryTTL)
	}
	return out, nil
}

func (s *ReadService) ListDishes(ctx context.Context, q domain.DishQuery) ([]domain.DishVO, error) {
	if s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}
	q = normalizeDishQuery(q)
	key := buildDishCacheKey(q)

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetDishes(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	unlock := s.lockKey(key)
	defer unlock()

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetDishes(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	dishes, err := s.deps.Repo.ListDishes(ctx, q)
	if err != nil {
		return nil, err
	}
	dishIDs := make([]int64, 0, len(dishes))
	for _, d := range dishes {
		dishIDs = append(dishIDs, d.ID)
	}

	flavorsByDishID := map[int64][]domain.DishFlavor{}
	if len(dishIDs) > 0 {
		flavorsByDishID, err = s.deps.Repo.ListDishFlavorsByDishIDs(ctx, dishIDs)
		if err != nil {
			return nil, err
		}
	}

	out := make([]domain.DishVO, 0, len(dishes))
	for _, d := range dishes {
		vo := domain.DishVO{
			ID:          d.ID,
			CategoryID:  d.CategoryID,
			Name:        d.Name,
			Price:       d.Price,
			Image:       d.Image,
			Description: d.Description,
		}
		for _, f := range flavorsByDishID[d.ID] {
			vo.Flavors = append(vo.Flavors, domain.DishFlavorVO{Name: f.Name, Value: f.Value})
		}
		out = append(out, vo)
	}

	if s.deps.Cache != nil {
		_ = s.deps.Cache.SetDishes(ctx, key, out, s.deps.DishTTL)
	}
	return out, nil
}

func (s *ReadService) GetDishDetail(ctx context.Context, id int64) (*domain.DishVO, error) {
	if id <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid dish id", nil)
	}
	if s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}
	item, err := s.deps.Repo.GetDishByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.NewBizError(domain.CodeNotFound, "dish not found", nil)
	}

	flavorsByDishID, err := s.deps.Repo.ListDishFlavorsByDishIDs(ctx, []int64{id})
	if err != nil {
		return nil, err
	}

	vo := &domain.DishVO{
		ID:          item.ID,
		CategoryID:  item.CategoryID,
		Name:        item.Name,
		Price:       item.Price,
		Image:       item.Image,
		Description: item.Description,
	}
	for _, f := range flavorsByDishID[id] {
		vo.Flavors = append(vo.Flavors, domain.DishFlavorVO{Name: f.Name, Value: f.Value})
	}
	return vo, nil
}

func (s *ReadService) ListSetmeals(ctx context.Context, q domain.SetmealQuery) ([]domain.SetmealVO, error) {
	if s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}
	q = normalizeSetmealQuery(q)
	key := buildSetmealCacheKey(q)

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetSetmeals(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	unlock := s.lockKey(key)
	defer unlock()

	if s.deps.Cache != nil {
		if cached, ok, err := s.deps.Cache.GetSetmeals(ctx, key); err == nil && ok {
			return cached, nil
		}
	}

	items, err := s.deps.Repo.ListSetmeals(ctx, q)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}

	setmealDishes := map[int64][]domain.SetmealDish{}
	if len(ids) > 0 {
		setmealDishes, err = s.deps.Repo.ListSetmealDishesBySetmealIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	out := make([]domain.SetmealVO, 0, len(items))
	for _, item := range items {
		vo := domain.SetmealVO{
			ID:          item.ID,
			CategoryID:  item.CategoryID,
			Name:        item.Name,
			Price:       item.Price,
			Image:       item.Image,
			Description: item.Description,
		}
		for _, sd := range setmealDishes[item.ID] {
			vo.Dishes = append(vo.Dishes, domain.SetmealDishVO{DishID: sd.DishID, Name: sd.Name, Copies: sd.Copies})
		}
		out = append(out, vo)
	}

	if s.deps.Cache != nil {
		_ = s.deps.Cache.SetSetmeals(ctx, key, out, s.deps.SetmealTTL)
	}
	return out, nil
}

func (s *ReadService) GetSetmealDetail(ctx context.Context, id int64) (*domain.SetmealVO, error) {
	if id <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid setmeal id", nil)
	}
	if s.deps.Repo == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}

	item, err := s.deps.Repo.GetSetmealByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, domain.NewBizError(domain.CodeNotFound, "setmeal not found", nil)
	}

	setmealDishes, err := s.deps.Repo.ListSetmealDishesBySetmealIDs(ctx, []int64{id})
	if err != nil {
		return nil, err
	}

	vo := &domain.SetmealVO{
		ID:          item.ID,
		CategoryID:  item.CategoryID,
		Name:        item.Name,
		Price:       item.Price,
		Image:       item.Image,
		Description: item.Description,
	}
	for _, sd := range setmealDishes[id] {
		vo.Dishes = append(vo.Dishes, domain.SetmealDishVO{DishID: sd.DishID, Name: sd.Name, Copies: sd.Copies})
	}
	return vo, nil
}

// Read chain usually does not require explicit tx.
// TODO: if Java side requires repeatable-read snapshot on specific endpoints, wrap repository calls by Tx in app layer.

func (s *ReadService) lockKey(key string) func() {
	s.mu.Lock()
	m, ok := s.inflight[key]
	if !ok {
		m = &sync.Mutex{}
		s.inflight[key] = m
	}
	s.mu.Unlock()

	m.Lock()
	return func() { m.Unlock() }
}

func normalizeCategoryQuery(q domain.CategoryQuery) domain.CategoryQuery {
	if q.ClientTag == "user" && q.Status == nil {
		enabled := domain.StatusEnabled
		q.Status = &enabled
	}
	return q
}

func normalizeDishQuery(q domain.DishQuery) domain.DishQuery {
	if q.ClientTag == "user" && q.Status == nil {
		enabled := domain.StatusEnabled
		q.Status = &enabled
	}
	return q
}

func normalizeSetmealQuery(q domain.SetmealQuery) domain.SetmealQuery {
	if q.ClientTag == "user" && q.Status == nil {
		enabled := domain.StatusEnabled
		q.Status = &enabled
	}
	return q
}

func buildCategoryCacheKey(q domain.CategoryQuery) string {
	return fmt.Sprintf("product:category:type=%d:status=%d:client=%s", intPtrOr(q.Type, -1), intPtrOr(q.Status, -1), q.ClientTag)
}

func buildDishCacheKey(q domain.DishQuery) string {
	return fmt.Sprintf("product:dish:cid=%d:status=%d:name=%s:client=%s", int64PtrOr(q.CategoryID, -1), intPtrOr(q.Status, -1), q.Name, q.ClientTag)
}

func buildSetmealCacheKey(q domain.SetmealQuery) string {
	return fmt.Sprintf("product:setmeal:cid=%d:status=%d:name=%s:client=%s", int64PtrOr(q.CategoryID, -1), intPtrOr(q.Status, -1), q.Name, q.ClientTag)
}

func intPtrOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

func int64PtrOr(v *int64, def int64) int64 {
	if v == nil {
		return def
	}
	return *v
}