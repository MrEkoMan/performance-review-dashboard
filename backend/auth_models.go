package main

// LoginInput is the credential payload for POST /api/auth/login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SessionUserResponse describes the logged-in account without secrets.
type SessionUserResponse struct {
	ID         int64  `json:"id"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	EngineerID *int64 `json:"engineerId,omitempty"`
	Name       string `json:"name,omitempty"`
	OpenMode   bool   `json:"openMode,omitempty"`
}

// RegisterInput is the payload for POST /api/auth/register. Role and
// EngineerID are only honored when an authenticated manager registers the
// account; the anonymous claiming flow always creates a manager.
type RegisterInput struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Role       string `json:"role,omitempty"`
	EngineerID *int64 `json:"engineerId,omitempty"`
}

// PortalAccessInput is the payload for creating or resetting an engineer's
// portal access. Password is required on create; on reset it may be omitted
// to keep the existing password.
type PortalAccessInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
