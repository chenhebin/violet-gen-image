package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"

	AdminRolePlatformAdmin   = "platform_admin"
	AdminRoleRetouchOperator = "retouch_operator"
)

type BaseModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"not null;index"`
	UpdatedAt time.Time `gorm:"not null"`
}

type User struct {
	BaseModel
	Email                  string    `gorm:"size:320;not null;uniqueIndex"`
	PasswordHash           string    `gorm:"size:255;not null"`
	Status                 string    `gorm:"size:32;not null;default:active;index"`
	TermsVersion           string    `gorm:"size:64;not null"`
	TermsAcceptedAt        time.Time `gorm:"not null"`
	DisabledAt             *time.Time
	DisabledReason         string `gorm:"size:500"`
	PasswordResetRequired  bool   `gorm:"not null;default:false"`
	TemporaryPasswordUntil *time.Time
}

type UserAIProcessingNotice struct {
	BaseModel
	UserID         string    `gorm:"type:uuid;not null;uniqueIndex"`
	NoticeVersion  string    `gorm:"size:64;not null"`
	AcknowledgedAt time.Time `gorm:"not null"`
	User           User      `gorm:"constraint:OnDelete:CASCADE"`
}

type AdminAccount struct {
	BaseModel
	Email                  string `gorm:"size:320;not null;uniqueIndex"`
	Name                   string `gorm:"size:100;not null"`
	PasswordHash           string `gorm:"size:255;not null"`
	Role                   string `gorm:"size:32;not null;index"`
	Status                 string `gorm:"size:32;not null;default:active;index"`
	DisabledAt             *time.Time
	DisabledReason         string `gorm:"size:500"`
	PasswordResetRequired  bool   `gorm:"not null;default:false"`
	TemporaryPasswordUntil *time.Time
}

type UserSession struct {
	BaseModel
	UserID        string     `gorm:"type:uuid;not null;index"`
	TokenDigest   string     `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt     time.Time  `gorm:"not null;index"`
	LastSeenAt    time.Time  `gorm:"not null"`
	RevokedAt     *time.Time `gorm:"index"`
	IPAddressHash string     `gorm:"size:64"`
	UserAgentHash string     `gorm:"size:64"`
	User          User       `gorm:"constraint:OnDelete:CASCADE"`
}

type AdminSession struct {
	BaseModel
	AdminID       string       `gorm:"type:uuid;not null;index"`
	TokenDigest   string       `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt     time.Time    `gorm:"not null;index"`
	LastSeenAt    time.Time    `gorm:"not null"`
	RevokedAt     *time.Time   `gorm:"index"`
	IPAddressHash string       `gorm:"size:64"`
	UserAgentHash string       `gorm:"size:64"`
	Admin         AdminAccount `gorm:"foreignKey:AdminID;constraint:OnDelete:CASCADE"`
}

type CreditAccount struct {
	BaseModel
	UserID  string `gorm:"type:uuid;not null;uniqueIndex"`
	Balance int    `gorm:"not null;default:0;check:balance_non_negative,balance >= 0"`
	Version int64  `gorm:"not null;default:1"`
	User    User   `gorm:"constraint:OnDelete:RESTRICT"`
}

type CreditReservation struct {
	BaseModel
	UserID         string `gorm:"type:uuid;not null;index"`
	BusinessType   string `gorm:"size:32;not null;uniqueIndex:idx_credit_reservation_business"`
	BusinessID     string `gorm:"type:uuid;not null;uniqueIndex:idx_credit_reservation_business"`
	Amount         int    `gorm:"not null;check:reservation_amount_positive,amount > 0"`
	SettledAmount  int    `gorm:"not null;default:0"`
	ReleasedAmount int    `gorm:"not null;default:0"`
	RefundedAmount int    `gorm:"not null;default:0"`
	Status         string `gorm:"size:32;not null;index"`
	SettledAt      *time.Time
	ReleasedAt     *time.Time
	RefundedAt     *time.Time
	Version        int64 `gorm:"not null;default:1"`
}

type CreditLedgerEntry struct {
	BaseModel
	UserID        string  `gorm:"type:uuid;not null"`
	Type          string  `gorm:"size:32;not null;index"`
	Amount        int     `gorm:"not null"`
	BalanceBefore int     `gorm:"not null"`
	BalanceAfter  int     `gorm:"not null;check:ledger_balance_after_non_negative,balance_after >= 0"`
	BusinessType  string  `gorm:"size:32;index"`
	BusinessID    *string `gorm:"type:uuid;index"`
	OperatorID    *string `gorm:"type:uuid;index"`
	Reason        string  `gorm:"size:500"`
	ReferenceNo   string  `gorm:"size:128;index"`
}

type RedemptionBatch struct {
	BaseModel
	Name           string     `gorm:"size:120;not null"`
	Quantity       int        `gorm:"not null;check:redemption_batch_quantity,quantity BETWEEN 1 AND 500"`
	CreditsPerCode int        `gorm:"not null;check:redemption_batch_credits,credits_per_code > 0"`
	ProductCode    string     `gorm:"size:100;not null;index"`
	ExpiresAt      *time.Time `gorm:"index"`
	Notes          string     `gorm:"size:1000"`
	CreatedBy      string     `gorm:"type:uuid;not null;index"`
}

type RedemptionCode struct {
	BaseModel
	BatchID        string     `gorm:"type:uuid;not null;index"`
	CodeDigest     string     `gorm:"size:64;not null;uniqueIndex"`
	CodeCiphertext []byte     `gorm:"type:bytea;not null"`
	MaskedCode     string     `gorm:"size:32;not null"`
	Credits        int        `gorm:"not null;check:redemption_code_credits,credits > 0"`
	ProductCode    string     `gorm:"size:100;not null;index"`
	ExpiresAt      *time.Time `gorm:"index"`
	RedeemedAt     *time.Time `gorm:"index"`
	RedeemedBy     *string    `gorm:"type:uuid;index"`
	DisabledAt     *time.Time `gorm:"index"`
	DisabledBy     *string    `gorm:"type:uuid;index"`
	DisabledReason string     `gorm:"size:500"`
	Version        int64      `gorm:"not null;default:1"`
}

type RedemptionClaim struct {
	BaseModel
	CodeID         string `gorm:"type:uuid;not null;uniqueIndex"`
	UserID         string `gorm:"type:uuid;not null;index"`
	CreditsGranted int    `gorm:"not null"`
	LedgerEntryID  string `gorm:"type:uuid;not null;uniqueIndex"`
	IdempotencyKey string `gorm:"size:128;not null"`
}

type Asset struct {
	BaseModel
	OwnerUserID       *string `gorm:"type:uuid;index"`
	CreatedByAdminID  *string `gorm:"type:uuid;index"`
	Kind              string  `gorm:"size:32;not null;index"`
	ReferenceRole     string  `gorm:"size:32;index"`
	OriginalName      string  `gorm:"size:255;not null"`
	MIMEType          string  `gorm:"size:100;not null"`
	SizeBytes         int64   `gorm:"not null;check:asset_size_non_negative,size_bytes >= 0"`
	Width             int     `gorm:"not null;default:0"`
	Height            int     `gorm:"not null;default:0"`
	SHA256            string  `gorm:"size:64;not null;index"`
	Bucket            string  `gorm:"size:128;not null"`
	ObjectKey         string  `gorm:"size:1024;not null;uniqueIndex"`
	RetainPermanently bool    `gorm:"not null;default:false;index"`
	RetainReason      string  `gorm:"size:500"`
	RetainedBy        *string `gorm:"type:uuid"`
	RetainedAt        *time.Time
	CleanedAt         *time.Time `gorm:"index"`
	CleanupReason     string     `gorm:"size:500"`
	Version           int64      `gorm:"not null;default:1"`
}

type AssetRelation struct {
	BaseModel
	AssetID      string `gorm:"type:uuid;not null;uniqueIndex:idx_asset_relation"`
	ResourceType string `gorm:"size:40;not null;uniqueIndex:idx_asset_relation"`
	ResourceID   string `gorm:"type:uuid;not null;uniqueIndex:idx_asset_relation;index"`
	RelationType string `gorm:"size:40;not null;uniqueIndex:idx_asset_relation"`
}

type PromptVersion struct {
	BaseModel
	UserID                string         `gorm:"type:uuid;not null;index"`
	Source                string         `gorm:"type:text;not null"`
	Mode                  string         `gorm:"size:32;not null"`
	Sections              datatypes.JSON `gorm:"type:jsonb;not null"`
	SourceAssetIDs        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	ReferenceAssets       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	ProviderID            *string        `gorm:"type:uuid"`
	ModelID               *string        `gorm:"type:uuid"`
	ProviderConfigVersion int64          `gorm:"not null;default:0"`
	ModelConfigVersion    int64          `gorm:"not null;default:0"`
	Status                string         `gorm:"size:32;not null;index"`
	ConfirmedAt           *time.Time
}

type AIProvider struct {
	BaseModel
	Name             string `gorm:"size:120;not null"`
	Code             string `gorm:"size:80;not null;uniqueIndex"`
	Protocol         string `gorm:"size:40;not null;default:openai-compatible"`
	BaseURL          string `gorm:"size:1000;not null"`
	APIKeyCiphertext []byte `gorm:"type:bytea;not null"`
	APIKeyMask       string `gorm:"size:64;not null"`
	Enabled          bool   `gorm:"not null;default:true;index"`
	ConnectionStatus string `gorm:"size:32;not null;default:untested;index"`
	LastTestedAt     *time.Time
	LastTestSummary  string         `gorm:"size:500"`
	LastTestDetails  datatypes.JSON `gorm:"type:jsonb"`
	ConfigVersion    int64          `gorm:"not null;default:1"`
	Notes            string         `gorm:"size:1000"`
	Version          int64          `gorm:"not null;default:1"`
}

func (AIProvider) TableName() string {
	return "ai_providers"
}

type AIModel struct {
	BaseModel
	ProviderID      string         `gorm:"type:uuid;not null;uniqueIndex:idx_provider_model"`
	DisplayName     string         `gorm:"size:120;not null"`
	ModelID         string         `gorm:"size:255;not null;uniqueIndex:idx_provider_model"`
	Type            string         `gorm:"size:32;not null;index"`
	Capabilities    datatypes.JSON `gorm:"type:jsonb;not null"`
	Enabled         bool           `gorm:"not null;default:true;index"`
	TestStatus      string         `gorm:"size:32;not null;default:untested;index"`
	LastTestedAt    *time.Time
	LastTestSummary string         `gorm:"size:500"`
	LastTestDetails datatypes.JSON `gorm:"type:jsonb"`
	ConfigVersion   int64          `gorm:"not null;default:1"`
	Version         int64          `gorm:"not null;default:1"`
}

type PlatformModelBinding struct {
	BaseModel
	BindingType           string `gorm:"size:32;not null;uniqueIndex"`
	ProviderID            string `gorm:"type:uuid;not null;index"`
	ModelID               string `gorm:"type:uuid;not null;index"`
	ProviderConfigVersion int64  `gorm:"not null"`
	ModelConfigVersion    int64  `gorm:"not null"`
	BoundBy               string `gorm:"type:uuid;not null"`
	Version               int64  `gorm:"not null;default:1"`
}

type ModelTestRun struct {
	BaseModel
	ProviderID    string  `gorm:"type:uuid;not null;index"`
	ModelID       *string `gorm:"type:uuid;index"`
	Kind          string  `gorm:"size:32;not null"`
	Status        string  `gorm:"size:32;not null;index"`
	LatencyMillis int64
	Summary       string         `gorm:"size:500"`
	Details       datatypes.JSON `gorm:"type:jsonb"`
	ConfigVersion int64          `gorm:"not null"`
	RequestedBy   string         `gorm:"type:uuid;not null"`
	CompletedAt   *time.Time
}

type GenerationTask struct {
	BaseModel
	UserID                   string         `gorm:"type:uuid;not null"`
	PromptVersionID          string         `gorm:"type:uuid;not null;index"`
	Title                    string         `gorm:"size:160;not null"`
	Mode                     string         `gorm:"size:32;not null"`
	Status                   string         `gorm:"size:32;not null;index"`
	Settings                 datatypes.JSON `gorm:"type:jsonb;not null"`
	OutputCount              int            `gorm:"not null;check:generation_output_count,output_count BETWEEN 1 AND 4"`
	CompletedOutputs         int            `gorm:"not null;default:0"`
	FailedOutputs            int            `gorm:"not null;default:0"`
	ReservedCredits          int            `gorm:"not null;default:0"`
	SpentCredits             int            `gorm:"not null;default:0"`
	RefundedCredits          int            `gorm:"not null;default:0"`
	CreditReservationID      string         `gorm:"type:uuid;not null;uniqueIndex"`
	ProviderID               string         `gorm:"type:uuid;not null"`
	ModelID                  string         `gorm:"type:uuid;not null"`
	ProviderNameSnapshot     string         `gorm:"size:120;not null"`
	ProviderBaseURLSnapshot  string         `gorm:"size:1000;not null"`
	APIKeyCiphertextSnapshot []byte         `gorm:"type:bytea;not null"`
	ModelNameSnapshot        string         `gorm:"size:255;not null"`
	ModelDisplayNameSnapshot string         `gorm:"size:255;not null"`
	ProviderConfigVersion    int64          `gorm:"not null"`
	ModelConfigVersion       int64          `gorm:"not null"`
	StartedAt                *time.Time
	CompletedAt              *time.Time
	CancelledAt              *time.Time
	TimedOutAt               *time.Time
	ErrorCode                string `gorm:"size:80"`
	ErrorSummary             string `gorm:"size:500"`
	Version                  int64  `gorm:"not null;default:1"`
}

type GenerationTaskAsset struct {
	BaseModel
	TaskID        string `gorm:"type:uuid;not null;uniqueIndex:idx_generation_task_asset"`
	AssetID       string `gorm:"type:uuid;not null;uniqueIndex:idx_generation_task_asset"`
	Usage         string `gorm:"size:32;not null"`
	ReferenceRole string `gorm:"size:32"`
}

type GenerationOutput struct {
	BaseModel
	TaskID             string  `gorm:"type:uuid;not null;uniqueIndex:idx_generation_output_index"`
	OutputIndex        int     `gorm:"not null;uniqueIndex:idx_generation_output_index"`
	Status             string  `gorm:"size:32;not null;index"`
	AssetID            *string `gorm:"type:uuid;uniqueIndex"`
	ProviderResponseID string  `gorm:"size:255"`
	ErrorCode          string  `gorm:"size:80"`
	ErrorSummary       string  `gorm:"size:500"`
	StartedAt          *time.Time
	CompletedAt        *time.Time
	Version            int64 `gorm:"not null;default:1"`
}

type GenerationJob struct {
	BaseModel
	JobType       string    `gorm:"size:40;not null;index"`
	TaskID        string    `gorm:"type:uuid;not null;index"`
	OutputID      *string   `gorm:"type:uuid;uniqueIndex"`
	Status        string    `gorm:"size:32;not null;index:idx_generation_job_claim,priority:1"`
	Attempts      int       `gorm:"not null;default:0"`
	MaxAttempts   int       `gorm:"not null;default:1"`
	AvailableAt   time.Time `gorm:"not null;index:idx_generation_job_claim,priority:2"`
	LockedBy      string    `gorm:"size:128;index"`
	LockedAt      *time.Time
	HeartbeatAt   *time.Time
	StartedAt     *time.Time
	DeadlineAt    *time.Time
	CompletedAt   *time.Time
	LastError     string `gorm:"size:1000"`
	TimeoutReason string `gorm:"size:80"`
	Version       int64  `gorm:"not null;default:1"`
}

type ProviderAttempt struct {
	BaseModel
	JobID             string `gorm:"type:uuid;not null;index;uniqueIndex:idx_provider_attempt"`
	ProviderID        string `gorm:"type:uuid;not null;index"`
	ModelID           string `gorm:"type:uuid;not null"`
	AttemptNo         int    `gorm:"not null;uniqueIndex:idx_provider_attempt"`
	Operation         string `gorm:"size:40;not null;default:''"`
	HTTPMethod        string `gorm:"size:10;not null;default:''"`
	EndpointPath      string `gorm:"size:255;not null;default:''"`
	ModelName         string `gorm:"size:255;not null;default:''"`
	Status            string `gorm:"size:32;not null;index"`
	ExternalRequestID string `gorm:"size:255"`
	RequestAccepted   *bool
	ResponseStatus    int `gorm:"not null;default:0"`
	LatencyMillis     int64
	ErrorCode         string         `gorm:"size:80"`
	ErrorKind         string         `gorm:"size:40"`
	ErrorSummary      string         `gorm:"size:500"`
	RequestSummary    datatypes.JSON `gorm:"type:jsonb"`
	ResponseMetadata  datatypes.JSON `gorm:"type:jsonb"`
	StartedAt         time.Time      `gorm:"not null"`
	CompletedAt       *time.Time
}

type OutboxEvent struct {
	BaseModel
	AggregateType string         `gorm:"size:40;not null;index"`
	AggregateID   string         `gorm:"type:uuid;not null;index"`
	EventType     string         `gorm:"size:80;not null;index"`
	Payload       datatypes.JSON `gorm:"type:jsonb;not null"`
	AvailableAt   time.Time      `gorm:"not null;index"`
	PublishedAt   *time.Time     `gorm:"index"`
	Attempts      int            `gorm:"not null;default:0"`
	LastError     string         `gorm:"size:1000"`
}

type RetouchTicket struct {
	BaseModel
	TicketNo            string  `gorm:"size:40;not null;uniqueIndex"`
	UserID              string  `gorm:"type:uuid;not null;index"`
	TaskID              string  `gorm:"type:uuid;not null;index"`
	Status              string  `gorm:"size:40;not null;index"`
	ClosureType         string  `gorm:"size:40"`
	Requirements        string  `gorm:"type:text;not null"`
	CurrentQuoteID      *string `gorm:"type:uuid"`
	CreditReservationID *string `gorm:"type:uuid;uniqueIndex"`
	ReservedCredits     int     `gorm:"not null;default:0"`
	SpentCredits        int     `gorm:"not null;default:0"`
	RefundedCredits     int     `gorm:"not null;default:0"`
	RevisionUsed        bool    `gorm:"not null;default:false"`
	ClosedReason        string  `gorm:"size:1000"`
	QuotedAt            *time.Time
	QuoteDueAt          *time.Time
	FirstDeliveryDueAt  *time.Time
	RevisionDueAt       *time.Time
	AcceptedAt          *time.Time
	StartedAt           *time.Time
	DeliveredAt         *time.Time
	ClosedAt            *time.Time
	Version             int64 `gorm:"not null;default:1"`
}

type RetouchQuote struct {
	BaseModel
	TicketID      string `gorm:"type:uuid;not null;uniqueIndex:idx_retouch_quote_version"`
	QuoteVersion  int    `gorm:"not null;uniqueIndex:idx_retouch_quote_version"`
	Credits       int    `gorm:"not null;check:retouch_quote_credits,credits BETWEEN 1 AND 999"`
	Notes         string `gorm:"size:500"`
	Status        string `gorm:"size:32;not null;index"`
	CreatedBy     string `gorm:"type:uuid;not null"`
	AcceptedAt    *time.Time
	InvalidatedAt *time.Time
	ExpiresAt     time.Time `gorm:"not null;index"`
}

type RetouchRevision struct {
	BaseModel
	TicketID     string `gorm:"type:uuid;not null;uniqueIndex"`
	Requirements string `gorm:"type:text;not null"`
	RequestedBy  string `gorm:"type:uuid;not null"`
}

type RetouchSelectedResult struct {
	BaseModel
	TicketID string `gorm:"type:uuid;not null;uniqueIndex:idx_retouch_selected"`
	AssetID  string `gorm:"type:uuid;not null;uniqueIndex:idx_retouch_selected"`
}

type RetouchDeliverable struct {
	BaseModel
	TicketID  string `gorm:"type:uuid;not null;uniqueIndex:idx_retouch_deliverable"`
	AssetID   string `gorm:"type:uuid;not null;uniqueIndex:idx_retouch_deliverable"`
	CreatedBy string `gorm:"type:uuid;not null"`
	VersionNo int    `gorm:"not null;default:1"`
}

type RetouchEvent struct {
	BaseModel
	TicketID   string         `gorm:"type:uuid;not null"`
	FromStatus string         `gorm:"size:40"`
	ToStatus   string         `gorm:"size:40;not null"`
	Action     string         `gorm:"size:80;not null"`
	ActorRealm string         `gorm:"size:32;not null"`
	ActorID    string         `gorm:"type:uuid;not null"`
	Summary    string         `gorm:"size:500"`
	Metadata   datatypes.JSON `gorm:"type:jsonb"`
}

type IdempotencyRecord struct {
	BaseModel
	PrincipalRealm string         `gorm:"size:32;not null;uniqueIndex:idx_idempotency_scope"`
	PrincipalID    string         `gorm:"type:uuid;not null;uniqueIndex:idx_idempotency_scope"`
	Method         string         `gorm:"size:16;not null;uniqueIndex:idx_idempotency_scope"`
	Path           string         `gorm:"size:500;not null;uniqueIndex:idx_idempotency_scope"`
	Key            string         `gorm:"size:128;not null;uniqueIndex:idx_idempotency_scope"`
	RequestDigest  string         `gorm:"size:64;not null"`
	ResourceType   string         `gorm:"size:40"`
	ResourceID     *string        `gorm:"type:uuid"`
	HTTPStatus     int            `gorm:"not null"`
	ResponseCode   int            `gorm:"not null"`
	ResponseData   datatypes.JSON `gorm:"type:jsonb"`
	ExpiresAt      time.Time      `gorm:"not null;index"`
}

type AuditLog struct {
	ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminID       *string        `gorm:"type:uuid;index"`
	AdminEmail    string         `gorm:"size:320;index"`
	AdminRole     string         `gorm:"size:32;index"`
	Action        string         `gorm:"size:100;not null;index"`
	ResourceType  string         `gorm:"size:40;not null;index"`
	ResourceID    *string        `gorm:"type:uuid;index"`
	BeforeSummary datatypes.JSON `gorm:"type:jsonb"`
	AfterSummary  datatypes.JSON `gorm:"type:jsonb"`
	Reason        string         `gorm:"size:500"`
	Result        string         `gorm:"size:32;not null;index"`
	RequestID     string         `gorm:"size:128;index"`
	IPAddressHash string         `gorm:"size:64"`
	UserAgentHash string         `gorm:"size:64"`
	CreatedAt     time.Time      `gorm:"not null;index"`
}

func AllModels() []any {
	return []any{
		&User{}, &UserAIProcessingNotice{}, &AdminAccount{}, &UserSession{}, &AdminSession{},
		&CreditAccount{}, &CreditReservation{}, &CreditLedgerEntry{},
		&RedemptionBatch{}, &RedemptionCode{}, &RedemptionClaim{},
		&Asset{}, &AssetRelation{}, &PromptVersion{},
		&AIProvider{}, &AIModel{}, &PlatformModelBinding{}, &ModelTestRun{},
		&GenerationTask{}, &GenerationTaskAsset{}, &GenerationOutput{},
		&GenerationJob{}, &ProviderAttempt{}, &OutboxEvent{},
		&RetouchTicket{}, &RetouchQuote{}, &RetouchRevision{},
		&RetouchSelectedResult{}, &RetouchDeliverable{}, &RetouchEvent{},
		&IdempotencyRecord{}, &AuditLog{},
	}
}
