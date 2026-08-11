package user

import (
	"context"

	"gorm.io/gorm"

	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/model"
)

type Service struct {
	db      *gorm.DB
	credits *credit.Service
}

type Entitlement struct {
	Balance   int    `json:"balance"`
	CanCreate bool   `json:"canCreate"`
	Status    string `json:"status"`
}

type LedgerDTO struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Amount       int    `json:"amount"`
	BalanceAfter int    `json:"balanceAfter"`
	Description  string `json:"description"`
	CreatedAt    string `json:"createdAt"`
}

type Quote struct {
	Action    string `json:"action"`
	Cost      int    `json:"cost"`
	Balance   int    `json:"balance"`
	CanSubmit bool   `json:"canSubmit"`
}

func New(db *gorm.DB, credits *credit.Service) *Service {
	return &Service{db: db, credits: credits}
}

func (s *Service) Entitlement(ctx context.Context, userID string) (Entitlement, error) {
	var account model.CreditAccount
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&account).Error; err != nil {
		return Entitlement{}, err
	}
	var ledgerCount int64
	if err := s.db.WithContext(ctx).Model(&model.CreditLedgerEntry{}).
		Where("user_id = ?", userID).Count(&ledgerCount).Error; err != nil {
		return Entitlement{}, err
	}
	status := "unredeemed"
	if account.Balance > 0 {
		status = "active"
	} else if ledgerCount > 0 {
		status = "empty"
	}
	return Entitlement{
		Balance: account.Balance, CanCreate: account.Balance > 0, Status: status,
	}, nil
}

func (s *Service) Ledger(ctx context.Context, userID string) ([]LedgerDTO, error) {
	entries, err := s.credits.Ledger(userID, 0)
	if err != nil {
		return nil, err
	}
	result := make([]LedgerDTO, 0, len(entries))
	for _, entry := range entries {
		description := entry.Reason
		if description == "" {
			description = entry.Type
		}
		result = append(result, LedgerDTO{
			ID: entry.ID, Type: entry.Type, Amount: entry.Amount,
			BalanceAfter: entry.BalanceAfter, Description: description,
			CreatedAt: entry.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		})
	}
	return result, nil
}

func (s *Service) Quote(ctx context.Context, userID string, outputCount int) (Quote, error) {
	entitlement, err := s.Entitlement(ctx, userID)
	if err != nil {
		return Quote{}, err
	}
	return Quote{
		Action: "generate", Cost: outputCount, Balance: entitlement.Balance,
		CanSubmit: outputCount > 0 && outputCount <= 4 && entitlement.Balance >= outputCount,
	}, nil
}
