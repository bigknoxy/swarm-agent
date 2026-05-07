package roles

import "fmt"

type Role string

const (
	Architect Role = "architect"
	Developer Role = "developer"
	Utility   Role = "utility"
)

type RoleConfig struct {
	Name        Role
	Prompt      string
	Temperature float64
}

var RoleRegistry = map[Role]RoleConfig{
	Architect: {
		Name:        Architect,
		Prompt:      "You are an architect. Focus on system design, blueprints, and architecture.",
		Temperature: 0.0,
	},
	Developer: {
		Name:        Developer,
		Prompt:      "You are a developer. Focus on implementation, code, and testing.",
		Temperature: 0.7,
	},
	Utility: {
		Name:        Utility,
		Prompt:      "You are a utility. Focus on cleanup, logistics, and project state.",
		Temperature: 0.2,
	},
}

func GetRoleConfig(roleName string) (RoleConfig, error) {
	role := Role(roleName)
	if config, ok := RoleRegistry[role]; ok {
		return config, nil
	}
	return RoleConfig{}, fmt.Errorf("role not found: %s", roleName)
}
