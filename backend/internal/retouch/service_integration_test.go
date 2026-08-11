package retouch

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/migration"
	"yingyan.local/backend/internal/model"
)

func TestAcceptedTicketCanReleaseReservedCredits(t *testing.T) {
	db := openRetouchIntegrationDB(t)
	service := New(db, credit.New(db), asset.New(db, nil), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("user cancel", func(t *testing.T) {
		scenario := seedAcceptedTicket(t, db)
		ticket, balance, err := service.Cancel(
			ctx,
			scenario.userID,
			scenario.ticketID,
			"cancel-"+uuid.NewString(),
		)
		if err != nil {
			t.Fatalf("cancel accepted ticket: %v", err)
		}
		if ticket.Status != StatusCancelled || ticket.RefundedCredits != scenario.credits {
			t.Fatalf("unexpected cancelled ticket: %#v", ticket)
		}
		if balance != scenario.credits {
			t.Fatalf("balance = %d, want %d", balance, scenario.credits)
		}
		assertReleasedReservation(t, db, scenario)
	})

	t.Run("admin fulfillment failure", func(t *testing.T) {
		scenario := seedAcceptedTicket(t, db)
		ticket, err := service.Fail(
			ctx,
			uuid.NewString(),
			scenario.ticketID,
			"无法履约",
			"fail-"+uuid.NewString(),
		)
		if err != nil {
			t.Fatalf("fail accepted ticket: %v", err)
		}
		if ticket.Status != StatusRejected || ticket.RefundedCredits != scenario.credits {
			t.Fatalf("unexpected failed ticket: %#v", ticket)
		}
		var account model.CreditAccount
		if err := db.First(&account, "user_id = ?", scenario.userID).Error; err != nil {
			t.Fatalf("load credit account: %v", err)
		}
		if account.Balance != scenario.credits {
			t.Fatalf("balance = %d, want %d", account.Balance, scenario.credits)
		}
		assertReleasedReservation(t, db, scenario)
	})
}

type acceptedTicketScenario struct {
	userID        string
	ticketID      string
	reservationID string
	credits       int
}

func seedAcceptedTicket(t *testing.T, db *gorm.DB) acceptedTicketScenario {
	t.Helper()
	now := time.Now().UTC()
	userID := uuid.NewString()
	taskID := uuid.NewString()
	ticketID := uuid.NewString()
	promptID := uuid.NewString()
	generationReservationID := uuid.NewString()
	retouchReservationID := uuid.NewString()
	credits := 3

	user := model.User{
		BaseModel:       model.BaseModel{ID: userID},
		Email:           userID + "@integration.local",
		PasswordHash:    "integration-test-only",
		Status:          model.UserStatusActive,
		TermsVersion:    "test",
		TermsAcceptedAt: now,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&model.CreditAccount{
		UserID:  userID,
		Balance: 0,
		Version: 1,
	}).Error; err != nil {
		t.Fatalf("create credit account: %v", err)
	}
	if err := db.Create(&model.PromptVersion{
		BaseModel:       model.BaseModel{ID: promptID},
		UserID:          userID,
		Source:          "integration prompt",
		Mode:            "text-to-image",
		Sections:        datatypes.JSON([]byte(`{}`)),
		SourceAssetIDs:  datatypes.JSON([]byte(`[]`)),
		ReferenceAssets: datatypes.JSON([]byte(`[]`)),
		Status:          "confirmed",
		ConfirmedAt:     &now,
	}).Error; err != nil {
		t.Fatalf("create prompt version: %v", err)
	}
	if err := db.Create(&model.CreditReservation{
		BaseModel:     model.BaseModel{ID: generationReservationID},
		UserID:        userID,
		BusinessType:  "generation",
		BusinessID:    taskID,
		Amount:        1,
		SettledAmount: 1,
		Status:        credit.ReservationSettled,
		Version:       1,
	}).Error; err != nil {
		t.Fatalf("create generation reservation: %v", err)
	}
	if err := db.Create(&model.GenerationTask{
		BaseModel:                model.BaseModel{ID: taskID},
		UserID:                   userID,
		PromptVersionID:          promptID,
		Title:                    "integration task",
		Mode:                     "text-to-image",
		Status:                   "completed",
		Settings:                 datatypes.JSON([]byte(`{}`)),
		OutputCount:              1,
		CompletedOutputs:         1,
		ReservedCredits:          1,
		SpentCredits:             1,
		CreditReservationID:      generationReservationID,
		ProviderID:               uuid.NewString(),
		ModelID:                  uuid.NewString(),
		ProviderNameSnapshot:     "integration",
		ProviderBaseURLSnapshot:  "https://provider.invalid/v1",
		APIKeyCiphertextSnapshot: []byte("integration"),
		ModelNameSnapshot:        "image-model",
		ModelDisplayNameSnapshot: "Image Model",
		ProviderConfigVersion:    1,
		ModelConfigVersion:       1,
		Version:                  1,
	}).Error; err != nil {
		t.Fatalf("create generation task: %v", err)
	}
	if err := db.Create(&model.CreditReservation{
		BaseModel:    model.BaseModel{ID: retouchReservationID},
		UserID:       userID,
		BusinessType: "retouch",
		BusinessID:   ticketID,
		Amount:       credits,
		Status:       credit.ReservationActive,
		Version:      1,
	}).Error; err != nil {
		t.Fatalf("create retouch reservation: %v", err)
	}
	if err := db.Create(&model.RetouchTicket{
		BaseModel:           model.BaseModel{ID: ticketID},
		TicketNo:            "RT-TEST-" + strings.ToUpper(strings.ReplaceAll(ticketID, "-", ""))[:8],
		UserID:              userID,
		TaskID:              taskID,
		Status:              StatusAccepted,
		Requirements:        "integration retouch",
		CreditReservationID: &retouchReservationID,
		ReservedCredits:     credits,
		SpentCredits:        0,
		RefundedCredits:     0,
		AcceptedAt:          &now,
		Version:             1,
	}).Error; err != nil {
		t.Fatalf("create accepted ticket: %v", err)
	}
	return acceptedTicketScenario{
		userID: userID, ticketID: ticketID,
		reservationID: retouchReservationID, credits: credits,
	}
}

func assertReleasedReservation(t *testing.T, db *gorm.DB, scenario acceptedTicketScenario) {
	t.Helper()
	var reservation model.CreditReservation
	if err := db.First(&reservation, "id = ?", scenario.reservationID).Error; err != nil {
		t.Fatalf("load reservation: %v", err)
	}
	if reservation.Status != credit.ReservationReleased ||
		reservation.ReleasedAmount != scenario.credits ||
		reservation.SettledAmount != 0 ||
		reservation.RefundedAmount != 0 {
		t.Fatalf("unexpected released reservation: %#v", reservation)
	}
}

func openRetouchIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}

	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	schema := "retouch_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := base.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create integration schema: %v", err)
	}
	t.Cleanup(func() {
		if err := base.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Errorf("drop integration schema: %v", err)
		}
		closeGorm(t, base)
	})

	schemaDSN, err := withSearchPath(dsn, schema)
	if err != nil {
		t.Fatalf("set integration search path: %v", err)
	}
	db, err := gorm.Open(postgres.Open(schemaDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect integration schema: %v", err)
	}
	t.Cleanup(func() { closeGorm(t, db) })
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := migration.Apply(context.Background(), db, logger); err != nil {
		t.Fatalf("apply integration migrations: %v", err)
	}
	return db
}

func withSearchPath(dsn string, schema string) (string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func closeGorm(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Errorf("resolve database pool: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		t.Errorf("close database pool: %v", err)
	}
}
