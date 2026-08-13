package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
)

var ErrAdminAlreadyExists = errors.New("an administrator already exists")

type AdminInput struct {
	Email    string
	Password string
	Name     string
}

func CreateFirstAdmin(
	ctx context.Context,
	db *gorm.DB,
	input AdminInput,
	bcryptCost int,
) (model.AdminAccount, error) {
	input, err := validateAdminInput(input)
	if err != nil {
		return model.AdminAccount{}, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcryptCost)
	if err != nil {
		return model.AdminAccount{}, fmt.Errorf("hash bootstrap administrator password: %w", err)
	}

	var created model.AdminAccount
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(894_211_002)).Error; err != nil {
			return fmt.Errorf("acquire bootstrap administrator lock: %w", err)
		}

		var count int64
		if err := tx.Model(&model.AdminAccount{}).Count(&count).Error; err != nil {
			return fmt.Errorf("count administrators: %w", err)
		}
		if count > 0 {
			return ErrAdminAlreadyExists
		}

		created = model.AdminAccount{
			Email:        input.Email,
			Name:         input.Name,
			PasswordHash: string(passwordHash),
			Role:         model.AdminRolePlatformAdmin,
			Status:       model.UserStatusActive,
		}
		if err := tx.Create(&created).Error; err != nil {
			return fmt.Errorf("create bootstrap administrator: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.AdminAccount{}, err
	}
	return created, nil
}

func validateAdminInput(input AdminInput) (AdminInput, error) {
	input.Email = security.NormalizeEmail(input.Email)
	input.Name = strings.TrimSpace(input.Name)

	address, err := mail.ParseAddress(input.Email)
	if err != nil || address.Address != input.Email || len(input.Email) > 320 {
		return AdminInput{}, errors.New("PLATFORM_ADMIN_EMAIL must be a valid email address")
	}
	if len(input.Password) < 12 || len(input.Password) > 72 {
		return AdminInput{}, errors.New("PLATFORM_ADMIN_PASSWORD must contain 12 to 72 bytes")
	}
	if input.Name == "" || len([]rune(input.Name)) > 100 {
		return AdminInput{}, errors.New("PLATFORM_ADMIN_NAME must contain 1 to 100 characters")
	}
	return input, nil
}
