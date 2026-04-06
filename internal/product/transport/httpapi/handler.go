package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type Handler struct {
	read domain.ProductReadUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(read domain.ProductReadUsecase) *Handler {
	return &Handler{read: read}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/product/category/list", http.HandlerFunc(h.listCategories))
	mux.Handle("/product/dish/list", http.HandlerFunc(h.listDishes))
	mux.Handle("/product/dish/detail", http.HandlerFunc(h.getDishDetail))
	mux.Handle("/product/setmeal/list", http.HandlerFunc(h.listSetmeals))
	mux.Handle("/product/setmeal/detail", http.HandlerFunc(h.getSetmealDetail))
	return mux
}

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	q, err := parseCategoryQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.read.ListCategories(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) listDishes(w http.ResponseWriter, r *http.Request) {
	q, err := parseDishQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.read.ListDishes(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) getDishDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Query().Get("id"))
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid id", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.read.GetDishDetail(r.Context(), id)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) listSetmeals(w http.ResponseWriter, r *http.Request) {
	q, err := parseSetmealQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.read.ListSetmeals(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) getSetmealDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.URL.Query().Get("id"))
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid id", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.read.GetSetmealDetail(r.Context(), id)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func parseCategoryQuery(r *http.Request) (domain.CategoryQuery, error) {
	query := r.URL.Query()
	out := domain.CategoryQuery{ClientTag: strings.TrimSpace(query.Get("client"))}
	if out.ClientTag == "" {
		out.ClientTag = "user"
	}
	if v := strings.TrimSpace(query.Get("type")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return domain.CategoryQuery{}, err
		}
		out.Type = &n
	}
	if v := strings.TrimSpace(query.Get("status")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return domain.CategoryQuery{}, err
		}
		out.Status = &n
	}
	return out, nil
}

func parseDishQuery(r *http.Request) (domain.DishQuery, error) {
	query := r.URL.Query()
	out := domain.DishQuery{ClientTag: strings.TrimSpace(query.Get("client")), Name: strings.TrimSpace(query.Get("name"))}
	if out.ClientTag == "" {
		out.ClientTag = "user"
	}
	if v := strings.TrimSpace(query.Get("categoryId")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return domain.DishQuery{}, err
		}
		out.CategoryID = &n
	}
	if v := strings.TrimSpace(query.Get("status")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return domain.DishQuery{}, err
		}
		out.Status = &n
	}
	return out, nil
}

func parseSetmealQuery(r *http.Request) (domain.SetmealQuery, error) {
	query := r.URL.Query()
	out := domain.SetmealQuery{ClientTag: strings.TrimSpace(query.Get("client")), Name: strings.TrimSpace(query.Get("name"))}
	if out.ClientTag == "" {
		out.ClientTag = "user"
	}
	if v := strings.TrimSpace(query.Get("categoryId")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return domain.SetmealQuery{}, err
		}
		out.CategoryID = &n
	}
	if v := strings.TrimSpace(query.Get("status")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return domain.SetmealQuery{}, err
		}
		out.Status = &n
	}
	return out, nil
}

func parseID(v string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(v), 10, 64)
}

func (h *Handler) writeOK(ctx context.Context, w http.ResponseWriter, data any) {
	h.writeJSON(ctx, w, http.StatusOK, response{Code: "0", Message: "success", Data: data, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			h.writeError(ctx, w, err, http.StatusBadRequest)
		case domain.CodeNotFound:
			h.writeError(ctx, w, err, http.StatusNotFound)
		default:
			h.writeError(ctx, w, err, http.StatusInternalServerError)
		}
		return
	}
	h.writeError(ctx, w, err, http.StatusInternalServerError)
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	_ = ctx
	code := string(domain.CodeInternal)
	msg := "internal error"
	if bizErr, ok := err.(*domain.BizError); ok {
		code = string(bizErr.Code)
		msg = bizErr.Message
	}
	h.writeJSON(ctx, w, status, response{Code: code, Message: msg, Timestamp: time.Now().UnixMilli()})
}

func (h *Handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, resp response) {
	_ = ctx
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}