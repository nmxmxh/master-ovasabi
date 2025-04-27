package utils

import "fmt"

// GenerateTestEmail generates a unique test email address.
func GenerateTestEmail(i int) string {
	return fmt.Sprintf("test%d@example.com", i)
}

// GenerateTestPassword generates a test password.
func GenerateTestPassword() string {
	return "password123"
}

// GenerateTestName generates a test user name.
func GenerateTestName(i int) string {
	return fmt.Sprintf("Test User %d", i)
}
