package cart

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestReader_LoadCheckedItems_AmountCalculation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	reader := NewReader(db, 200, 100, 1000, 50)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT item_type, item_id, flavor, name, unit_price, quantity FROM cart WHERE user_id=? ORDER BY update_time DESC, id DESC")).WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "flavor", "name", "unit_price", "quantity"}).
			AddRow("dish", 11, "", "A", 600, 2))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, is_deleted FROM dish WHERE id=? LIMIT 1")).WithArgs(int64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "is_deleted"}).AddRow(1, 0))

	items, total, err := reader.LoadCheckedItems(context.Background(), 1)
	if err != nil {
		t.Fatalf("LoadCheckedItems failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if total != 1250 {
		t.Fatalf("unexpected total amount: %d", total)
	}
}

func TestReader_LoadCheckedItems_DisabledItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	reader := NewReader(db, 0, 0, 0, 0)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT item_type, item_id, flavor, name, unit_price, quantity FROM cart WHERE user_id=? ORDER BY update_time DESC, id DESC")).WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "item_id", "flavor", "name", "unit_price", "quantity"}).
			AddRow("setmeal", 21, "", "S", 1000, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status, is_deleted FROM setmeal WHERE id=? LIMIT 1")).WithArgs(int64(21)).
		WillReturnRows(sqlmock.NewRows([]string{"status", "is_deleted"}).AddRow(0, 0))

	_, _, err = reader.LoadCheckedItems(context.Background(), 1)
	if err == nil {
		t.Fatal("expected disabled item error")
	}
}
