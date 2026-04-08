package domain

import (
	"context"
	"time"
)

type ProductWriteUsecase interface {
	CreateCategory(ctx context.Context, cmd CreateCategoryCmd, idemKey string) (int64, error)
	UpdateCategory(ctx context.Context, cmd UpdateCategoryCmd, idemKey string) error
	ChangeCategoryStatus(ctx context.Context, cmd ChangeCategoryStatusCmd, idemKey string) error
	DeleteCategory(ctx context.Context, id int64, idemKey string) error

	CreateDish(ctx context.Context, cmd CreateDishCmd, idemKey string) (int64, error)
	UpdateDish(ctx context.Context, cmd UpdateDishCmd, idemKey string) error
	ChangeDishStatus(ctx context.Context, cmd ChangeDishStatusCmd, idemKey string) error
	DeleteDish(ctx context.Context, id int64, idemKey string) error

	CreateSetmeal(ctx context.Context, cmd CreateSetmealCmd, idemKey string) (int64, error)
	UpdateSetmeal(ctx context.Context, cmd UpdateSetmealCmd, idemKey string) error
	ChangeSetmealStatus(ctx context.Context, cmd ChangeSetmealStatusCmd, idemKey string) error
	DeleteSetmeal(ctx context.Context, id int64, idemKey string) error
}

type ProductWriteRepository interface {
	CreateCategory(ctx context.Context, c Category) (int64, error)
	UpdateCategory(ctx context.Context, c Category, expectedVersion int64) (bool, error)
	UpdateCategoryStatus(ctx context.Context, id int64, status int) (bool, error)
	DeleteCategory(ctx context.Context, id int64) (bool, error)

	CreateDishWithFlavors(ctx context.Context, d Dish, flavors []DishFlavor) (int64, error)
	UpdateDishWithFlavors(ctx context.Context, d Dish, flavors []DishFlavor, expectedVersion int64) (bool, error)
	UpdateDishStatus(ctx context.Context, id int64, status int) (bool, error)
	DeleteDish(ctx context.Context, id int64) (bool, error)

	CreateSetmealWithItems(ctx context.Context, s Setmeal, items []SetmealDish) (int64, error)
	UpdateSetmealWithItems(ctx context.Context, s Setmeal, items []SetmealDish, expectedVersion int64) (bool, error)
	UpdateSetmealStatus(ctx context.Context, id int64, status int) (bool, error)
	DeleteSetmeal(ctx context.Context, id int64) (bool, error)

	ExistsDishUsedBySetmeal(ctx context.Context, dishID int64) (bool, error)
	ExistsCategoryUsedByDish(ctx context.Context, categoryID int64) (bool, error)
	ExistsCategoryUsedBySetmeal(ctx context.Context, categoryID int64) (bool, error)
}

type ProductCacheInvalidator interface {
	InvalidateCategory(ctx context.Context, categoryID int64) error
	InvalidateDish(ctx context.Context, dishID int64, categoryID int64) error
	InvalidateSetmeal(ctx context.Context, setmealID int64, categoryID int64) error
	InvalidateByCategory(ctx context.Context, categoryID int64) error
}

type IdempotencyStore interface {
	Acquire(ctx context.Context, scene, key string, ttl time.Duration) (token string, acquired bool, err error)
	MarkDone(ctx context.Context, scene, key, token string, result []byte) error
	MarkFailed(ctx context.Context, scene, key, token, reason string) error
	GetDoneResult(ctx context.Context, scene, key string) (result []byte, found bool, err error)
}

type CacheInvalidateTask struct {
	Operation   string
	CategoryID  int64
	EntityID    int64
	EnqueueAtMS int64
	RetryCount  int
}

type CacheInvalidationOutbox interface {
	Enqueue(ctx context.Context, task CacheInvalidateTask) error
	RunOnce(ctx context.Context, invalidator ProductCacheInvalidator, limit int) (processed int, err error)
}
