package domain

import "time"

const (
	StatusDisabled = 0
	StatusEnabled  = 1
)

type Category struct {
	ID         int64
	Type       int
	Name       string
	Sort       int
	Status     int
	UpdateTime time.Time
}

type Dish struct {
	ID          int64
	CategoryID  int64
	Name        string
	Price       int64
	Image       string
	Description string
	Status      int
	Sort        int
	UpdateTime  time.Time
}

type DishFlavor struct {
	ID     int64
	DishID int64
	Name   string
	Value  string
}

type Setmeal struct {
	ID          int64
	CategoryID  int64
	Name        string
	Price       int64
	Image       string
	Description string
	Status      int
	UpdateTime  time.Time
}

type SetmealDish struct {
	ID        int64
	SetmealID int64
	DishID    int64
	Name      string
	Copies    int
}

type CategoryVO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

type DishFlavorVO struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type DishVO struct {
	ID          int64          `json:"id"`
	CategoryID  int64          `json:"categoryId"`
	Name        string         `json:"name"`
	Price       int64          `json:"price"`
	Image       string         `json:"image"`
	Description string         `json:"description"`
	Flavors     []DishFlavorVO `json:"flavors,omitempty"`
}

type SetmealDishVO struct {
	DishID int64  `json:"dishId"`
	Name   string `json:"name"`
	Copies int    `json:"copies"`
}

type SetmealVO struct {
	ID          int64          `json:"id"`
	CategoryID  int64          `json:"categoryId"`
	Name        string         `json:"name"`
	Price       int64          `json:"price"`
	Image       string         `json:"image"`
	Description string         `json:"description"`
	Dishes      []SetmealDishVO `json:"dishes,omitempty"`
}

type CategoryQuery struct {
	Type      *int
	Status    *int
	ClientTag string
}

type DishQuery struct {
	CategoryID *int64
	Status     *int
	Name       string
	ClientTag  string
}

type SetmealQuery struct {
	CategoryID *int64
	Status     *int
	Name       string
	ClientTag  string
}