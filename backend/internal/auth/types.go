package auth

import "time"

const (
	RealmUser   = "user"
	RealmManage = "manage"

	PermissionPlatformManage = "platform:manage"
	PermissionRetouchManage  = "retouch:manage"
)

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type RegisterInput struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Remember      bool   `json:"remember"`
	AcceptedTerms bool   `json:"acceptedTerms"`
}

type UserDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminSessionDTO struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	Permissions []string  `json:"permissions"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	CSRFToken   string    `json:"csrfToken"`
}

type SessionToken struct {
	Raw       string
	ExpiresAt time.Time
	Remember  bool
}

type UserPrincipal struct {
	SessionID string
	User      UserDTO
	RawToken  string
}

type AdminPrincipal struct {
	SessionID string
	Admin     AdminSessionDTO
	RawToken  string
	CSRFToken string
}
