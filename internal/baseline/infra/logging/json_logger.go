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
	if messageID, ok := MessageIDFromContext(ctx); ok {
		entry["message_id"] = messageID
	}
	if orderID, ok := OrderIDFromContext(ctx); ok {
		entry["order_id"] = orderID
	}
	if tableSuffix, ok := TableSuffixFromContext(ctx); ok {
		entry["table_suffix"] = tableSuffix
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

const (
	requestIDKey   contextKey = "request_id"
	messageIDKey   contextKey = "message_id"
	orderIDKey     contextKey = "order_id"
	tableSuffixKey contextKey = "table_suffix"
)

func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(requestIDKey)
	requestID, ok := v.(string)
	return requestID, ok
}

func ContextWithMessageID(ctx context.Context, messageID string) context.Context {
	return context.WithValue(ctx, messageIDKey, messageID)
}

func MessageIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(messageIDKey)
	messageID, ok := v.(string)
	return messageID, ok
}

func ContextWithOrderID(ctx context.Context, orderID string) context.Context {
	return context.WithValue(ctx, orderIDKey, orderID)
}

func OrderIDFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(orderIDKey)
	orderID, ok := v.(string)
	return orderID, ok
}

func ContextWithTableSuffix(ctx context.Context, tableSuffix string) context.Context {
	return context.WithValue(ctx, tableSuffixKey, tableSuffix)
}

func TableSuffixFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(tableSuffixKey)
	tableSuffix, ok := v.(string)
	return tableSuffix, ok
}
