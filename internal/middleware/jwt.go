package middleware

import (
	"fmt"
	"fyp/domain/errs"
	"fyp/domain/param"
	"fyp/internal/contexts"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func JWTUserContext() echo.MiddlewareFunc {
	return JWTUserContextWithSecret(os.Getenv("JWT_SECRET"))
}

func JWTUserContextWithSecret(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := strings.TrimSpace(c.Request().Header.Get("Authorization"))
			if header == "" {
				return next(c)
			}

			tokenString, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || strings.TrimSpace(tokenString) == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header")
			}

			user, err := userFromToken(strings.TrimSpace(tokenString), secret)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			req := c.Request()
			c.SetRequest(req.WithContext(contexts.WithUser(req.Context(), user)))

			return next(c)
		}
	}
}

func userFromToken(tokenString string, secret string) (*param.AuthUserParam, error) {
	if secret == "" {
		log.Println("JWT_SECRET is not set")
		return nil, errs.ErrInternal
	}

	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	userID, err := strconv.ParseInt(claimString(claims, "sub"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid subject claim: %w", err)
	}

	return &param.AuthUserParam{
		UserID:   userID,
		Email:    claimString(claims, "email"),
		Username: claimString(claims, "username"),
	}, nil
}

func claimString(claims jwt.MapClaims, key string) string {
	value, _ := claims[key].(string)
	return value
}
