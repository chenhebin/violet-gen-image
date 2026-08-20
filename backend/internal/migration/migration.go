package migration

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"yingyan.local/backend/internal/model"
)

const LatestVersion = 8

const retouchTicketCreditConstraint = `ALTER TABLE retouch_tickets
	ADD CONSTRAINT retouch_ticket_credit_equation
	CHECK (
		reserved_credits >= 0 AND spent_credits >= 0 AND refunded_credits >= 0
			AND spent_credits <= reserved_credits
			AND refunded_credits <= reserved_credits
	)`

type schemaMigration struct {
	Version   int       `gorm:"primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigration) TableName() string {
	return "schema_migrations"
}

type step struct {
	version int
	name    string
	up      func(*gorm.DB) error
}

func Apply(ctx context.Context, db *gorm.DB, logger *slog.Logger) error {
	applied := make([]step, 0)
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", int64(894_211_001)).Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return fmt.Errorf("create schema migrations table: %w", err)
		}

		for _, migration := range migrationSteps() {
			var count int64
			if err := tx.Model(&schemaMigration{}).
				Where("version = ?", migration.version).Count(&count).Error; err != nil {
				return fmt.Errorf("read migration %d: %w", migration.version, err)
			}
			if count > 0 {
				continue
			}
			if err := migration.up(tx); err != nil {
				return fmt.Errorf("apply migration %d %s: %w", migration.version, migration.name, err)
			}
			if err := tx.Create(&schemaMigration{
				Version: migration.version, Name: migration.name, AppliedAt: time.Now().UTC(),
			}).Error; err != nil {
				return fmt.Errorf("record migration %d %s: %w", migration.version, migration.name, err)
			}
			applied = append(applied, migration)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, migration := range applied {
		logger.Info("migration_applied", "version", migration.version, "name", migration.name)
	}
	return nil
}

func migrationSteps() []step {
	return []step{
		{version: 1, name: "initial_schema", up: initialSchema},
		{version: 2, name: "generation_title_and_query_indexes", up: generationTitleAndQueryIndexes},
		{version: 3, name: "generation_model_display_name_snapshot", up: generationModelDisplayNameSnapshot},
		{version: 4, name: "retouch_credit_equation", up: retouchCreditEquation},
		{version: 5, name: "generation_deadlines", up: generationDeadlines},
		{version: 6, name: "provider_observability", up: providerObservability},
		{version: 7, name: "ai_processing_notice", up: aiProcessingNotice},
		{version: 8, name: "retouch_quote_expiry_and_sla", up: retouchQuoteExpiryAndSLA},
	}
}

func aiProcessingNotice(db *gorm.DB) error {
	return db.AutoMigrate(&model.UserAIProcessingNotice{})
}

func retouchQuoteExpiryAndSLA(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE retouch_quotes ADD COLUMN IF NOT EXISTS expires_at timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP + interval '48 hours')`,
		`CREATE INDEX IF NOT EXISTS idx_retouch_quotes_expires_at ON retouch_quotes (expires_at)`,
		`ALTER TABLE retouch_tickets ADD COLUMN IF NOT EXISTS quote_due_at timestamptz`,
		`ALTER TABLE retouch_tickets ADD COLUMN IF NOT EXISTS first_delivery_due_at timestamptz`,
		`ALTER TABLE retouch_tickets ADD COLUMN IF NOT EXISTS revision_due_at timestamptz`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func initialSchema(db *gorm.DB) error {
	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto`).Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return err
	}

	statements := []string{
		`ALTER TABLE credit_reservations
			ADD CONSTRAINT fk_credit_reservations_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE credit_ledger_entries
			ADD CONSTRAINT fk_credit_ledger_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_batches
			ADD CONSTRAINT fk_redemption_batches_admin
			FOREIGN KEY (created_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_codes
			ADD CONSTRAINT fk_redemption_codes_batch
			FOREIGN KEY (batch_id) REFERENCES redemption_batches(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_codes
			ADD CONSTRAINT fk_redemption_codes_redeemed_user
			FOREIGN KEY (redeemed_by) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_codes
			ADD CONSTRAINT fk_redemption_codes_disabled_admin
			FOREIGN KEY (disabled_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_claims
			ADD CONSTRAINT fk_redemption_claims_code
			FOREIGN KEY (code_id) REFERENCES redemption_codes(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_claims
			ADD CONSTRAINT fk_redemption_claims_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE redemption_claims
			ADD CONSTRAINT fk_redemption_claims_ledger
			FOREIGN KEY (ledger_entry_id) REFERENCES credit_ledger_entries(id) ON DELETE RESTRICT`,
		`ALTER TABLE assets
			ADD CONSTRAINT fk_assets_owner
			FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE assets
			ADD CONSTRAINT fk_assets_admin
			FOREIGN KEY (created_by_admin_id) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE asset_relations
			ADD CONSTRAINT fk_asset_relations_asset
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT`,
		`ALTER TABLE prompt_versions
			ADD CONSTRAINT fk_prompt_versions_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE ai_models
			ADD CONSTRAINT fk_ai_models_provider
			FOREIGN KEY (provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT`,
		`ALTER TABLE platform_model_bindings
			ADD CONSTRAINT fk_platform_bindings_provider
			FOREIGN KEY (provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT`,
		`ALTER TABLE platform_model_bindings
			ADD CONSTRAINT fk_platform_bindings_model
			FOREIGN KEY (model_id) REFERENCES ai_models(id) ON DELETE RESTRICT`,
		`ALTER TABLE platform_model_bindings
			ADD CONSTRAINT fk_platform_bindings_admin
			FOREIGN KEY (bound_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE model_test_runs
			ADD CONSTRAINT fk_model_tests_provider
			FOREIGN KEY (provider_id) REFERENCES ai_providers(id) ON DELETE RESTRICT`,
		`ALTER TABLE model_test_runs
			ADD CONSTRAINT fk_model_tests_model
			FOREIGN KEY (model_id) REFERENCES ai_models(id) ON DELETE RESTRICT`,
		`ALTER TABLE model_test_runs
			ADD CONSTRAINT fk_model_tests_admin
			FOREIGN KEY (requested_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_tasks
			ADD CONSTRAINT fk_generation_tasks_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_tasks
			ADD CONSTRAINT fk_generation_tasks_prompt
			FOREIGN KEY (prompt_version_id) REFERENCES prompt_versions(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_tasks
			ADD CONSTRAINT fk_generation_tasks_reservation
			FOREIGN KEY (credit_reservation_id) REFERENCES credit_reservations(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_task_assets
			ADD CONSTRAINT fk_generation_task_assets_task
			FOREIGN KEY (task_id) REFERENCES generation_tasks(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_task_assets
			ADD CONSTRAINT fk_generation_task_assets_asset
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_outputs
			ADD CONSTRAINT fk_generation_outputs_task
			FOREIGN KEY (task_id) REFERENCES generation_tasks(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_outputs
			ADD CONSTRAINT fk_generation_outputs_asset
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_jobs
			ADD CONSTRAINT fk_generation_jobs_task
			FOREIGN KEY (task_id) REFERENCES generation_tasks(id) ON DELETE RESTRICT`,
		`ALTER TABLE generation_jobs
			ADD CONSTRAINT fk_generation_jobs_output
			FOREIGN KEY (output_id) REFERENCES generation_outputs(id) ON DELETE RESTRICT`,
		`ALTER TABLE provider_attempts
			ADD CONSTRAINT fk_provider_attempts_job
			FOREIGN KEY (job_id) REFERENCES generation_jobs(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_tickets
			ADD CONSTRAINT fk_retouch_tickets_user
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_tickets
			ADD CONSTRAINT fk_retouch_tickets_task
			FOREIGN KEY (task_id) REFERENCES generation_tasks(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_tickets
			ADD CONSTRAINT fk_retouch_tickets_reservation
			FOREIGN KEY (credit_reservation_id) REFERENCES credit_reservations(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_quotes
			ADD CONSTRAINT fk_retouch_quotes_ticket
			FOREIGN KEY (ticket_id) REFERENCES retouch_tickets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_quotes
			ADD CONSTRAINT fk_retouch_quotes_admin
			FOREIGN KEY (created_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_tickets
			ADD CONSTRAINT fk_retouch_tickets_current_quote
			FOREIGN KEY (current_quote_id) REFERENCES retouch_quotes(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_revisions
			ADD CONSTRAINT fk_retouch_revisions_ticket
			FOREIGN KEY (ticket_id) REFERENCES retouch_tickets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_selected_results
			ADD CONSTRAINT fk_retouch_selected_ticket
			FOREIGN KEY (ticket_id) REFERENCES retouch_tickets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_selected_results
			ADD CONSTRAINT fk_retouch_selected_asset
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_deliverables
			ADD CONSTRAINT fk_retouch_deliverables_ticket
			FOREIGN KEY (ticket_id) REFERENCES retouch_tickets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_deliverables
			ADD CONSTRAINT fk_retouch_deliverables_asset
			FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_deliverables
			ADD CONSTRAINT fk_retouch_deliverables_admin
			FOREIGN KEY (created_by) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE retouch_events
			ADD CONSTRAINT fk_retouch_events_ticket
			FOREIGN KEY (ticket_id) REFERENCES retouch_tickets(id) ON DELETE RESTRICT`,
		`ALTER TABLE audit_logs
			ADD CONSTRAINT fk_audit_logs_admin
			FOREIGN KEY (admin_id) REFERENCES admin_accounts(id) ON DELETE RESTRICT`,
		`ALTER TABLE credit_reservations
			ADD CONSTRAINT credit_reservation_amounts_valid
			CHECK (
				settled_amount >= 0 AND released_amount >= 0 AND refunded_amount >= 0
				AND settled_amount + released_amount <= amount
				AND refunded_amount <= settled_amount
			)`,
		`ALTER TABLE credit_ledger_entries
			ADD CONSTRAINT credit_ledger_balance_equation
			CHECK (balance_after = balance_before + amount)`,
		`ALTER TABLE generation_tasks
			ADD CONSTRAINT generation_task_credit_equation
			CHECK (
				reserved_credits >= 0 AND spent_credits >= 0 AND refunded_credits >= 0
				AND spent_credits + refunded_credits <= reserved_credits
			)`,
		retouchTicketCreditConstraint,
		`CREATE UNIQUE INDEX idx_retouch_active_task
			ON retouch_tickets (task_id)
			WHERE status NOT IN ('delivered', 'rejected', 'cancelled')`,
		`CREATE INDEX idx_credit_ledger_user_created_desc
			ON credit_ledger_entries (user_id, created_at DESC)`,
		`CREATE INDEX idx_generation_user_created_desc
			ON generation_tasks (user_id, created_at DESC)`,
		`CREATE INDEX idx_retouch_event_created_asc
			ON retouch_events (ticket_id, created_at ASC)`,
		`CREATE INDEX idx_audit_created_desc
			ON audit_logs (created_at DESC)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func generationTitleAndQueryIndexes(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE generation_tasks
			ADD COLUMN IF NOT EXISTS title varchar(160)`,
		`UPDATE generation_tasks AS task
			SET title = CASE
				WHEN char_length(btrim(prompt.source)) > 24
					THEN left(btrim(prompt.source), 24) || '...'
				WHEN char_length(btrim(prompt.source)) > 0
					THEN btrim(prompt.source)
				ELSE '未命名创作'
			END
			FROM prompt_versions AS prompt
			WHERE prompt.id = task.prompt_version_id
				AND (task.title IS NULL OR btrim(task.title) = '')`,
		`UPDATE generation_tasks
			SET title = '未命名创作'
			WHERE title IS NULL OR btrim(title) = ''`,
		`ALTER TABLE generation_tasks
			ALTER COLUMN title SET NOT NULL`,
		`DROP INDEX IF EXISTS idx_credit_ledger_user_created`,
		`DROP INDEX IF EXISTS idx_generation_user_created`,
		`DROP INDEX IF EXISTS idx_retouch_event_created`,
		`CREATE INDEX IF NOT EXISTS idx_credit_ledger_user_created_desc
			ON credit_ledger_entries (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_generation_user_created_desc
			ON generation_tasks (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_retouch_event_created_asc
			ON retouch_events (ticket_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_generation_status_created_desc
			ON generation_tasks (status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_generation_provider_created_desc
			ON generation_tasks (provider_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_generation_model_created_desc
			ON generation_tasks (model_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_retouch_user_created_desc
			ON retouch_tickets (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_retouch_status_created_desc
			ON retouch_tickets (status, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_owner_created_desc
			ON assets (owner_user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_kind_created_desc
			ON assets (kind, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_redemption_codes_batch_created_desc
			ON redemption_codes (batch_id, created_at DESC)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func generationModelDisplayNameSnapshot(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE generation_tasks
			ADD COLUMN IF NOT EXISTS model_display_name_snapshot varchar(255)`,
		`UPDATE generation_tasks AS task
			SET model_display_name_snapshot = ai_model.display_name
			FROM ai_models AS ai_model
			WHERE ai_model.id = task.model_id
				AND (
					task.model_display_name_snapshot IS NULL
					OR btrim(task.model_display_name_snapshot) = ''
				)`,
		`UPDATE generation_tasks
			SET model_display_name_snapshot = model_name_snapshot
			WHERE model_display_name_snapshot IS NULL
				OR btrim(model_display_name_snapshot) = ''`,
		`ALTER TABLE generation_tasks
			ALTER COLUMN model_display_name_snapshot SET NOT NULL`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func retouchCreditEquation(db *gorm.DB) error {
	if err := db.Exec(`ALTER TABLE retouch_tickets
		DROP CONSTRAINT IF EXISTS retouch_ticket_credit_equation`).Error; err != nil {
		return err
	}
	return db.Exec(retouchTicketCreditConstraint).Error
}

func generationDeadlines(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE generation_tasks ADD COLUMN IF NOT EXISTS timed_out_at timestamptz`,
		`ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS deadline_at timestamptz`,
		`ALTER TABLE generation_jobs ADD COLUMN IF NOT EXISTS timeout_reason varchar(80)`,
		`CREATE INDEX IF NOT EXISTS idx_generation_jobs_deadline ON generation_jobs (status, deadline_at)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func providerObservability(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE ai_providers ADD COLUMN IF NOT EXISTS last_test_details jsonb`,
		`ALTER TABLE ai_models ADD COLUMN IF NOT EXISTS last_test_details jsonb`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS operation varchar(40) NOT NULL DEFAULT ''`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS http_method varchar(10) NOT NULL DEFAULT ''`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS endpoint_path varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS model_name varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS response_status integer NOT NULL DEFAULT 0`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS error_kind varchar(40)`,
		`ALTER TABLE provider_attempts ADD COLUMN IF NOT EXISTS request_summary jsonb`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
