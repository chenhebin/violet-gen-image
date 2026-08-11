package retouch

import (
	"time"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/generation"
)

type CreateInput struct {
	SelectedResultIDs    []string `json:"selectedResultIds"`
	Requirement          string   `json:"requirement"`
	SupplementalAssetIDs []string `json:"supplementalAssetIds"`
}

type QuoteInput struct {
	Credits int    `json:"credits"`
	Note    string `json:"note"`
}

type DeliveryRequest struct {
	FileDigest string `json:"fileDigest"`
	Note       string `json:"note"`
}

type TimelineEntry struct {
	Status    string    `json:"status"`
	Action    string    `json:"action,omitempty"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type QuoteDTO struct {
	ID        string    `json:"id"`
	Credits   int       `json:"credits"`
	CreatedAt time.Time `json:"createdAt"`
}

type RevisionDTO struct {
	Message     string    `json:"message"`
	RequestedAt time.Time `json:"requestedAt"`
}

type TicketDTO struct {
	ID                 string                 `json:"id"`
	TicketNo           string                 `json:"ticketNo"`
	TaskID             string                 `json:"taskId"`
	TaskTitle          string                 `json:"taskTitle"`
	Status             string                 `json:"status"`
	SelectedResults    []generation.ResultDTO `json:"selectedResults"`
	Requirement        string                 `json:"requirement"`
	SupplementalAssets []asset.DTO            `json:"supplementalAssets"`
	Quote              *QuoteDTO              `json:"quote,omitempty"`
	Timeline           []TimelineEntry        `json:"timeline"`
	ReservedCredits    int                    `json:"reservedCredits"`
	SpentCredits       int                    `json:"spentCredits"`
	RefundedCredits    int                    `json:"refundedCredits"`
	Revision           *RevisionDTO           `json:"revision,omitempty"`
	Deliverables       []generation.ResultDTO `json:"deliverables"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
}

type SummaryDTO struct {
	ID           string    `json:"id"`
	TicketNo     string    `json:"ticketNo"`
	TaskID       string    `json:"taskId"`
	TaskTitle    string    `json:"taskTitle"`
	Status       string    `json:"status"`
	QuoteCredits *int      `json:"quoteCredits,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type UserSummary struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

type ManageSummaryDTO struct {
	SummaryDTO
	User UserSummary `json:"user"`
}

type ManageTicketDTO struct {
	TicketDTO
	User             UserSummary         `json:"user"`
	SourceTaskDetail ManageSourceTaskDTO `json:"sourceTaskDetail"`
}

type ManageSourceTaskDTO struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Mode              string `json:"mode"`
	Status            string `json:"status"`
	ModelName         string `json:"modelName"`
	SourceRequirement string `json:"sourceRequirement"`
}
