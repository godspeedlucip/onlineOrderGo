package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-baseline-skeleton/internal/cart/domain"
)

type Deps struct {
	Repo        domain.CartRepository
	Products    domain.ProductGateway
	Users       domain.UserContext
	Tx          domain.TxManager
	Idempotency domain.IdempotencyStore

	// Optional cross-domain dependencies. Keep injected for future expansion.
	Cache     domain.CachePort
	MQ        domain.MQPort
	WebSocket domain.WebSocketPort
	Payment   domain.PaymentPort

	IdempotencyTTL time.Duration
}

type Service struct {
	deps Deps
}

func NewService(deps Deps) *Service {
	if deps.IdempotencyTTL <= 0 {
		deps.IdempotencyTTL = 5 * time.Minute
	}
	return &Service{deps: deps}
}

func (s *Service) AddItem(ctx context.Context, cmd domain.AddCartItemCmd, idemKey string) (*domain.CartItemVO, error) {
	if cmd.ItemID <= 0 || cmd.Count <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid add command", nil)
	}
	if err := validateItemType(cmd.ItemType); err != nil {
		return nil, err
	}

	return s.withIdempotency(ctx, "cart:add", idemKey, func(txCtx context.Context) (*domain.CartItemVO, error) {
		if s.deps.Repo == nil || s.deps.Products == nil || s.deps.Users == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart dependencies not initialized", nil)
		}
		userID, ok := s.deps.Users.CurrentUserID(txCtx)
		if !ok || userID <= 0 {
			return nil, domain.NewBizError(domain.CodeInvalidArgument, "user not found in context", nil)
		}

		key := domain.CartItemKey{ItemType: cmd.ItemType, ItemID: cmd.ItemID, Flavor: normalizeFlavor(cmd.Flavor)}
		snap, err := s.loadSnapshot(txCtx, cmd.ItemType, cmd.ItemID)
		if err != nil {
			return nil, err
		}
		if snap == nil || !snap.SaleEnabled {
			return nil, domain.NewBizError(domain.CodeConflict, "item is not available", nil)
		}

		existing, err := s.deps.Repo.GetByKey(txCtx, userID, key)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			newQty := existing.Quantity + cmd.Count
			updated, updateErr := s.deps.Repo.UpdateQuantity(txCtx, existing.ID, newQty, existing.Version)
			if updateErr != nil {
				return nil, updateErr
			}
			if !updated {
				return nil, domain.NewBizError(domain.CodeConflict, "cart concurrent update", nil)
			}
			latest, latestErr := s.deps.Repo.GetByKey(txCtx, userID, key)
			if latestErr != nil {
				return nil, latestErr
			}
			if latest == nil {
				return nil, domain.NewBizError(domain.CodeInternal, "cart item lost after update", nil)
			}
			return toVO(*latest), nil
		}

		item := domain.CartItem{
			UserID:    userID,
			ItemType:  cmd.ItemType,
			ItemID:    cmd.ItemID,
			Flavor:    key.Flavor,
			Name:      snap.Name,
			Image:     snap.Image,
			UnitPrice: snap.Price,
			Quantity:  cmd.Count,
			Amount:    snap.Price * int64(cmd.Count),
		}
		id, createErr := s.deps.Repo.Create(txCtx, item)
		if createErr != nil {
			return nil, createErr
		}
		created, getErr := s.deps.Repo.GetByKey(txCtx, userID, key)
		if getErr != nil {
			return nil, getErr
		}
		if created == nil {
			// TODO: replace with direct repository GetByID when available.
			created = &domain.CartItem{ID: id, UserID: userID, ItemType: cmd.ItemType, ItemID: cmd.ItemID, Flavor: key.Flavor, Name: snap.Name, Image: snap.Image, UnitPrice: snap.Price, Quantity: cmd.Count, Amount: snap.Price * int64(cmd.Count)}
		}
		return toVO(*created), nil
	})
}

func (s *Service) SubItem(ctx context.Context, cmd domain.SubCartItemCmd, idemKey string) (*domain.CartItemVO, error) {
	if cmd.ItemID <= 0 || cmd.Count <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid sub command", nil)
	}
	if err := validateItemType(cmd.ItemType); err != nil {
		return nil, err
	}

	return s.withIdempotency(ctx, "cart:sub", idemKey, func(txCtx context.Context) (*domain.CartItemVO, error) {
		if s.deps.Repo == nil || s.deps.Users == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart dependencies not initialized", nil)
		}
		userID, ok := s.deps.Users.CurrentUserID(txCtx)
		if !ok || userID <= 0 {
			return nil, domain.NewBizError(domain.CodeInvalidArgument, "user not found in context", nil)
		}
		key := domain.CartItemKey{ItemType: cmd.ItemType, ItemID: cmd.ItemID, Flavor: normalizeFlavor(cmd.Flavor)}

		existing, err := s.deps.Repo.GetByKey(txCtx, userID, key)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, domain.NewBizError(domain.CodeNotFound, "cart item not found", nil)
		}

		newQty := existing.Quantity - cmd.Count
		if newQty <= 0 {
			deleted, deleteErr := s.deps.Repo.DeleteByID(txCtx, existing.ID)
			if deleteErr != nil {
				return nil, deleteErr
			}
			if !deleted {
				return nil, domain.NewBizError(domain.CodeConflict, "cart concurrent update", nil)
			}
			return &domain.CartItemVO{}, nil
		}

		updated, updateErr := s.deps.Repo.UpdateQuantity(txCtx, existing.ID, newQty, existing.Version)
		if updateErr != nil {
			return nil, updateErr
		}
		if !updated {
			return nil, domain.NewBizError(domain.CodeConflict, "cart concurrent update", nil)
		}
		latest, getErr := s.deps.Repo.GetByKey(txCtx, userID, key)
		if getErr != nil {
			return nil, getErr
		}
		if latest == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart item lost after update", nil)
		}
		return toVO(*latest), nil
	})
}

func (s *Service) UpdateQuantity(ctx context.Context, cmd domain.UpdateCartQtyCmd, idemKey string) (*domain.CartItemVO, error) {
	if cmd.ItemID <= 0 || cmd.Count < 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "invalid update command", nil)
	}
	if err := validateItemType(cmd.ItemType); err != nil {
		return nil, err
	}

	return s.withIdempotency(ctx, "cart:update", idemKey, func(txCtx context.Context) (*domain.CartItemVO, error) {
		if s.deps.Repo == nil || s.deps.Users == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart dependencies not initialized", nil)
		}
		userID, ok := s.deps.Users.CurrentUserID(txCtx)
		if !ok || userID <= 0 {
			return nil, domain.NewBizError(domain.CodeInvalidArgument, "user not found in context", nil)
		}
		key := domain.CartItemKey{ItemType: cmd.ItemType, ItemID: cmd.ItemID, Flavor: normalizeFlavor(cmd.Flavor)}
		existing, err := s.deps.Repo.GetByKey(txCtx, userID, key)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, domain.NewBizError(domain.CodeNotFound, "cart item not found", nil)
		}

		if cmd.Count == 0 {
			deleted, deleteErr := s.deps.Repo.DeleteByID(txCtx, existing.ID)
			if deleteErr != nil {
				return nil, deleteErr
			}
			if !deleted {
				return nil, domain.NewBizError(domain.CodeConflict, "cart concurrent update", nil)
			}
			return &domain.CartItemVO{}, nil
		}

		updated, updateErr := s.deps.Repo.UpdateQuantity(txCtx, existing.ID, cmd.Count, existing.Version)
		if updateErr != nil {
			return nil, updateErr
		}
		if !updated {
			return nil, domain.NewBizError(domain.CodeConflict, "cart concurrent update", nil)
		}
		latest, getErr := s.deps.Repo.GetByKey(txCtx, userID, key)
		if getErr != nil {
			return nil, getErr
		}
		if latest == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart item lost after update", nil)
		}
		return toVO(*latest), nil
	})
}

func (s *Service) ListItems(ctx context.Context) ([]domain.CartItemVO, error) {
	if s.deps.Repo == nil || s.deps.Users == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "cart dependencies not initialized", nil)
	}
	userID, ok := s.deps.Users.CurrentUserID(ctx)
	if !ok || userID <= 0 {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "user not found in context", nil)
	}
	items, err := s.deps.Repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CartItemVO, 0, len(items))
	for _, item := range items {
		out = append(out, *toVO(item))
	}
	return out, nil
}

func (s *Service) Clear(ctx context.Context, idemKey string) error {
	_, err := s.withIdempotency(ctx, "cart:clear", idemKey, func(txCtx context.Context) (*domain.CartItemVO, error) {
		if s.deps.Repo == nil || s.deps.Users == nil {
			return nil, domain.NewBizError(domain.CodeInternal, "cart dependencies not initialized", nil)
		}
		userID, ok := s.deps.Users.CurrentUserID(txCtx)
		if !ok || userID <= 0 {
			return nil, domain.NewBizError(domain.CodeInvalidArgument, "user not found in context", nil)
		}
		if err := s.deps.Repo.ClearByUser(txCtx, userID); err != nil {
			return nil, err
		}
		return &domain.CartItemVO{}, nil
	})
	return err
}

func (s *Service) withIdempotency(
	ctx context.Context,
	scene string,
	idemKey string,
	action func(ctx context.Context) (*domain.CartItemVO, error),
) (*domain.CartItemVO, error) {
	if s.deps.Tx == nil {
		return nil, domain.NewBizError(domain.CodeInternal, "tx manager not initialized", nil)
	}
	if action == nil {
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "action is nil", nil)
	}

	runInTx := func(runCtx context.Context) (*domain.CartItemVO, error) {
		var out *domain.CartItemVO
		err := s.deps.Tx.RunInTx(runCtx, func(txCtx context.Context) error {
			result, e := action(txCtx)
			if e != nil {
				return e
			}
			out = result
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	if s.deps.Idempotency == nil || idemKey == "" {
		return runInTx(ctx)
	}

	token, acquired, err := s.deps.Idempotency.Acquire(ctx, scene, idemKey, s.deps.IdempotencyTTL)
	if err != nil {
		return nil, err
	}
	if !acquired {
		return nil, domain.NewBizError(domain.CodeConflict, fmt.Sprintf("duplicate request: %s", scene), nil)
	}

	out, runErr := runInTx(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, scene, idemKey, token, runErr.Error())
		return nil, runErr
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, scene, idemKey, token); doneErr != nil {
		// TODO: add retry path for idempotency mark done failure.
		_ = doneErr
	}
	return out, nil
}

func (s *Service) loadSnapshot(ctx context.Context, itemType domain.ItemType, itemID int64) (*domain.ItemSnapshot, error) {
	switch itemType {
	case domain.ItemTypeDish:
		return s.deps.Products.GetDishSnapshot(ctx, itemID)
	case domain.ItemTypeSetmeal:
		return s.deps.Products.GetSetmealSnapshot(ctx, itemID)
	default:
		return nil, domain.NewBizError(domain.CodeInvalidArgument, "unsupported itemType", nil)
	}
}

func validateItemType(itemType domain.ItemType) error {
	if itemType != domain.ItemTypeDish && itemType != domain.ItemTypeSetmeal {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid itemType", nil)
	}
	return nil
}

func normalizeFlavor(v string) string {
	return strings.TrimSpace(v)
}

func toVO(item domain.CartItem) *domain.CartItemVO {
	return &domain.CartItemVO{
		ID:        item.ID,
		ItemType:  string(item.ItemType),
		ItemID:    item.ItemID,
		Flavor:    item.Flavor,
		Name:      item.Name,
		Image:     item.Image,
		UnitPrice: item.UnitPrice,
		Quantity:  item.Quantity,
		Amount:    item.Amount,
	}
}