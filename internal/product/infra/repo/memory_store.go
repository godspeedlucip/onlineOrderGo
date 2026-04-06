package repo

import (
	"sync"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type memoryStore struct {
	mu sync.RWMutex

	nextCategoryID   int64
	nextDishID       int64
	nextFlavorID     int64
	nextSetmealID    int64
	nextSetmealDishID int64

	categories      map[int64]domain.Category
	categoryVersion map[int64]int64
	dishes          map[int64]domain.Dish
	dishVersion     map[int64]int64
	dishFlavors     map[int64][]domain.DishFlavor
	setmeals        map[int64]domain.Setmeal
	setmealVersion  map[int64]int64
	setmealDishes   map[int64][]domain.SetmealDish
}

func newMemoryStoreWithSeed() *memoryStore {
	now := time.Now()
	s := &memoryStore{
		nextCategoryID:    3,
		nextDishID:        103,
		nextFlavorID:      3,
		nextSetmealID:     202,
		nextSetmealDishID: 3,
		categories: map[int64]domain.Category{
			1: {ID: 1, Type: 1, Name: "Hot Dish", Sort: 1, Status: domain.StatusEnabled, UpdateTime: now},
			2: {ID: 2, Type: 2, Name: "Combo", Sort: 2, Status: domain.StatusEnabled, UpdateTime: now},
		},
		categoryVersion: map[int64]int64{1: 1, 2: 1},
		dishes: map[int64]domain.Dish{
			101: {ID: 101, CategoryID: 1, Name: "Kung Pao Chicken", Price: 3800, Status: domain.StatusEnabled, Sort: 1, UpdateTime: now},
			102: {ID: 102, CategoryID: 1, Name: "Mapo Tofu", Price: 2200, Status: domain.StatusEnabled, Sort: 2, UpdateTime: now},
		},
		dishVersion: map[int64]int64{101: 1, 102: 1},
		dishFlavors: map[int64][]domain.DishFlavor{
			101: {{ID: 1, DishID: 101, Name: "spicy", Value: "medium"}},
			102: {{ID: 2, DishID: 102, Name: "spicy", Value: "low"}},
		},
		setmeals: map[int64]domain.Setmeal{
			201: {ID: 201, CategoryID: 2, Name: "Lunch Combo", Price: 5200, Status: domain.StatusEnabled, UpdateTime: now},
		},
		setmealVersion: map[int64]int64{201: 1},
		setmealDishes: map[int64][]domain.SetmealDish{
			201: {
				{ID: 1, SetmealID: 201, DishID: 101, Name: "Kung Pao Chicken", Copies: 1},
				{ID: 2, SetmealID: 201, DishID: 102, Name: "Mapo Tofu", Copies: 1},
			},
		},
	}
	return s
}

var defaultStore = newMemoryStoreWithSeed()