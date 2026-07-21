package model

import "github.com/google/uuid"

type Permission string

type Role struct {
	ID          uuid.UUID
	Name        string
	Permissions []Permission
}

// HasPermission reports whether the role grants the given permission,
// honoring wildcards: "*", "org:*", "*:read".
func (r Role) HasPermission(want Permission) bool {
	for _, p := range r.Permissions {
		s := string(p)
		switch {
		case s == "*":
			return true
		case s == string(want):
			return true
		case isPrefixWildcard(s, string(want)):
			return true
		}
	}
	return false
}

// PermissionsMatch reports whether any of the roles grant the wanted permission.
func PermissionsMatch(roles []Role, want Permission) bool {
	for _, r := range roles {
		if r.HasPermission(want) {
			return true
		}
	}
	return false
}

// isPrefixWildcard matches "org:*" against "org:read", or "*:read" against "user:read".
func isPrefixWildcard(pattern, want string) bool {
	n := len(pattern)
	if n < 2 || pattern[n-1] != '*' {
		return false
	}
	prefix := pattern[:n-1]
	if len(prefix) == 0 || prefix == ":" {
		return false
	}
	return len(want) >= len(prefix) && want[:len(prefix)] == prefix
}
