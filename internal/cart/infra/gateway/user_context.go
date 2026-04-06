package gateway

import "context"

type ctxKey string

const userIDKey ctxKey = "user_id"

type UserContext struct{}

func NewUserContext() *UserContext { return &UserContext{} }

func (u *UserContext) CurrentUserID(ctx context.Context) (int64, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(int64)
	return id, ok
}

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}