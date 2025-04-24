package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSONMarshalling(t *testing.T) {
	t0 := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
	u := User{
		ID:        "123",
		Username:  "johndoe",
		Email:     "john@example.com",
		Password:  "secret",
		Roles:     []string{"admin", "user"},
		CreatedAt: t0,
		UpdatedAt: t0.Add(time.Hour),
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal into a map to inspect fields
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal to map failed: %v", err)
	}

	// Password should not be present in JSON
	if _, ok := m["password"]; ok {
		t.Error("Password field should not be in JSON")
	}

	if m["id"] != u.ID {
		t.Errorf("Expected id %q, got %v", u.ID, m["id"])
	}
	if m["username"] != u.Username {
		t.Errorf("Expected username %q, got %v", u.Username, m["username"])
	}
	if m["email"] != u.Email {
		t.Errorf("Expected email %q, got %v", u.Email, m["email"])
	}

	// Check roles
	roles, ok := m["roles"].([]interface{})
	if !ok {
		t.Fatalf("Expected roles to be a slice, got %T", m["roles"])
	}
	if len(roles) != len(u.Roles) {
		t.Errorf("Expected roles length %d, got %d", len(u.Roles), len(roles))
	}
	for i, r := range roles {
		if r != u.Roles[i] {
			t.Errorf("Expected roles[%d] %q, got %v", i, u.Roles[i], r)
		}
	}

	// Check timestamps are strings
	if _, ok := m["created_at"].(string); !ok {
		t.Errorf("Expected created_at to be a string, got %T", m["created_at"])
	}
	if _, ok := m["updated_at"].(string); !ok {
		t.Errorf("Expected updated_at to be a string, got %T", m["updated_at"])
	}
}

func TestUserJSONUnmarshalIgnoresPassword(t *testing.T) {
	jsonStr := `{
        "id":"123",
        "username":"johndoe",
        "email":"john@example.com",
        "password":"secret",
        "roles":["admin","user"],
        "created_at":"2023-01-01T12:00:00Z",
        "updated_at":"2023-01-01T13:00:00Z"
    }`
	var u User
	if err := json.Unmarshal([]byte(jsonStr), &u); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if u.Password != "" {
		t.Errorf("Expected password to be empty, got %q", u.Password)
	}

	if u.ID != "123" {
		t.Errorf("Expected ID '123', got %q", u.ID)
	}
	if u.Username != "johndoe" {
		t.Errorf("Expected Username 'johndoe', got %q", u.Username)
	}
	if u.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got %q", u.Email)
	}

	expectedCreated := time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)
	if !u.CreatedAt.Equal(expectedCreated) {
		t.Errorf("Expected CreatedAt %v, got %v", expectedCreated, u.CreatedAt)
	}
	expectedUpdated := time.Date(2023, time.January, 1, 13, 0, 0, 0, time.UTC)
	if !u.UpdatedAt.Equal(expectedUpdated) {
		t.Errorf("Expected UpdatedAt %v, got %v", expectedUpdated, u.UpdatedAt)
	}

	if len(u.Roles) != 2 || u.Roles[0] != "admin" || u.Roles[1] != "user" {
		t.Errorf("Expected Roles ['admin','user'], got %v", u.Roles)
	}
}
