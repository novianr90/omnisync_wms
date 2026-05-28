package handlers

import (
	"auth_services/internal/database"
	"auth_services/internal/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func setupAuthTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	err = db.AutoMigrate(
		&models.Role{},
		&models.User{},
	)
	if err != nil {
		t.Fatalf("failed to migrate auth test db: %v", err)
	}

	database.DB = db
}

func TestJWTGenerationAndValidation(t *testing.T) {
	setupAuthTestDB(t)

	// Seed Role
	roleAdmin := models.Role{
		ID:          uuid.New().String(),
		Name:        "System Admin",
		Description: "Overall access",
	}
	if err := database.DB.Create(&roleAdmin).Error; err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}

	user := models.User{
		ID:        uuid.New().String(),
		Email:     "admin@omnisync.com",
		FirstName: "John",
		LastName:  "Doe",
		RoleID:    roleAdmin.ID,
		Role:      roleAdmin,
	}

	// 1. Test generateToken
	tokenString, err := generateToken(user)
	if err != nil {
		t.Fatalf("unexpected error during token generation: %v", err)
	}
	if tokenString == "" {
		t.Fatal("token should not be empty")
	}

	// 2. Parse and verify token claims
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		t.Fatalf("failed to parse generated token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token should be valid")
	}

	if claims.UserID != user.ID || claims.Email != user.Email || claims.Role != "System Admin" {
		t.Errorf("unexpected token claims: %+v", claims)
	}

	// Verify expiration is set in the future (~24 hours)
	expTime := claims.ExpiresAt.Time
	if expTime.Before(time.Now().Add(23 * time.Hour)) {
		t.Errorf("token expiration time is too early: %v", expTime)
	}
}

func TestVerifyTokenEndpoint(t *testing.T) {
	setupAuthTestDB(t)

	role := models.Role{ID: uuid.New().String(), Name: "POS"}
	_ = database.DB.Create(&role)

	user := models.User{
		ID:        uuid.New().String(),
		Email:     "operator@omnisync.com",
		FirstName: "Alex",
		LastName:  "Operator",
		RoleID:    role.ID,
		Role:      role,
	}

	tokenString, _ := generateToken(user)

	app := fiber.New()
	app.Get("/auth/verify", VerifyToken)

	// Test 1: Verify token successfully
	req := httptest.NewRequest("GET", "/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed mock request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var verifyResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&verifyResp)
	if verifyResp["valid"] != true {
		t.Errorf("expected valid to be true, got %+v", verifyResp)
	}

	// Test 2: Verify fails with invalid token
	reqInvalid := httptest.NewRequest("GET", "/auth/verify", nil)
	reqInvalid.Header.Set("Authorization", "Bearer invalid-token-value")
	respInvalid, _ := app.Test(reqInvalid)
	if respInvalid.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid token, got %d", respInvalid.StatusCode)
	}
}

func TestAdminMiddleware(t *testing.T) {
	setupAuthTestDB(t)

	// Seed roles
	roleAdmin := models.Role{ID: uuid.New().String(), Name: "System Admin"}
	roleOperator := models.Role{ID: uuid.New().String(), Name: "POS"}
	database.DB.Create(&roleAdmin)
	database.DB.Create(&roleOperator)

	adminUser := models.User{
		ID:        uuid.New().String(),
		Email:     "admin@omnisync.com",
		RoleID:    roleAdmin.ID,
		Role:      roleAdmin,
		FirstName: "Admin",
	}
	operatorUser := models.User{
		ID:        uuid.New().String(),
		Email:     "operator@omnisync.com",
		RoleID:    roleOperator.ID,
		Role:      roleOperator,
		FirstName: "Operator",
	}

	adminToken, _ := generateToken(adminUser)
	operatorToken, _ := generateToken(operatorUser)

	app := fiber.New()
	app.Get("/protected-admin-only", AdminMiddleware, func(c *fiber.Ctx) error {
		return c.SendString("Success")
	})

	// Test 1: Access allowed for Admin
	reqAdmin := httptest.NewRequest("GET", "/protected-admin-only", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminToken)
	respAdmin, _ := app.Test(reqAdmin)
	if respAdmin.StatusCode != http.StatusOK {
		t.Errorf("expected admin to access, got %d", respAdmin.StatusCode)
	}

	// Test 2: Access denied for Operator (403 Forbidden)
	reqOperator := httptest.NewRequest("GET", "/protected-admin-only", nil)
	reqOperator.Header.Set("Authorization", "Bearer "+operatorToken)
	respOperator, _ := app.Test(reqOperator)
	if respOperator.StatusCode != http.StatusForbidden {
		t.Errorf("expected operator to be forbidden (403), got %d", respOperator.StatusCode)
	}

	// Test 3: Unauthorized if missing token
	reqNone := httptest.NewRequest("GET", "/protected-admin-only", nil)
	respNone, _ := app.Test(reqNone)
	if respNone.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected unauthorized if missing token (401), got %d", respNone.StatusCode)
	}
}
