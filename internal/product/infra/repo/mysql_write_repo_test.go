package repo

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/product/domain"
)

func TestMySQLWriteRepository_CreateCategory_SQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLWriteRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM category WHERE type = ? AND name = ? AND id <> ?")).
		WithArgs(1, "Main", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO category (type, name, sort, status, create_time, update_time, create_user, update_user) VALUES (?, ?, ?, ?, ?, ?, ?, ?)")).
		WithArgs(1, "Main", 10, domain.StatusEnabled, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(0), int64(0)).
		WillReturnResult(sqlmock.NewResult(101, 1))

	id, err := repo.CreateCategory(context.Background(), domain.Category{Name: "Main", Type: 1, Sort: 10, Status: domain.StatusEnabled})
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if id != 101 {
		t.Fatalf("CreateCategory id mismatch: got=%d", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectation mismatch: %v", err)
	}
}

func TestMySQLWriteRepository_UpdateDishWithFlavors_SQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLWriteRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(1) FROM dish WHERE category_id = ? AND name = ? AND id <> ?")).
		WithArgs(int64(3), "Noodles", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE dish SET name = ?, category_id = ?, price = ?, image = ?, description = ?, status = ?, update_time = ?, update_user = ? WHERE id = ?")).
		WithArgs("Noodles", int64(3), int64(2300), "img", "desc", 1, sqlmock.AnyArg(), int64(0), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM dish_flavor WHERE dish_id = ?")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO dish_flavor (dish_id, name, value) VALUES (?, ?, ?)")).
		WithArgs(int64(7), "size", "large").
		WillReturnResult(sqlmock.NewResult(0, 1))

	ok, err := repo.UpdateDishWithFlavors(context.Background(), domain.Dish{
		ID:          7,
		CategoryID:  3,
		Name:        "Noodles",
		Price:       2300,
		Image:       "img",
		Description: "desc",
		Status:      1,
	}, []domain.DishFlavor{{Name: "size", Value: "large"}}, 0)
	if err != nil {
		t.Fatalf("UpdateDishWithFlavors failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateDishWithFlavors expected true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectation mismatch: %v", err)
	}
}

func TestMySQLWriteRepository_DeleteSetmeal_SQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	repo := NewMySQLWriteRepository(db)
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM setmeal WHERE id = ?")).
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM setmeal_dish WHERE setmeal_id = ?")).
		WithArgs(int64(99)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	ok, err := repo.DeleteSetmeal(context.Background(), 99)
	if err != nil {
		t.Fatalf("DeleteSetmeal failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteSetmeal expected true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectation mismatch: %v", err)
	}
}
