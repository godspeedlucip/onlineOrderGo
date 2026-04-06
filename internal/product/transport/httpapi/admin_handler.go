package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-baseline-skeleton/internal/product/domain"
)

type AdminHandler struct {
	write domain.ProductWriteUsecase
}

type adminResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewAdminHandler(write domain.ProductWriteUsecase) *AdminHandler {
	return &AdminHandler{write: write}
}

func (h *AdminHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/admin/category/create", http.HandlerFunc(h.createCategory))
	mux.Handle("/admin/category/update", http.HandlerFunc(h.updateCategory))
	mux.Handle("/admin/category/status", http.HandlerFunc(h.changeCategoryStatus))
	mux.Handle("/admin/category/delete", http.HandlerFunc(h.deleteCategory))

	mux.Handle("/admin/dish/create", http.HandlerFunc(h.createDish))
	mux.Handle("/admin/dish/update", http.HandlerFunc(h.updateDish))
	mux.Handle("/admin/dish/status", http.HandlerFunc(h.changeDishStatus))
	mux.Handle("/admin/dish/delete", http.HandlerFunc(h.deleteDish))

	mux.Handle("/admin/setmeal/create", http.HandlerFunc(h.createSetmeal))
	mux.Handle("/admin/setmeal/update", http.HandlerFunc(h.updateSetmeal))
	mux.Handle("/admin/setmeal/status", http.HandlerFunc(h.changeSetmealStatus))
	mux.Handle("/admin/setmeal/delete", http.HandlerFunc(h.deleteSetmeal))
	return mux
}

func (h *AdminHandler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateCategoryCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	id, err := h.write.CreateCategory(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"id": id})
}

func (h *AdminHandler) updateCategory(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateCategoryCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.UpdateCategory(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) changeCategoryStatus(w http.ResponseWriter, r *http.Request) {
	var req domain.ChangeCategoryStatusCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.ChangeCategoryStatus(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) deleteCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid id", err), http.StatusBadRequest)
		return
	}
	if err := h.write.DeleteCategory(r.Context(), id, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"deleted": true})
}

func (h *AdminHandler) createDish(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateDishCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	id, err := h.write.CreateDish(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"id": id})
}

func (h *AdminHandler) updateDish(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateDishCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.UpdateDish(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) changeDishStatus(w http.ResponseWriter, r *http.Request) {
	var req domain.ChangeDishStatusCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.ChangeDishStatus(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) deleteDish(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid id", err), http.StatusBadRequest)
		return
	}
	if err := h.write.DeleteDish(r.Context(), id, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"deleted": true})
}

func (h *AdminHandler) createSetmeal(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateSetmealCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	id, err := h.write.CreateSetmeal(r.Context(), req, idemKey(r))
	if err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"id": id})
}

func (h *AdminHandler) updateSetmeal(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateSetmealCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.UpdateSetmeal(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) changeSetmealStatus(w http.ResponseWriter, r *http.Request) {
	var req domain.ChangeSetmealStatusCmd
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid body", err), http.StatusBadRequest)
		return
	}
	if err := h.write.ChangeSetmealStatus(r.Context(), req, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"updated": true})
}

func (h *AdminHandler) deleteSetmeal(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		h.writeError(r.Context(), w, domain.NewBizError(domain.CodeInvalidArgument, "invalid id", err), http.StatusBadRequest)
		return
	}
	if err := h.write.DeleteSetmeal(r.Context(), id, idemKey(r)); err != nil {
		h.writeBizError(r.Context(), w, err)
		return
	}
	h.writeOK(r.Context(), w, map[string]any{"deleted": true})
}

func idemKey(r *http.Request) string {
	return r.Header.Get("Idempotency-Key")
}

func (h *AdminHandler) writeOK(ctx context.Context, w http.ResponseWriter, data any) {
	h.writeJSON(ctx, w, http.StatusOK, adminResponse{Code: "0", Message: "success", Data: data, Timestamp: time.Now().UnixMilli()})
}

func (h *AdminHandler) writeBizError(ctx context.Context, w http.ResponseWriter, err error) {
	if bizErr, ok := err.(*domain.BizError); ok {
		switch bizErr.Code {
		case domain.CodeInvalidArgument:
			h.writeError(ctx, w, err, http.StatusBadRequest)
		case domain.CodeNotFound:
			h.writeError(ctx, w, err, http.StatusNotFound)
		case domain.CodeConflict:
			h.writeError(ctx, w, err, http.StatusConflict)
		default:
			h.writeError(ctx, w, err, http.StatusInternalServerError)
		}
		return
	}
	h.writeError(ctx, w, err, http.StatusInternalServerError)
}

func (h *AdminHandler) writeError(ctx context.Context, w http.ResponseWriter, err error, status int) {
	_ = ctx
	code := string(domain.CodeInternal)
	msg := "internal error"
	if bizErr, ok := err.(*domain.BizError); ok {
		code = string(bizErr.Code)
		msg = bizErr.Message
	}
	h.writeJSON(ctx, w, status, adminResponse{Code: code, Message: msg, Timestamp: time.Now().UnixMilli()})
}

func (h *AdminHandler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, resp adminResponse) {
	_ = ctx
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}