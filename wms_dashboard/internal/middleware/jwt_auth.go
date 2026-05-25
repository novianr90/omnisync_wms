package middleware

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// jwtKey is loaded once at package init from the JWT_SECRET_KEY environment variable.
// The application will abort at startup if the variable is not set.
var jwtKey []byte

func init() {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Fatal("JWT_SECRET_KEY environment variable is not set")
	}
	jwtKey = []byte(secret)
}

type Claims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	jwt.RegisteredClaims
}

func JWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Cookies("jwt_token")
		if tokenString == "" {
			authHeader := c.Get("Authorization")
			if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = authHeader[7:]
			}
		}

		path := c.Path()

		// Skip checks for static assets and login paths
		if path == "/login" || strings.HasPrefix(path, "/static/") || path == "/auth/login-submit" {
			return c.Next()
		}

		if tokenString == "" {
			if isHtmlRequest(c) {
				return c.Redirect("/login")
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized, please login",
			})
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			// Clear invalid cookie
			c.Cookie(&fiber.Cookie{
				Name:     "jwt_token",
				Value:    "",
				Path:     "/",
				HTTPOnly: true,
			})
			if isHtmlRequest(c) {
				return c.Redirect("/login")
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Session expired or invalid, please login again",
			})
		}

		// Save verified claims in Fiber context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("user_name", claims.FirstName+" "+claims.LastName)

		return c.Next()
	}
}

func isHtmlRequest(c *fiber.Ctx) bool {
	accept := c.Get("Accept")
	return strings.Contains(accept, "text/html")
}
