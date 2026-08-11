package generation

import (
	"time"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/prompt"
)

type Settings struct {
	AspectRatio       string `json:"aspectRatio"`
	OutputCount       int    `json:"outputCount"`
	ReferenceStrength int    `json:"referenceStrength"`
}

type CreateInput struct {
	PromptVersionID string                  `json:"promptVersionId,omitempty"`
	Source          string                  `json:"source,omitempty"`
	ReferenceAssets []prompt.ReferenceAsset `json:"referenceAssets,omitempty"`
	AssetIDs        []string                `json:"assetIds"`
	Settings        Settings                `json:"settings"`
}

type ResultDTO struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	DownloadURL string `json:"downloadUrl"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type RetouchSummary struct {
	ID           string    `json:"id"`
	TicketNo     string    `json:"ticketNo"`
	Status       string    `json:"status"`
	UpdatedAt    time.Time `json:"updatedAt"`
	QuoteCredits *int      `json:"quoteCredits,omitempty"`
}

type TaskDTO struct {
	ID              string          `json:"id"`
	Mode            string          `json:"mode"`
	Title           string          `json:"title"`
	Status          string          `json:"status"`
	Prompt          prompt.DTO      `json:"prompt"`
	Settings        Settings        `json:"settings"`
	Assets          []asset.DTO     `json:"assets"`
	RequestedCount  int             `json:"requestedCount"`
	SuccessfulCount int             `json:"successfulCount"`
	ReservedCredits int             `json:"reservedCredits"`
	SpentCredits    int             `json:"spentCredits"`
	RefundedCredits int             `json:"refundedCredits"`
	Progress        int             `json:"progress"`
	Results         []ResultDTO     `json:"results"`
	RetouchTicket   *RetouchSummary `json:"retouchTicket,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}
