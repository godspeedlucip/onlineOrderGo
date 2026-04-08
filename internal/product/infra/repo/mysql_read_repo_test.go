package repo

import (
	"context"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/product/domain"
)

func TestMySQLReadRepository_ListCategories_SQLAligned(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	typ := 1
	status := 1
	repo := NewMySQLReadRepository(db)

	q := "SELECT id, type, name, sort, status FROM category WHERE 1=1 AND type = ? AND status = ? ORDER BY sort ASC, create_time DESC"
	rows := sqlmock.NewRows([]string{"id", "type", "name", "sort", "status"}).AddRow(10, 1, "Hot", 1, 1)
	mock.ExpectQuery(regexp.QuoteMeta(q)).WithArgs(1, 1).WillReturnRows(rows)

	items, err := repo.ListCategories(context.Background(), domain.CategoryQuery{Type: &typ, Status: &status})
	if err != nil {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Hot" {
		t.Fatalf("unexpected categories: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestMySQLReadRepository_ListDishes_SQLAligned(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	cid := int64(9)
	status := 1
	repo := NewMySQLReadRepository(db)

	q := "SELECT id, category_id, name, price, image, description, status, sort FROM dish WHERE 1=1 AND name LIKE CONCAT('%', ?, '%') AND category_id = ? AND status = ? ORDER BY create_time DESC"
	rows := sqlmock.NewRows([]string{"id", "category_id", "name", "price", "image", "description", "status", "sort"}).AddRow(100, 9, "Mapo", 2200, "", "", 1, 1)
	mock.ExpectQuery(regexp.QuoteMeta(q)).WithArgs("Map", int64(9), 1).WillReturnRows(rows)

	items, err := repo.ListDishes(context.Background(), domain.DishQuery{Name: "Map", CategoryID: &cid, Status: &status})
	if err != nil {
		t.Fatalf("ListDishes failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != 100 {
		t.Fatalf("unexpected dishes: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestMySQLReadRepository_ListSetmeals_NoOrderByAsJavaListSQL(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	cid := int64(2)
	status := 1
	repo := NewMySQLReadRepository(db)

	q := "SELECT id, category_id, name, price, image, description, status FROM setmeal WHERE 1=1 AND category_id = ? AND status = ?"
	rows := sqlmock.NewRows([]string{"id", "category_id", "name", "price", "image", "description", "status"})
	mock.ExpectQuery(regexp.QuoteMeta(q)).WithArgs(int64(2), 1).WillReturnRows(rows)

	items, err := repo.ListSetmeals(context.Background(), domain.SetmealQuery{CategoryID: &cid, Status: &status})
	if err != nil {
		t.Fatalf("ListSetmeals failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty setmeals, got: %+v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
