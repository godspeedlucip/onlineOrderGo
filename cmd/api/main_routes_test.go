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
	cart := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	report := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	baseline := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := composeRoutes(identity, product, cart, report, baseline)

	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/product/category/list", nil))
	if rec1.Code != http.StatusAccepted {
		t.Fatalf("product route not mounted, got=%d", rec1.Code)
	}

	recAdmin := httptest.NewRecorder()
	router.ServeHTTP(recAdmin, httptest.NewRequest(http.MethodGet, "/admin/category/create", nil))
	if recAdmin.Code != http.StatusAccepted {
		t.Fatalf("admin product route not mounted, got=%d", recAdmin.Code)
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/identity/me", nil))
	if rec2.Code != http.StatusTeapot {
		t.Fatalf("identity route broken, got=%d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/cart/list", nil))
	if rec3.Code != http.StatusCreated {
		t.Fatalf("cart route broken, got=%d", rec3.Code)
	}

	recReport := httptest.NewRecorder()
	router.ServeHTTP(recReport, httptest.NewRequest(http.MethodGet, "/report/overview", nil))
	if recReport.Code != http.StatusNoContent {
		t.Fatalf("report route broken, got=%d", recReport.Code)
	}

	recAdminReport := httptest.NewRecorder()
	router.ServeHTTP(recAdminReport, httptest.NewRequest(http.MethodGet, "/admin/report/orders", nil))
	if recAdminReport.Code != http.StatusNoContent {
		t.Fatalf("admin report route broken, got=%d", recAdminReport.Code)
	}

	rec4 := httptest.NewRecorder()
	router.ServeHTTP(rec4, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec4.Code != http.StatusOK {
		t.Fatalf("baseline route broken, got=%d", rec4.Code)
	}
}
