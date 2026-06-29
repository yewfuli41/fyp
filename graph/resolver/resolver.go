package resolver

import (
	"fyp/app"
	"strconv"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	App *app.App
}

func NewResolver(app *app.App) *Resolver {
	return &Resolver{
		App: app,
	}
}

func parseID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}
