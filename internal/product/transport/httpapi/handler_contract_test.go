package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-baseline-skeleton/internal/product/domain"
)

type fakeReadUsecase struct{}

func (f *fakeReadUsecase) ListCategories(ctx context.Context, q domain.CategoryQuery) ([]domain.CategoryVO, error) {
	return []domain.CategoryVO{{ID: 1, Name: "Hot", Type: 1}}, nil
}

func (f *fakeReadUsecase) ListDishes(ctx context.Context, q domain.DishQuery) ([]domain.DishVO, error) {
	return []domain.DishVO{}, nil
}

func (f *fakeReadUsecase) GetDishDetail(ctx context.Context, id int64) (*domain.DishVO, error) {
	return &domain.DishVO{ID: id, Name: "Dish"}, nil
}

func (f *fakeReadUsecase) ListSetmeals(ctx context.Context, q domain.SetmealQuery) ([]domain.SetmealVO, error) {
	return []domain.SetmealVO{}, nil
}

func (f *fakeReadUsecase) GetSetmealDetail(ctx context.Context, id int64) (*domain.SetmealVO, error) {
	return &domain.SetmealVO{ID: id, Name: "Setmeal"}, nil
}

type apiEnvelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

func TestHandler_ListCategories_ResponseContract(t *testing.T) {
	h := NewHandler(&fakeReadUsecase{})
	req := httptest.NewRequest(http.MethodGet, "/product/category/list?type=1", nil)
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status mismatch: got=%d want=%d", rec.Code, http.StatusOK)
	}

	var resp apiEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode envelope failed: %v", err)
	}
	if resp.Code != "0" || resp.Message != "success" || resp.Timestamp <= 0 {
		t.Fatalf("envelope mismatch: %+v", resp)
	}

	var categories []domain.CategoryVO
	if err := json.Unmarshal(resp.Data, &categories); err != nil {
		t.Fatalf("decode categories failed: %v", err)
	}
	if len(categories) != 1 || categories[0].ID != 1 || categories[0].Type != 1 {
		t.Fatalf("categories mismatch: %+v", categories)
	}
}
