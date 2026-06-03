package param

import "time"

type AuthResult struct {
	Token string
	User  *AuthUserParam
}

type AuthUserParam struct {
	UserID              int64
	Username            string
	Email               string
	ContactNumber       *string
	Password            string
	FailedLoginAttempts int
	LockedUntil         *time.Time
}
