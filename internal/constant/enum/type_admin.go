package enum

type UserRole int

const (
	superadmin UserRole = iota
	admin
	staff
)

func (typeUser UserRole) String() string {
	switch typeUser {
	case superadmin:
		return "superadmin"
	case admin:
		return "admin"
	case staff:
		return "staff"
	default:
		return "unknown"
	}
}
