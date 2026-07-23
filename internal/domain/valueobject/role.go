package valueobject

// Role — роль участника внутри команды.
type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// CanManageMembers сообщает, может ли роль приглашать/управлять участниками команды.
func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}
