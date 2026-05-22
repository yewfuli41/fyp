package main

import (
	"fyp/app"
	"fyp/graph"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/vektah/gqlparser/v2/ast"
)

const defaultPort = "8080"

func server(app *app.App) {
	e := echo.New()
	//CORS
	e.Use(echoMiddleware.CORS())
	// CSP
	e.Use(func(
		next echo.HandlerFunc,
	) echo.HandlerFunc {
		return func(
			c echo.Context,
		) error {
			c.Response().
				Header().
				Set(
					"Content-Security-Policy",
					"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline';",
				)

			return next(c)
		}
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: graph.NewResolver(app)}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	e.POST(
		"/query",
		echo.WrapHandler(
			srv,
		),
	)

	e.Logger.Fatal(e.Start(":" + port))
}
