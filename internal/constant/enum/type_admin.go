package enum

import (
	"errors"
	"strings"
)

type UserRole int

const (
	superadmin UserRole = iota // 0
	_                          // 1: dulu admin, kini tidak dipakai (direservasi agar staff tetap 2)
	staff                      // 2
)

func (typeUser UserRole) String() string {
	switch typeUser {
	case superadmin:
		return "superadmin"
	case staff:
		return "staff"
	default:
		return "unknown"
	}
}

func ParseUserRole(roleStr string) (UserRole, error) {
	switch strings.ToLower(roleStr) {
	case "superadmin":
		return superadmin, nil
	case "staff":
		return staff, nil
	default:
		return 0, errors.New("role not valid")
	}
}
