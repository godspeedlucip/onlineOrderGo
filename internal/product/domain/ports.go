package domain

import (
	"context"
	"time"
)

type ProductReadUsecase interface {
	ListCategories(ctx context.Context, q CategoryQuery) ([]CategoryVO, error)
	ListDishes(ctx context.Context, q DishQuery) ([]DishVO, error)
	GetDishDetail(ctx context.Context, id int64) (*DishVO, error)
	ListSetmeals(ctx context.Context, q SetmealQuery) ([]SetmealVO, error)
	GetSetmealDetail(ctx context.Context, id int64) (*SetmealVO, error)
}

type ProductReadRepository interface {
	ListCategories(ctx context.Context, q CategoryQuery) ([]Category, error)
	ListDishes(ctx context.Context, q DishQuery) ([]Dish, error)
	ListDishFlavorsByDishIDs(ctx context.Context, dishIDs []int64) (map[int64][]DishFlavor, error)
	GetDishByID(ctx context.Context, id int64) (*Dish, error)
	ListSetmeals(ctx context.Context, q SetmealQuery) ([]Setmeal, error)
	ListSetmealDishesBySetmealIDs(ctx context.Context, setmealIDs []int64) (map[int64][]SetmealDish, error)
	GetSetmealByID(ctx context.Context, id int64) (*Setmeal, error)
}

type ProductReadCache interface {
	GetCategories(ctx context.Context, key string) ([]CategoryVO, bool, error)
	SetCategories(ctx context.Context, key string, value []CategoryVO, ttl time.Duration) error
	GetDishes(ctx context.Context, key string) ([]DishVO, bool, error)
	SetDishes(ctx context.Context, key string, value []DishVO, ttl time.Duration) error
	GetSetmeals(ctx context.Context, key string) ([]SetmealVO, bool, error)
	SetSetmeals(ctx context.Context, key string, value []SetmealVO, ttl time.Duration) error
}

type TxManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type CachePort interface {
	Ping(ctx context.Context) error
}

type MQPort interface {
	Ping(ctx context.Context) error
}

type WebSocketPort interface {
	Ping(ctx context.Context) error
}

type PaymentPort interface {
	Ping(ctx context.Context) error
}