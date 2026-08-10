package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"fyp/internal/contexts"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWTUserContext", func() {
	const secret = "test-secret"

	var e *echo.Echo

	BeforeEach(func() {
		e = echo.New()
	})

	It("continues without a user when no token is provided", func() {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := JWTUserContextWithSecret(secret)(func(c echo.Context) error {
			_, err := contexts.CurrentUser(c.Request().Context())
			Expect(err).To(HaveOccurred())
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("rejects malformed authorization header", func() {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "abc123")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := JWTUserContextWithSecret(secret)(
			func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			},
		)(c)

		httpErr, ok := err.(*echo.HTTPError)
		Expect(ok).To(BeTrue())
		Expect(httpErr.Code).To(Equal(http.StatusUnauthorized))
	})

	It("rejects empty bearer token", func() {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "Bearer ")

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := JWTUserContextWithSecret(secret)(
			func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			},
		)(c)

		httpErr, ok := err.(*echo.HTTPError)
		Expect(ok).To(BeTrue())
		Expect(httpErr.Code).To(Equal(http.StatusUnauthorized))
	})

	It("attaches the authenticated user from a valid bearer token", func() {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "Bearer "+signedToken(secret, 42, "finn", "finn@example.com", time.Hour))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := JWTUserContextWithSecret(secret)(func(c echo.Context) error {
			user, err := contexts.CurrentUser(c.Request().Context())
			Expect(err).NotTo(HaveOccurred())
			Expect(user.UserID).To(Equal(int64(42)))
			Expect(user.Username).To(Equal("finn"))
			Expect(user.Email).To(Equal("finn@example.com"))
			return c.NoContent(http.StatusOK)
		})(c)

		Expect(err).NotTo(HaveOccurred())
		Expect(rec.Code).To(Equal(http.StatusOK))
	})

	It("rejects invalid bearer tokens", func() {
		req := httptest.NewRequest(http.MethodPost, "/query", nil)
		req.Header.Set("Authorization", "Bearer not-a-jwt")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := JWTUserContextWithSecret(secret)(func(c echo.Context) error {
			return c.NoContent(http.StatusOK)
		})(c)

		httpErr, ok := err.(*echo.HTTPError)
		Expect(ok).To(BeTrue())
		Expect(httpErr.Code).To(Equal(http.StatusUnauthorized))
	})

})

func signedToken(secret string, userID int64, username string, email string, expiresIn time.Duration) string {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      strconv.FormatInt(userID, 10),
		"username": username,
		"email":    email,
		"iat":      now.Unix(),
		"exp":      now.Add(expiresIn).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	Expect(err).NotTo(HaveOccurred())
	return tokenString
}
