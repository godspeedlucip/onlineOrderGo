package logging

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

type JSONLogger struct {
	appName string
	env     string
}

func NewJSONLogger(appName, env string) *JSONLogger {
	return &JSONLogger{appName: appName, env: env}
}

func (l *JSONLogger) Info(ctx context.Context, msg string, fields map[string]any) {
	l.write(ctx, "INFO", msg, nil, fields)
}

func (l *JSONLogger) Error(ctx context.Context, msg string, err error, fields map[string]any) {
	l.write(ctx, "ERROR", msg, err, fields)
}

func (l *JSONLogger) write(ctx context.Context, level, msg string, err error, fields map[string]any) {
	entry := map[string]any{
		"ts":      time.Now().Format(time.RFC3339Nano),
		"level":   level,
		"app":     l.appName,
		"env":     l.env,
		"message": msg,
	}

	if rid, ok := RequestIDFromContext(ctx); ok {
		entry["request_id"] = rid
	}

	for k, v := range fields {
		entry[k] = v
	}

	if err != nil {
		entry["error"] = err.Error()
	}

	b, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		log.Printf("{\"level\":\"ERROR\",\"message\":\"marshal log failed\",\"error\":%q}", marshalErr.Error())
		return
	}
	log.Println(string(b))
}

type contextKey string

const requestIDKey contextKey = "request_id"

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(requestIDKey)
	requestID, ok := v.(string)
	return requestID, ok
}