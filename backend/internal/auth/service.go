package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
	now func() time.Time
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg, now: time.Now}
}

func (s *Service) Register(ctx context.Context, input RegisterInput, ipAddress, userAgent string) (UserDTO, SessionToken, error) {
	email, err := validateCredentials(input.Email, input.Password)
	if err != nil {
		return UserDTO{}, SessionToken{}, err
	}
	if !input.AcceptedTerms {
		return UserDTO{}, SessionToken{}, apierror.Invalid("请先同意用户协议和隐私政策", map[string]string{"acceptedTerms": "required"})
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.cfg.Security.BcryptCost)
	if err != nil {
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}
	token, err := s.newSessionToken(input.Remember, false)
	if err != nil {
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}

	now := s.now().UTC()
	user := model.User{
		Email:           email,
		PasswordHash:    string(passwordHash),
		Status:          model.UserStatusActive,
		TermsVersion:    s.cfg.Auth.TermsVersion,
		TermsAcceptedAt: now,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return apierror.Invalid("该邮箱已注册", map[string]string{"email": "already_registered"})
			}
			return err
		}
		if err := tx.Create(&model.CreditAccount{UserID: user.ID, Balance: 0, Version: 1}).Error; err != nil {
			return err
		}
		session := s.userSession(user.ID, token, now, ipAddress, userAgent)
		return tx.Create(&session).Error
	})
	if err != nil {
		if _, ok := err.(*apierror.AppError); ok {
			return UserDTO{}, SessionToken{}, err
		}
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}
	return userDTO(user), token, nil
}

func (s *Service) LoginUser(ctx context.Context, input LoginInput, ipAddress, userAgent string) (UserDTO, SessionToken, error) {
	email, err := validateCredentials(input.Email, input.Password)
	if err != nil {
		return UserDTO{}, SessionToken{}, err
	}

	var user model.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(input.Password))
			return UserDTO{}, SessionToken{}, apierror.InvalidCredentials()
		}
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return UserDTO{}, SessionToken{}, apierror.InvalidCredentials()
	}
	if user.Status != model.UserStatusActive {
		return UserDTO{}, SessionToken{}, apierror.Disabled()
	}
	if user.PasswordResetRequired && user.TemporaryPasswordUntil != nil && user.TemporaryPasswordUntil.Before(s.now()) {
		return UserDTO{}, SessionToken{}, apierror.InvalidCredentials()
	}

	token, err := s.newSessionToken(input.Remember, false)
	if err != nil {
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}
	session := s.userSession(user.ID, token, s.now().UTC(), ipAddress, userAgent)
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return UserDTO{}, SessionToken{}, apierror.Internal(err)
	}
	return userDTO(user), token, nil
}

func (s *Service) LoginAdmin(ctx context.Context, input LoginInput, ipAddress, userAgent string) (AdminSessionDTO, SessionToken, error) {
	email, err := validateCredentials(input.Email, input.Password)
	if err != nil {
		return AdminSessionDTO{}, SessionToken{}, err
	}

	var admin model.AdminAccount
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(input.Password))
			return AdminSessionDTO{}, SessionToken{}, apierror.InvalidCredentials()
		}
		return AdminSessionDTO{}, SessionToken{}, apierror.Internal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.Password)) != nil {
		return AdminSessionDTO{}, SessionToken{}, apierror.InvalidCredentials()
	}
	if admin.Status != model.UserStatusActive {
		return AdminSessionDTO{}, SessionToken{}, apierror.Disabled()
	}

	token, err := s.newSessionToken(input.Remember, true)
	if err != nil {
		return AdminSessionDTO{}, SessionToken{}, apierror.Internal(err)
	}
	session := s.adminSession(admin.ID, token, s.now().UTC(), ipAddress, userAgent)
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return AdminSessionDTO{}, SessionToken{}, apierror.Internal(err)
	}
	csrf := s.csrfToken(token.Raw)
	return adminDTO(admin, csrf), token, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, rawToken string) (UserPrincipal, error) {
	if rawToken == "" {
		return UserPrincipal{}, apierror.AuthRequired()
	}
	digest := security.HMACDigest(rawToken, s.cfg.Security.TokenPepper)
	now := s.now().UTC()
	var session model.UserSession
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("token_digest = ? AND revoked_at IS NULL AND expires_at > ?", digest, now).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return UserPrincipal{}, apierror.AuthRequired()
		}
		return UserPrincipal{}, apierror.Internal(err)
	}
	if session.User.Status != model.UserStatusActive {
		return UserPrincipal{}, apierror.Disabled()
	}
	s.touchUserSession(ctx, &session, now)
	return UserPrincipal{SessionID: session.ID, User: userDTO(session.User), RawToken: rawToken}, nil
}

func (s *Service) AuthenticateAdmin(ctx context.Context, rawToken string) (AdminPrincipal, error) {
	if rawToken == "" {
		return AdminPrincipal{}, apierror.AuthRequired()
	}
	digest := security.HMACDigest(rawToken, s.cfg.Security.TokenPepper)
	now := s.now().UTC()
	var session model.AdminSession
	err := s.db.WithContext(ctx).
		Preload("Admin").
		Where("token_digest = ? AND revoked_at IS NULL AND expires_at > ?", digest, now).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminPrincipal{}, apierror.AuthRequired()
		}
		return AdminPrincipal{}, apierror.Internal(err)
	}
	if session.Admin.Status != model.UserStatusActive {
		return AdminPrincipal{}, apierror.Disabled()
	}
	s.touchAdminSession(ctx, &session, now)
	csrf := s.csrfToken(rawToken)
	return AdminPrincipal{
		SessionID: session.ID,
		Admin:     adminDTO(session.Admin, csrf),
		RawToken:  rawToken,
		CSRFToken: csrf,
	}, nil
}

func (s *Service) LogoutUser(ctx context.Context, sessionID string) error {
	return s.revokeSession(ctx, &model.UserSession{}, sessionID)
}

func (s *Service) LogoutAdmin(ctx context.Context, sessionID string) error {
	return s.revokeSession(ctx, &model.AdminSession{}, sessionID)
}

func (s *Service) revokeSession(ctx context.Context, target any, sessionID string) error {
	now := s.now().UTC()
	if err := s.db.WithContext(ctx).Model(target).
		Where("id = ? AND revoked_at IS NULL", sessionID).
		Update("revoked_at", now).Error; err != nil {
		return apierror.Internal(err)
	}
	return nil
}

func (s *Service) newSessionToken(remember, admin bool) (SessionToken, error) {
	raw, err := security.RandomToken(32)
	if err != nil {
		return SessionToken{}, err
	}
	ttl := s.cfg.Auth.UserSessionTTL
	if remember {
		ttl = s.cfg.Auth.UserRememberTTL
	}
	if admin {
		ttl = s.cfg.Auth.AdminSessionTTL
		if remember {
			ttl = s.cfg.Auth.AdminRememberTTL
		}
	}
	return SessionToken{Raw: raw, ExpiresAt: s.now().UTC().Add(ttl), Remember: remember}, nil
}

func (s *Service) userSession(userID string, token SessionToken, now time.Time, ipAddress, userAgent string) model.UserSession {
	return model.UserSession{
		UserID:        userID,
		TokenDigest:   security.HMACDigest(token.Raw, s.cfg.Security.TokenPepper),
		ExpiresAt:     token.ExpiresAt,
		LastSeenAt:    now,
		IPAddressHash: security.HashMetadata(ipAddress, s.cfg.Security.TokenPepper),
		UserAgentHash: security.HashMetadata(userAgent, s.cfg.Security.TokenPepper),
	}
}

func (s *Service) adminSession(adminID string, token SessionToken, now time.Time, ipAddress, userAgent string) model.AdminSession {
	return model.AdminSession{
		AdminID:       adminID,
		TokenDigest:   security.HMACDigest(token.Raw, s.cfg.Security.TokenPepper),
		ExpiresAt:     token.ExpiresAt,
		LastSeenAt:    now,
		IPAddressHash: security.HashMetadata(ipAddress, s.cfg.Security.TokenPepper),
		UserAgentHash: security.HashMetadata(userAgent, s.cfg.Security.TokenPepper),
	}
}

func (s *Service) csrfToken(rawSessionToken string) string {
	return security.HMACDigest("csrf:"+rawSessionToken, s.cfg.Security.TokenPepper)
}

func (s *Service) touchUserSession(ctx context.Context, session *model.UserSession, now time.Time) {
	if now.Sub(session.LastSeenAt) < 5*time.Minute {
		return
	}
	_ = s.db.WithContext(ctx).Model(&model.UserSession{}).
		Where("id = ?", session.ID).Update("last_seen_at", now).Error
}

func (s *Service) touchAdminSession(ctx context.Context, session *model.AdminSession, now time.Time) {
	if now.Sub(session.LastSeenAt) < 5*time.Minute {
		return
	}
	_ = s.db.WithContext(ctx).Model(&model.AdminSession{}).
		Where("id = ?", session.ID).Update("last_seen_at", now).Error
}

func userDTO(user model.User) UserDTO {
	return UserDTO{ID: user.ID, Email: user.Email, Status: user.Status, CreatedAt: user.CreatedAt.UTC()}
}

func adminDTO(admin model.AdminAccount, csrf string) AdminSessionDTO {
	return AdminSessionDTO{
		ID:          admin.ID,
		Email:       admin.Email,
		Name:        admin.Name,
		Role:        admin.Role,
		Permissions: PermissionsForRole(admin.Role),
		Status:      admin.Status,
		CreatedAt:   admin.CreatedAt.UTC(),
		CSRFToken:   csrf,
	}
}

func PermissionsForRole(role string) []string {
	switch role {
	case model.AdminRolePlatformAdmin:
		return []string{PermissionPlatformManage, PermissionRetouchManage}
	case model.AdminRoleRetouchOperator:
		return []string{PermissionRetouchManage}
	default:
		return []string{}
	}
}

func HasPermission(admin AdminSessionDTO, permission string) bool {
	for _, candidate := range admin.Permissions {
		if candidate == permission {
			return true
		}
	}
	return false
}

func validateCredentials(rawEmail, password string) (string, error) {
	email := security.NormalizeEmail(rawEmail)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 320 {
		return "", apierror.Invalid("请输入有效邮箱", map[string]string{"email": "invalid"})
	}
	if len(password) < 8 || len(password) > 128 {
		return "", apierror.Invalid("密码长度需为 8 到 128 位", map[string]string{"password": "invalid_length"})
	}
	return email, nil
}

var dummyPasswordHash = []byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhiLK5aUaxc1wfcgSV5DteawQcBPLz9e")

func IsAllowedOrigin(origin string, allowedOrigins []string) bool {
	origin = strings.TrimRight(strings.TrimSpace(origin), "/")
	for _, allowed := range allowedOrigins {
		if origin == strings.TrimRight(allowed, "/") {
			return true
		}
	}
	return false
}
