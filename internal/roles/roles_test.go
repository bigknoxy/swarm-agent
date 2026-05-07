package roles

import (
	"strings"
	"testing"
)

func TestGetRoleConfig_Architect(t *testing.T) {
	cfg, err := GetRoleConfig("architect")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Name != Architect {
		t.Errorf("expected Architect, got %s", cfg.Name)
	}
	if cfg.Temperature != 0.0 {
		t.Errorf("expected temp 0.0, got %f", cfg.Temperature)
	}
	if cfg.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
}

func TestGetRoleConfig_Developer(t *testing.T) {
	cfg, err := GetRoleConfig("developer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("expected temp 0.7, got %f", cfg.Temperature)
	}
}

func TestGetRoleConfig_Utility(t *testing.T) {
	cfg, err := GetRoleConfig("utility")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Temperature != 0.2 {
		t.Errorf("expected temp 0.2, got %f", cfg.Temperature)
	}
}

func TestGetRoleConfig_Invalid(t *testing.T) {
	_, err := GetRoleConfig("invalid_role")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestGetRoleConfig_CaseSensitive(t *testing.T) {
	_, err := GetRoleConfig("Architect")
	if err == nil {
		t.Fatal("expected error — role names are lowercase")
	}
}

func TestRoleRegistry_Count(t *testing.T) {
	if len(RoleRegistry) != 3 {
		t.Errorf("expected 3 roles, got %d", len(RoleRegistry))
	}
}
