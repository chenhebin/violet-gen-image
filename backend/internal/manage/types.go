package manage

import (
	"time"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/retouch"
)

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
	HasMore  bool  `json:"hasMore"`
}

type RedemptionBatchDTO struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	ProductCode    string         `json:"productCode"`
	Quantity       int            `json:"quantity"`
	CreditsPerCode int            `json:"creditsPerCode"`
	ExpiresAt      *time.Time     `json:"expiresAt"`
	NeverExpires   bool           `json:"neverExpires"`
	Note           string         `json:"note,omitempty"`
	CreatedBy      string         `json:"createdBy"`
	CreatedAt      time.Time      `json:"createdAt"`
	Counts         map[string]int `json:"counts"`
	UsageRate      float64        `json:"usageRate"`
}

type RedemptionCodeDTO struct {
	ID              string     `json:"id"`
	MaskedCode      string     `json:"maskedCode"`
	BatchID         string     `json:"batchId"`
	BatchName       string     `json:"batchName"`
	ProductCode     string     `json:"productCode"`
	Credits         int        `json:"credits"`
	Status          string     `json:"status"`
	ExpiresAt       *time.Time `json:"expiresAt"`
	RedeemedBy      string     `json:"redeemedBy,omitempty"`
	RedeemedByEmail string     `json:"redeemedByEmail,omitempty"`
	RedeemedAt      *time.Time `json:"redeemedAt,omitempty"`
	DisabledBy      string     `json:"disabledBy,omitempty"`
	DisabledAt      *time.Time `json:"disabledAt,omitempty"`
	DisabledReason  string     `json:"disabledReason,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	ExpiringSoon    bool       `json:"expiringSoon"`
}

type ManagedUserDTO struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	Status         string     `json:"status"`
	Balance        int        `json:"balance"`
	TotalRedeemed  int        `json:"totalRedeemed"`
	TotalConsumed  int        `json:"totalConsumed"`
	TaskCount      int64      `json:"taskCount"`
	TicketCount    int64      `json:"ticketCount"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	DisabledReason string     `json:"disabledReason,omitempty"`
}

type LedgerDTO struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	Type          string    `json:"type"`
	Amount        int       `json:"amount"`
	BalanceBefore int       `json:"balanceBefore"`
	BalanceAfter  int       `json:"balanceAfter"`
	Description   string    `json:"description"`
	Reason        string    `json:"reason,omitempty"`
	ReferenceNo   string    `json:"referenceNo,omitempty"`
	OperatorID    *string   `json:"operatorId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

type ManagedAssetDTO struct {
	ID                  string     `json:"id"`
	OwnerID             string     `json:"ownerId"`
	OwnerEmail          string     `json:"ownerEmail"`
	Name                string     `json:"name"`
	Kind                string     `json:"kind"`
	Role                string     `json:"role,omitempty"`
	MIMEType            string     `json:"mimeType"`
	Size                int64      `json:"size"`
	Width               int        `json:"width"`
	Height              int        `json:"height"`
	PreviewURL          string     `json:"previewUrl,omitempty"`
	PreviewURLExpiresAt *time.Time `json:"previewUrlExpiresAt,omitempty"`
	TaskID              string     `json:"taskId,omitempty"`
	TicketID            string     `json:"ticketId,omitempty"`
	Retained            bool       `json:"retained"`
	RetentionExpiresAt  *time.Time `json:"retentionExpiresAt"`
	DeletedAt           *time.Time `json:"deletedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type ManagedTaskSummaryDTO struct {
	ID               string    `json:"id"`
	OwnerID          string    `json:"ownerId"`
	OwnerEmail       string    `json:"ownerEmail"`
	Title            string    `json:"title"`
	Mode             string    `json:"mode"`
	Status           string    `json:"status"`
	Progress         int       `json:"progress"`
	RequestedCount   int       `json:"requestedCount"`
	SuccessfulCount  int       `json:"successfulCount"`
	ReservedCredits  int       `json:"reservedCredits"`
	SpentCredits     int       `json:"spentCredits"`
	RefundedCredits  int       `json:"refundedCredits"`
	ProviderName     string    `json:"providerName"`
	ModelName        string    `json:"modelName"`
	HasRetouchTicket bool      `json:"hasRetouchTicket"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ManagedTaskDTO struct {
	ManagedTaskSummaryDTO
	SourceRequirement string               `json:"sourceRequirement"`
	OptimizedPrompt   prompt.Sections      `json:"optimizedPrompt"`
	ConfirmedPrompt   prompt.Sections      `json:"confirmedPrompt"`
	Settings          map[string]any       `json:"settings"`
	Assets            []ManagedAssetDTO    `json:"assets"`
	Results           []ManagedAssetDTO    `json:"results"`
	ExecutionSnapshot map[string]any       `json:"executionSnapshot"`
	ProviderAttempts  []ProviderAttemptDTO `json:"providerAttempts"`
	ErrorMessage      string               `json:"errorMessage,omitempty"`
	RetouchTicket     *retouch.SummaryDTO  `json:"retouchTicket,omitempty"`
}

type ProviderAttemptDTO struct {
	ID                string         `json:"id"`
	JobID             string         `json:"jobId"`
	AttemptNo         int            `json:"attemptNo"`
	Operation         string         `json:"operation"`
	Method            string         `json:"method"`
	Path              string         `json:"path"`
	Model             string         `json:"model"`
	Status            string         `json:"status"`
	ExternalRequestID string         `json:"externalRequestId,omitempty"`
	ResponseStatus    int            `json:"responseStatus,omitempty"`
	LatencyMillis     int64          `json:"latencyMs"`
	ErrorCode         string         `json:"errorCode,omitempty"`
	ErrorKind         string         `json:"errorKind,omitempty"`
	ErrorSummary      string         `json:"errorSummary,omitempty"`
	RequestSummary    map[string]any `json:"requestSummary,omitempty"`
	ResponseMetadata  map[string]any `json:"responseMetadata,omitempty"`
	StartedAt         time.Time      `json:"startedAt"`
	CompletedAt       *time.Time     `json:"completedAt,omitempty"`
}

type AuditDTO struct {
	ID            string         `json:"id"`
	OperatorID    string         `json:"operatorId"`
	OperatorEmail string         `json:"operatorEmail"`
	OperatorRole  string         `json:"operatorRole"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resourceType"`
	ResourceID    string         `json:"resourceId"`
	Before        map[string]any `json:"before,omitempty"`
	After         map[string]any `json:"after,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	Result        string         `json:"result"`
	RequestID     string         `json:"requestId"`
	IP            string         `json:"ip,omitempty"`
	Device        string         `json:"device,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type SignedAssetDTO struct {
	asset.DTO
	ExpiresAt time.Time `json:"expiresAt"`
}
