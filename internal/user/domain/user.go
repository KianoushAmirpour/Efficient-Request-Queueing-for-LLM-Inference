package domain

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	UserID string
	Role   UserRole
	Tier   string
}

type UpdateUserTierRequest struct {
	UserID string
	Tier   string
}
