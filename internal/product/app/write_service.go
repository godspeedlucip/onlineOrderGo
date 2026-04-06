package app

import (
	"context"
	"fmt"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type WriteDeps struct {
	Repo        domain.ProductWriteRepository
	Tx          domain.TxManager
	Invalidator domain.ProductCacheInvalidator
	Idempotency domain.IdempotencyStore

	// Optional cross-domain dependencies. Keep injected for future expansion.
	CacheInfra domain.CachePort
	MQ         domain.MQPort
	WebSocket  domain.WebSocketPort
	Payment    domain.PaymentPort

	IdempotencyTTL time.Duration
}

type WriteService struct {
	deps WriteDeps
}

func NewWriteService(deps WriteDeps) *WriteService {
	if deps.IdempotencyTTL <= 0 {
		deps.IdempotencyTTL = 5 * time.Minute
	}
	return &WriteService{deps: deps}
}

func (s *WriteService) CreateCategory(ctx context.Context, cmd domain.CreateCategoryCmd, idemKey string) (int64, error) {
	if cmd.Name == "" {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "name is required", nil)
	}
	id, err := s.withIdempotencyID(ctx, "product_write:create_category", idemKey, func(txCtx context.Context) (int64, error) {
		return s.deps.Repo.CreateCategory(txCtx, domain.Category{Name: cmd.Name, Type: cmd.Type, Sort: cmd.Sort, Status: domain.StatusEnabled})
	})
	if err != nil {
		return 0, err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateCategory(ctx, id)
	}
	return id, nil
}

func (s *WriteService) UpdateCategory(ctx context.Context, cmd domain.UpdateCategoryCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid category id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:update_category", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateCategory(txCtx, domain.Category{ID: cmd.ID, Name: cmd.Name, Type: cmd.Type, Sort: cmd.Sort}, cmd.ExpectedVersion)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeConflict, "category concurrent update", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateByCategory(ctx, cmd.ID)
	}
	return nil
}

func (s *WriteService) ChangeCategoryStatus(ctx context.Context, cmd domain.ChangeCategoryStatusCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid category id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:change_category_status", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateCategoryStatus(txCtx, cmd.ID, cmd.Status)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeNotFound, "category not found", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateByCategory(ctx, cmd.ID)
	}
	return nil
}

func (s *WriteService) DeleteCategory(ctx context.Context, id int64, idemKey string) error {
	if id <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid category id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:delete_category", idemKey, func(txCtx context.Context) (int64, error) {
		usedByDish, err := s.deps.Repo.ExistsCategoryUsedByDish(txCtx, id)
		if err != nil {
			return 0, err
		}
		if usedByDish {
			return 0, domain.NewBizError(domain.CodeConflict, "category used by dish", nil)
		}
		usedBySetmeal, err := s.deps.Repo.ExistsCategoryUsedBySetmeal(txCtx, id)
		if err != nil {
			return 0, err
		}
		if usedBySetmeal {
			return 0, domain.NewBizError(domain.CodeConflict, "category used by setmeal", nil)
		}
		deleted, deleteErr := s.deps.Repo.DeleteCategory(txCtx, id)
		if deleteErr != nil {
			return 0, deleteErr
		}
		if !deleted {
			return 0, domain.NewBizError(domain.CodeNotFound, "category not found", nil)
		}
		return id, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateByCategory(ctx, id)
	}
	return nil
}

func (s *WriteService) CreateDish(ctx context.Context, cmd domain.CreateDishCmd, idemKey string) (int64, error) {
	if cmd.CategoryID <= 0 || cmd.Name == "" {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid dish command", nil)
	}
	id, err := s.withIdempotencyID(ctx, "product_write:create_dish", idemKey, func(txCtx context.Context) (int64, error) {
		return s.deps.Repo.CreateDishWithFlavors(txCtx, domain.Dish{
			CategoryID:  cmd.CategoryID,
			Name:        cmd.Name,
			Price:       cmd.Price,
			Image:       cmd.Image,
			Description: cmd.Description,
			Status:      cmd.Status,
		}, cmd.Flavors)
	})
	if err != nil {
		return 0, err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateDish(ctx, id, cmd.CategoryID)
	}
	return id, nil
}

func (s *WriteService) UpdateDish(ctx context.Context, cmd domain.UpdateDishCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid dish id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:update_dish", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateDishWithFlavors(txCtx, domain.Dish{
			ID:          cmd.ID,
			CategoryID:  cmd.CategoryID,
			Name:        cmd.Name,
			Price:       cmd.Price,
			Image:       cmd.Image,
			Description: cmd.Description,
			Status:      cmd.Status,
		}, cmd.Flavors, cmd.ExpectedVersion)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeConflict, "dish concurrent update", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateDish(ctx, cmd.ID, cmd.CategoryID)
	}
	return nil
}

func (s *WriteService) ChangeDishStatus(ctx context.Context, cmd domain.ChangeDishStatusCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid dish id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:change_dish_status", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateDishStatus(txCtx, cmd.ID, cmd.Status)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeNotFound, "dish not found", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateDish(ctx, cmd.ID, 0)
	}
	return nil
}

func (s *WriteService) DeleteDish(ctx context.Context, id int64, idemKey string) error {
	if id <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid dish id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:delete_dish", idemKey, func(txCtx context.Context) (int64, error) {
		used, usedErr := s.deps.Repo.ExistsDishUsedBySetmeal(txCtx, id)
		if usedErr != nil {
			return 0, usedErr
		}
		if used {
			return 0, domain.NewBizError(domain.CodeConflict, "dish used by setmeal", nil)
		}
		deleted, deleteErr := s.deps.Repo.DeleteDish(txCtx, id)
		if deleteErr != nil {
			return 0, deleteErr
		}
		if !deleted {
			return 0, domain.NewBizError(domain.CodeNotFound, "dish not found", nil)
		}
		return id, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateDish(ctx, id, 0)
	}
	return nil
}

func (s *WriteService) CreateSetmeal(ctx context.Context, cmd domain.CreateSetmealCmd, idemKey string) (int64, error) {
	if cmd.CategoryID <= 0 || cmd.Name == "" {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "invalid setmeal command", nil)
	}
	id, err := s.withIdempotencyID(ctx, "product_write:create_setmeal", idemKey, func(txCtx context.Context) (int64, error) {
		return s.deps.Repo.CreateSetmealWithItems(txCtx, domain.Setmeal{
			CategoryID:  cmd.CategoryID,
			Name:        cmd.Name,
			Price:       cmd.Price,
			Image:       cmd.Image,
			Description: cmd.Description,
			Status:      cmd.Status,
		}, cmd.Items)
	})
	if err != nil {
		return 0, err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateSetmeal(ctx, id, cmd.CategoryID)
	}
	return id, nil
}

func (s *WriteService) UpdateSetmeal(ctx context.Context, cmd domain.UpdateSetmealCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid setmeal id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:update_setmeal", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateSetmealWithItems(txCtx, domain.Setmeal{
			ID:          cmd.ID,
			CategoryID:  cmd.CategoryID,
			Name:        cmd.Name,
			Price:       cmd.Price,
			Image:       cmd.Image,
			Description: cmd.Description,
			Status:      cmd.Status,
		}, cmd.Items, cmd.ExpectedVersion)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeConflict, "setmeal concurrent update", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateSetmeal(ctx, cmd.ID, cmd.CategoryID)
	}
	return nil
}

func (s *WriteService) ChangeSetmealStatus(ctx context.Context, cmd domain.ChangeSetmealStatusCmd, idemKey string) error {
	if cmd.ID <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid setmeal id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:change_setmeal_status", idemKey, func(txCtx context.Context) (int64, error) {
		updated, updateErr := s.deps.Repo.UpdateSetmealStatus(txCtx, cmd.ID, cmd.Status)
		if updateErr != nil {
			return 0, updateErr
		}
		if !updated {
			return 0, domain.NewBizError(domain.CodeNotFound, "setmeal not found", nil)
		}
		return cmd.ID, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateSetmeal(ctx, cmd.ID, 0)
	}
	return nil
}

func (s *WriteService) DeleteSetmeal(ctx context.Context, id int64, idemKey string) error {
	if id <= 0 {
		return domain.NewBizError(domain.CodeInvalidArgument, "invalid setmeal id", nil)
	}
	_, err := s.withIdempotencyID(ctx, "product_write:delete_setmeal", idemKey, func(txCtx context.Context) (int64, error) {
		deleted, deleteErr := s.deps.Repo.DeleteSetmeal(txCtx, id)
		if deleteErr != nil {
			return 0, deleteErr
		}
		if !deleted {
			return 0, domain.NewBizError(domain.CodeNotFound, "setmeal not found", nil)
		}
		return id, nil
	})
	if err != nil {
		return err
	}
	if s.deps.Invalidator != nil {
		_ = s.deps.Invalidator.InvalidateSetmeal(ctx, id, 0)
	}
	return nil
}

func (s *WriteService) withIdempotencyID(
	ctx context.Context,
	scene string,
	idemKey string,
	action func(ctx context.Context) (int64, error),
) (int64, error) {
	if s.deps.Repo == nil {
		return 0, domain.NewBizError(domain.CodeInternal, "repository not initialized", nil)
	}
	if action == nil {
		return 0, domain.NewBizError(domain.CodeInvalidArgument, "action is nil", nil)
	}

	runInTx := func(runCtx context.Context) (int64, error) {
		if s.deps.Tx == nil {
			// TODO: decide whether to fail fast when tx manager is missing.
			return action(runCtx)
		}
		var out int64
		err := s.deps.Tx.RunInTx(runCtx, func(txCtx context.Context) error {
			result, e := action(txCtx)
			if e != nil {
				return e
			}
			out = result
			return nil
		})
		if err != nil {
			return 0, err
		}
		return out, nil
	}

	if s.deps.Idempotency == nil || idemKey == "" {
		return runInTx(ctx)
	}

	token, acquired, err := s.deps.Idempotency.Acquire(ctx, scene, idemKey, s.deps.IdempotencyTTL)
	if err != nil {
		return 0, err
	}
	if !acquired {
		return 0, domain.NewBizError(domain.CodeConflict, fmt.Sprintf("duplicate request: %s", scene), nil)
	}

	out, runErr := runInTx(ctx)
	if runErr != nil {
		_ = s.deps.Idempotency.MarkFailed(ctx, scene, idemKey, token, runErr.Error())
		return 0, runErr
	}
	if doneErr := s.deps.Idempotency.MarkDone(ctx, scene, idemKey, token); doneErr != nil {
		// Keep write result successful; idempotency mark can be retried asynchronously.
		// TODO: add async retry/outbox for idempotency mark failure.
		_ = doneErr
	}
	return out, nil
}