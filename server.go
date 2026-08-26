package main

import (
	"fyp/app"
	"fyp/config"
	"fyp/graph"
	"fyp/graph/resolver"
	authMiddleware "fyp/internal/middleware"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/vektah/gqlparser/v2/ast"
)

const defaultPort = "8080"

func server(app *app.App, cfg *config.Config) {
	e := echo.New()
	//CORS
	e.Use(echoMiddleware.CORS())
	e.Use(authMiddleware.JWTUserContext(cfg.Auth.JWTSecret))
	// CSP (second layer defense for XSS)
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
					"default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self' 'unsafe-inline';",
				)

			return next(c)
		}
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver.NewResolver(app)}))

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

	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:  "./client",
		HTML5: true,
	}))

	e.Logger.Fatal(e.Start(":" + port))
}
