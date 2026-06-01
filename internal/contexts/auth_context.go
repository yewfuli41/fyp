package contexts

import (
	"context"
	"fyp/domain/errs"
	"fyp/domain/param"
)

type userContextKey struct{}

func WithUser(ctx context.Context, user *param.AuthUserParam) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func CurrentUser(ctx context.Context) (*param.AuthUserParam, error) {
	user, ok := ctx.Value(userContextKey{}).(*param.AuthUserParam)
	if !ok || user == nil {
		return nil, errs.ErrUnauthenticated
	}
	return user, nil
}
