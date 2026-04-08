package errors

import (
	"context"
	"net/http"
	"testing"

	"go-baseline-skeleton/internal/baseline/domain"
)

func TestHTTPErrorMapper_ToHTTP_BizErrorMapping(t *testing.T) {
	mapper := NewHTTPErrorMapper()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid argument", err: domain.NewBizError(domain.CodeInvalidArgument, "bad request", nil), wantStatus: http.StatusBadRequest, wantCode: string(domain.CodeInvalidArgument)},
		{name: "conflict", err: domain.NewBizError(domain.CodeConflict, "duplicate request", nil), wantStatus: http.StatusConflict, wantCode: string(domain.CodeConflict)},
		{name: "service unavailable", err: domain.NewBizError(domain.CodeServiceUnavailable, "tx failed", nil), wantStatus: http.StatusServiceUnavailable, wantCode: string(domain.CodeServiceUnavailable)},
		{name: "internal", err: domain.NewBizError(domain.CodeInternal, "internal", nil), wantStatus: http.StatusInternalServerError, wantCode: string(domain.CodeInternal)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotCode, _ := mapper.ToHTTP(tt.err)
			if gotStatus != tt.wantStatus {
				t.Fatalf("status mismatch: got=%d want=%d", gotStatus, tt.wantStatus)
			}
			if gotCode != tt.wantCode {
				t.Fatalf("code mismatch: got=%s want=%s", gotCode, tt.wantCode)
			}
		})
	}
}

func TestHTTPErrorMapper_ToHTTP_ContextTimeout(t *testing.T) {
	mapper := NewHTTPErrorMapper()
	status, code, _ := mapper.ToHTTP(context.DeadlineExceeded)

	if status != http.StatusServiceUnavailable {
		t.Fatalf("status mismatch: got=%d want=%d", status, http.StatusServiceUnavailable)
	}
	if code != string(domain.CodeServiceUnavailable) {
		t.Fatalf("code mismatch: got=%s want=%s", code, domain.CodeServiceUnavailable)
	}
}
