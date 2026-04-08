package repo

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"go-baseline-skeleton/internal/cart/domain"
)

func TestMySQLCartRepo_CreateUniqueConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()
	repo := NewMySQLCartRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO cart (user_id, item_type, item_id, flavor, name, image, unit_price, quantity, amount, version, create_time, update_time) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)")).
		WithArgs(int64(1), "dish", int64(101), "hot", "dish", "img", int64(100), 1, int64(100), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("Error 1062 (23000): Duplicate entry"))

	_, err = repo.Create(context.Background(), domain.CartItem{
		UserID:    1,
		ItemType:  domain.ItemTypeDish,
		ItemID:    101,
		Flavor:    "hot",
		Name:      "dish",
		Image:     "img",
		UnitPrice: 100,
		Quantity:  1,
	})
	if err == nil {
		t.Fatal("expected unique conflict")
	}
	bizErr, ok := err.(*domain.BizError)
	if !ok || bizErr.Code != domain.CodeConflict {
		t.Fatalf("expected conflict biz error, got=%v", err)
	}
}

func TestMySQLCartRepo_UpdateQuantityCASConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()
	repo := NewMySQLCartRepo(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE cart SET quantity=?, amount=unit_price*?, version=version+1, update_time=? WHERE id=? AND version=?")).
		WithArgs(5, 5, sqlmock.AnyArg(), int64(1), int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	ok, err := repo.UpdateQuantity(context.Background(), 1, 5, 10)
	if err != nil {
		t.Fatalf("UpdateQuantity failed: %v", err)
	}
	if ok {
		t.Fatal("expected cas conflict false")
	}
}
