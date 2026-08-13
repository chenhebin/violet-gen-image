package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/config"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
	"yingyan.local/backend/internal/storage"
)

type demoCode struct {
	value   string
	credits int
	state   string
}

func Run(
	ctx context.Context,
	db *gorm.DB,
	store storage.Store,
	cfg config.Config,
	logger *slog.Logger,
) error {
	if strings.EqualFold(cfg.App.Env, "production") && !cfg.Auth.AllowAccountSeed {
		return errors.New("account seed is disabled in production; set ALLOW_ACCOUNT_SEED=true only for an intentional seeded environment")
	}
	if err := cfg.SeedAccounts.Validate(); err != nil {
		return err
	}

	passwords, err := hashPasswords(cfg.SeedAccounts, cfg.Security.BcryptCost)
	if err != nil {
		return err
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := validateSeedIdentity(tx, cfg.SeedAccounts.ClientUserEmail); err != nil {
			return err
		}
		user, err := upsertUser(tx, cfg, cfg.SeedAccounts.ClientUserEmail, passwords.user)
		if err != nil {
			return err
		}
		admin, err := requirePlatformAdmin(tx, cfg.SeedAccounts.PlatformAdminEmail)
		if err != nil {
			return err
		}
		if _, err := upsertAdmin(tx, cfg.SeedAccounts.RetouchAdminEmail, "修图操作员", model.AdminRoleRetouchOperator, passwords.retouch); err != nil {
			return err
		}
		account, err := ensureCreditAccount(tx, user.ID)
		if err != nil {
			return err
		}
		if err := seedRedemptionCodes(tx, cfg, user, admin, account); err != nil {
			return err
		}
		provider, err := seedProvider(tx, cfg)
		if err != nil {
			return err
		}
		if err := seedDemoWorkspace(ctx, tx, store, user, admin, provider); err != nil {
			return err
		}
		logger.Info("account_seed_complete",
			"user", cfg.SeedAccounts.ClientUserEmail,
			"admin", cfg.SeedAccounts.PlatformAdminEmail,
			"retouch_operator", cfg.SeedAccounts.RetouchAdminEmail,
			"redemption_codes", 5,
		)
		return nil
	})
}

type passwordHashes struct {
	user    string
	retouch string
}

func hashPasswords(credentials config.SeedAccountsConfig, cost int) (passwordHashes, error) {
	source := []string{
		credentials.ClientUserPassword,
		credentials.RetouchAdminPassword,
	}
	result := make([]string, len(source))
	for index, password := range source {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
		if err != nil {
			return passwordHashes{}, fmt.Errorf("hash seeded account password: %w", err)
		}
		result[index] = string(hash)
	}
	return passwordHashes{user: result[0], retouch: result[1]}, nil
}

func validateSeedIdentity(db *gorm.DB, configuredEmail string) error {
	var seededAsset model.Asset
	err := db.Where("object_key = ?", "demo/source-portrait.png").First(&seededAsset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if seededAsset.OwnerUserID == nil {
		return errors.New("existing seeded workspace has no owner; refusing to change the seeded client account")
	}
	var owner model.User
	if err := db.First(&owner, "id = ?", *seededAsset.OwnerUserID).Error; err != nil {
		return fmt.Errorf("load existing seeded client account: %w", err)
	}
	if !strings.EqualFold(owner.Email, configuredEmail) {
		return fmt.Errorf("CLIENT_USER_EMAIL must remain %q for the existing seeded workspace", owner.Email)
	}
	return nil
}

func requirePlatformAdmin(db *gorm.DB, email string) (model.AdminAccount, error) {
	var admin model.AdminAccount
	if err := db.Where("email = ?", email).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.AdminAccount{}, errors.New("PLATFORM_ADMIN_EMAIL must identify an existing platform administrator; run bootstrap-admin first")
		}
		return model.AdminAccount{}, err
	}
	if admin.Role != model.AdminRolePlatformAdmin {
		return model.AdminAccount{}, errors.New("PLATFORM_ADMIN_EMAIL does not identify a platform administrator")
	}
	if admin.Status != model.UserStatusActive {
		return model.AdminAccount{}, errors.New("PLATFORM_ADMIN_EMAIL identifies a disabled platform administrator")
	}
	return admin, nil
}

func upsertUser(db *gorm.DB, cfg config.Config, email, passwordHash string) (model.User, error) {
	var user model.User
	err := db.Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{
			Email: email, PasswordHash: passwordHash, Status: model.UserStatusActive,
			TermsVersion: cfg.Auth.TermsVersion, TermsAcceptedAt: time.Now().UTC(),
		}
		err = db.Create(&user).Error
	} else if err == nil {
		err = db.Model(&user).Updates(map[string]any{
			"password_hash": passwordHash, "status": model.UserStatusActive,
			"password_reset_required": false, "temporary_password_until": nil,
		}).Error
	}
	return user, err
}

func upsertAdmin(db *gorm.DB, email, name, role, passwordHash string) (model.AdminAccount, error) {
	var admin model.AdminAccount
	err := db.Where("email = ?", email).First(&admin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		admin = model.AdminAccount{
			Email: email, Name: name, Role: role, PasswordHash: passwordHash, Status: model.UserStatusActive,
		}
		err = db.Create(&admin).Error
	} else if err == nil {
		if admin.Role != role {
			return model.AdminAccount{}, fmt.Errorf("administrator %q already exists with role %q; refusing to change it to %q", email, admin.Role, role)
		}
		err = db.Model(&admin).Updates(map[string]any{
			"name": name, "role": role, "password_hash": passwordHash,
			"status": model.UserStatusActive, "password_reset_required": false,
			"temporary_password_until": nil,
		}).Error
	}
	return admin, err
}

func ensureCreditAccount(db *gorm.DB, userID string) (model.CreditAccount, error) {
	var account model.CreditAccount
	err := db.Where("user_id = ?", userID).First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		account = model.CreditAccount{UserID: userID, Balance: 0, Version: 1}
		err = db.Create(&account).Error
	}
	return account, err
}

func seedRedemptionCodes(db *gorm.DB, cfg config.Config, user model.User, admin model.AdminAccount, account model.CreditAccount) error {
	var batch model.RedemptionBatch
	err := db.Where("name = ? AND product_code = ?", "演示兑换码", "xianyu-demo").First(&batch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		expires := time.Now().UTC().Add(90 * 24 * time.Hour)
		batch = model.RedemptionBatch{
			Name: "演示兑换码", Quantity: 5, CreditsPerCode: 10, ProductCode: "xianyu-demo",
			ExpiresAt: &expires, Notes: "仅供本地联调", CreatedBy: admin.ID,
		}
		if err := db.Create(&batch).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	now := time.Now().UTC()
	expiredAt := now.Add(-24 * time.Hour)
	future := now.Add(90 * 24 * time.Hour)
	codes := []demoCode{
		{value: "YINGYAN-START-10", credits: 10, state: "unused"},
		{value: "YINGYAN-PRO-30", credits: 30, state: "unused"},
		{value: "YINGYAN-USED-10", credits: 10, state: "redeemed"},
		{value: "YINGYAN-EXPIRED-10", credits: 10, state: "expired"},
		{value: "YINGYAN-DISABLED-10", credits: 10, state: "disabled"},
	}
	for _, seed := range codes {
		normalized := security.NormalizeRedemptionCode(seed.value)
		digest := security.HMACDigest(normalized, cfg.Security.RedemptionPepper)
		var count int64
		if err := db.Model(&model.RedemptionCode{}).Where("code_digest = ?", digest).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		ciphertext, err := security.Encrypt([]byte(normalized), cfg.Security.EncryptionKey)
		if err != nil {
			return err
		}
		code := model.RedemptionCode{
			BatchID: batch.ID, CodeDigest: digest, CodeCiphertext: ciphertext,
			MaskedCode: security.MaskSecret(normalized), Credits: seed.credits,
			ProductCode: batch.ProductCode, ExpiresAt: &future, Version: 1,
		}
		switch seed.state {
		case "expired":
			code.ExpiresAt = &expiredAt
		case "redeemed":
			code.RedeemedAt = &now
			code.RedeemedBy = &user.ID
		case "disabled":
			code.DisabledAt = &now
			code.DisabledBy = &admin.ID
			code.DisabledReason = "演示失效状态"
		}
		if err := db.Create(&code).Error; err != nil {
			return err
		}
		if seed.state == "redeemed" {
			if err := materializeDemoClaim(db, &account, code, user); err != nil {
				return err
			}
		}
	}
	return nil
}

func materializeDemoClaim(db *gorm.DB, account *model.CreditAccount, code model.RedemptionCode, user model.User) error {
	var claimCount int64
	if err := db.Model(&model.RedemptionClaim{}).Where("code_id = ?", code.ID).Count(&claimCount).Error; err != nil {
		return err
	}
	if claimCount > 0 {
		return nil
	}
	before := account.Balance
	account.Balance += code.Credits
	account.Version++
	if err := db.Model(account).Updates(map[string]any{"balance": account.Balance, "version": account.Version}).Error; err != nil {
		return err
	}
	ledger := model.CreditLedgerEntry{
		UserID: user.ID, Type: "redemption", Amount: code.Credits,
		BalanceBefore: before, BalanceAfter: account.Balance,
		BusinessType: "redemption", BusinessID: &code.ID,
		Reason: "演示兑换码核销", ReferenceNo: code.MaskedCode,
	}
	if err := db.Create(&ledger).Error; err != nil {
		return err
	}
	return db.Create(&model.RedemptionClaim{
		CodeID: code.ID, UserID: user.ID, CreditsGranted: code.Credits,
		LedgerEntryID: ledger.ID, IdempotencyKey: "demo-seed",
	}).Error
}

func seedProvider(db *gorm.DB, cfg config.Config) (model.AIProvider, error) {
	var provider model.AIProvider
	err := db.Where("code = ?", "test1").First(&provider).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ciphertext, encryptErr := security.Encrypt([]byte("sk-demo-not-a-real-key"), cfg.Security.EncryptionKey)
		if encryptErr != nil {
			return model.AIProvider{}, encryptErr
		}
		provider = model.AIProvider{
			Name: "test1", Code: "test1", Protocol: "openai-compatible",
			BaseURL: "https://example.invalid/v1", APIKeyCiphertext: ciphertext,
			APIKeyMask: "sk-d************-key", Enabled: true,
			ConnectionStatus: "untested", ConfigVersion: 1, Version: 1,
			Notes: "请在管理端替换为真实服务商配置",
		}
		if err := db.Create(&provider).Error; err != nil {
			return model.AIProvider{}, err
		}
	} else if err != nil {
		return model.AIProvider{}, err
	}

	capabilitySets := []struct {
		name, modelID, modelType string
		capabilities             map[string]bool
	}{
		{"演示对话模型", "demo-chat-model", "chat", map[string]bool{"promptOptimization": true, "visionInput": true}},
		{"演示生图模型", "demo-image-model", "image", map[string]bool{"textToImage": true, "imageToImage": true}},
	}
	for _, item := range capabilitySets {
		payload, _ := json.Marshal(item.capabilities)
		var count int64
		if err := db.Model(&model.AIModel{}).
			Where("provider_id = ? AND model_id = ?", provider.ID, item.modelID).
			Count(&count).Error; err != nil {
			return model.AIProvider{}, err
		}
		if count == 0 {
			if err := db.Create(&model.AIModel{
				ProviderID: provider.ID, DisplayName: item.name, ModelID: item.modelID,
				Type: item.modelType, Capabilities: datatypes.JSON(payload), Enabled: true,
				TestStatus: "untested", ConfigVersion: 1, Version: 1,
			}).Error; err != nil {
				return model.AIProvider{}, err
			}
		}
	}
	return provider, nil
}
