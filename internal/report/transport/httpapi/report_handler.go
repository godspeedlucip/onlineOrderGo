package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-baseline-skeleton/internal/report/domain"
)

type Handler struct {
	report domain.ReportUsecase
}

type response struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewHandler(report domain.ReportUsecase) *Handler {
	return &Handler{report: report}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/report/overview", http.HandlerFunc(h.overview))
	mux.Handle("/report/trend", http.HandlerFunc(h.trend))
	mux.Handle("/report/orders", http.HandlerFunc(h.orders))
	return mux
}

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	q, err := parseOverviewQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.report.QueryOverview(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) trend(w http.ResponseWriter, r *http.Request) {
	q, err := parseTrendQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.report.QueryTrend(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func (h *Handler) orders(w http.ResponseWriter, r *http.Request) {
	q, err := parseOrderListQuery(r)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid query", err), http.StatusBadRequest)
		return
	}
	out, bizErr := h.report.QueryOrderList(r.Context(), q)
	if bizErr != nil {
		h.writeBizError(r.Context(), w, bizErr)
		return
	}
	h.writeOK(r.Context(), w, out)
}

func parseOverviewQuery(r *http.Request) (domain.OverviewQuery, error) {
	query := r.URL.Query()
	from, err := parseTime(query.Get("from"))
	if err != nil {
		return domain.OverviewQuery{}, err
	}
	to, err := parseTime(query.Get("to"))
	if err != nil {
		return domain.OverviewQuery{}, err
	}
	storeID, _ := strconv.ParseInt(query.Get("storeId"), 10, 64)
	return domain.OverviewQuery{From: from, To: to, StoreID: storeID, Timezone: query.Get("timezone")}, nil
}

func parseTrendQuery(r *http.Request) (domain.TrendQuery, error) {
	query := r.URL.Query()
	from, err := parseTime(query.Get("from"))
	if err != nil {
		return domain.TrendQuery{}, err
	}
	to, err := parseTime(query.Get("to"))
	if err != nil {
		return domain.TrendQuery{}, err
	}
	storeID, _ := strconv.ParseInt(query.Get("storeId"), 10, 64)
	return domain.TrendQuery{From: from, To: to, Granularity: query.Get("granularity"), StoreID: storeID, Timezone: query.Get("timezone")}, nil
}

func parseOrderListQuery(r *http.Request) (domain.OrderListQuery, error) {
	query := r.URL.Query()
	from, err := parseTime(query.Get("from"))
	if err != nil {
		return domain.OrderListQuery{}, err
	}
	to, err := parseTime(query.Get("to"))
	if err != nil {
		return domain.OrderListQuery{}, err
	}
	storeID, _ := strconv.ParseInt(query.Get("storeId"), 10, 64)
	page, _ := strconv.Atoi(query.Get("page"))
	pageSize, _ := strconv.Atoi(query.Get("pageSize"))
	desc := query.Get("desc") == "true"
	return domain.OrderListQuery{From: from, To: to, StoreID: storeID, Page: page, PageSize: pageSize, SortBy: query.Get("sortBy"), Desc: desc}, nil
}

func parseTime(v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, v)
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