package domain

type CreateCategoryCmd struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	Sort int    `json:"sort"`
}

type UpdateCategoryCmd struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Type            int    `json:"type"`
	Sort            int    `json:"sort"`
	ExpectedVersion int64  `json:"expectedVersion"`
}

type ChangeCategoryStatusCmd struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type CreateDishCmd struct {
	CategoryID  int64       `json:"categoryId"`
	Name        string      `json:"name"`
	Price       int64       `json:"price"`
	Image       string      `json:"image"`
	Description string      `json:"description"`
	Status      int         `json:"status"`
	Flavors     []DishFlavor `json:"flavors"`
}

type UpdateDishCmd struct {
	ID              int64       `json:"id"`
	CategoryID      int64       `json:"categoryId"`
	Name            string      `json:"name"`
	Price           int64       `json:"price"`
	Image           string      `json:"image"`
	Description     string      `json:"description"`
	Status          int         `json:"status"`
	Flavors         []DishFlavor `json:"flavors"`
	ExpectedVersion int64       `json:"expectedVersion"`
}

type ChangeDishStatusCmd struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}

type CreateSetmealCmd struct {
	CategoryID  int64        `json:"categoryId"`
	Name        string       `json:"name"`
	Price       int64        `json:"price"`
	Image       string       `json:"image"`
	Description string       `json:"description"`
	Status      int          `json:"status"`
	Items       []SetmealDish `json:"items"`
}

type UpdateSetmealCmd struct {
	ID              int64        `json:"id"`
	CategoryID      int64        `json:"categoryId"`
	Name            string       `json:"name"`
	Price           int64        `json:"price"`
	Image           string       `json:"image"`
	Description     string       `json:"description"`
	Status          int          `json:"status"`
	Items           []SetmealDish `json:"items"`
	ExpectedVersion int64        `json:"expectedVersion"`
}

type ChangeSetmealStatusCmd struct {
	ID     int64 `json:"id"`
	Status int   `json:"status"`
}