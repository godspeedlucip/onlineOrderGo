package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComposeRoutes_ProductMounted(t *testing.T) {
	identity := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	product := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	baseline := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := composeRoutes(identity, product, baseline)

	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/product/category/list", nil))
	if rec1.Code != http.StatusAccepted {
		t.Fatalf("product route not mounted, got=%d", rec1.Code)
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/identity/me", nil))
	if rec2.Code != http.StatusTeapot {
		t.Fatalf("identity route broken, got=%d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec3.Code != http.StatusOK {
		t.Fatalf("baseline route broken, got=%d", rec3.Code)
	}
}
