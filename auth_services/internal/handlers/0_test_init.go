package handlers

import "os"

func init() {
	// Guarantee that JWT_SECRET_KEY is set before auth_handler.go's init() executes
	if os.Getenv("JWT_SECRET_KEY") == "" {
		os.Setenv("JWT_SECRET_KEY", "test-signing-key-for-auth-services-unit-tests-12345")
	}
}
