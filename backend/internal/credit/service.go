package credit

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/model"
)

const (
	LedgerRedemption = "redemption"
	LedgerReserve    = "reserve"
	LedgerRelease    = "release"
	LedgerRefund     = "refund"
	LedgerAdjustment = "adjustment"

	ReservationActive   = "active"
	ReservationSettled  = "settled"
	ReservationReleased = "released"
	ReservationRefunded = "refunded"
)

type Service struct {
	db *gorm.DB
}

type Mutation struct {
	UserID       string
	Type         string
	Amount       int
	BusinessType string
	BusinessID   *string
	OperatorID   *string
	Reason       string
	ReferenceNo  string
}

func New(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) EnsureAccountTx(tx *gorm.DB, userID string) error {
	account := model.CreditAccount{UserID: userID, Balance: 0, Version: 1}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&account).Error
}

func (s *Service) Balance(userID string) (int, error) {
	var account model.CreditAccount
	if err := s.db.Where("user_id = ?", userID).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, apierror.New(
				http.StatusForbidden,
				apierror.CodeEntitlementMissing,
				"请先兑换次数",
				nil,
			)
		}
		return 0, err
	}
	return account.Balance, nil
}

func (s *Service) Ledger(userID string, limit int) ([]model.CreditLedgerEntry, error) {
	if limit < 0 || limit > 100 {
		limit = 100
	}
	var entries []model.CreditLedgerEntry
	query := s.db.
		Where("user_id = ?", userID).
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&entries).Error
	return entries, err
}

func (s *Service) AddTx(tx *gorm.DB, mutation Mutation) (*model.CreditLedgerEntry, error) {
	if mutation.Amount == 0 {
		return nil, apierror.Invalid("次数变动不能为 0", nil)
	}
	account, err := lockAccount(tx, mutation.UserID)
	if err != nil {
		return nil, err
	}

	before := account.Balance
	after := before + mutation.Amount
	if after < 0 {
		return nil, apierror.New(
			http.StatusConflict,
			apierror.CodeInsufficientCredit,
			"剩余次数不足",
			map[string]int{"required": -mutation.Amount, "balance": before},
		)
	}
	if err := tx.Model(account).Updates(map[string]any{
		"balance": after,
		"version": gorm.Expr("version + 1"),
	}).Error; err != nil {
		return nil, err
	}

	entry := model.CreditLedgerEntry{
		UserID:        mutation.UserID,
		Type:          mutation.Type,
		Amount:        mutation.Amount,
		BalanceBefore: before,
		BalanceAfter:  after,
		BusinessType:  mutation.BusinessType,
		BusinessID:    mutation.BusinessID,
		OperatorID:    mutation.OperatorID,
		Reason:        mutation.Reason,
		ReferenceNo:   mutation.ReferenceNo,
	}
	if err := tx.Create(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

func (s *Service) ReserveTx(
	tx *gorm.DB,
	userID string,
	businessType string,
	businessID string,
	amount int,
	reason string,
) (*model.CreditReservation, *model.CreditLedgerEntry, error) {
	if amount <= 0 {
		return nil, nil, apierror.Invalid("预占次数必须大于 0", nil)
	}

	var existing model.CreditReservation
	err := tx.Where("business_type = ? AND business_id = ?", businessType, businessID).
		First(&existing).Error
	if err == nil {
		return nil, nil, apierror.New(
			http.StatusConflict,
			apierror.CodeIdempotencyConflict,
			"该业务已经预占过次数",
			nil,
		)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	businessIDCopy := businessID
	entry, err := s.AddTx(tx, Mutation{
		UserID:       userID,
		Type:         LedgerReserve,
		Amount:       -amount,
		BusinessType: businessType,
		BusinessID:   &businessIDCopy,
		Reason:       reason,
		ReferenceNo:  businessID,
	})
	if err != nil {
		return nil, nil, err
	}

	reservation := model.CreditReservation{
		UserID:       userID,
		BusinessType: businessType,
		BusinessID:   businessID,
		Amount:       amount,
		Status:       ReservationActive,
		Version:      1,
	}
	if err := tx.Create(&reservation).Error; err != nil {
		return nil, nil, err
	}
	return &reservation, entry, nil
}

func (s *Service) SettleTx(tx *gorm.DB, reservationID string, amount int) error {
	reservation, err := lockReservation(tx, reservationID)
	if err != nil {
		return err
	}
	remaining := reservation.Amount - reservation.SettledAmount - reservation.ReleasedAmount
	if amount <= 0 || amount > remaining {
		return apierror.Invalid("结算次数超出可结算范围", map[string]int{
			"remaining": remaining,
			"requested": amount,
		})
	}

	now := time.Now().UTC()
	settled := reservation.SettledAmount + amount
	status := ReservationActive
	if settled+reservation.ReleasedAmount == reservation.Amount {
		status = ReservationSettled
	}
	return tx.Model(reservation).Updates(map[string]any{
		"settled_amount": settled,
		"settled_at":     &now,
		"status":         status,
		"version":        gorm.Expr("version + 1"),
	}).Error
}

func (s *Service) ReleaseTx(
	tx *gorm.DB,
	reservationID string,
	amount int,
	reason string,
) (*model.CreditLedgerEntry, error) {
	reservation, err := lockReservation(tx, reservationID)
	if err != nil {
		return nil, err
	}
	remaining := reservation.Amount - reservation.SettledAmount - reservation.ReleasedAmount
	if amount <= 0 || amount > remaining {
		return nil, apierror.Invalid("释放次数超出可释放范围", map[string]int{
			"remaining": remaining,
			"requested": amount,
		})
	}

	businessID := reservation.BusinessID
	entry, err := s.AddTx(tx, Mutation{
		UserID:       reservation.UserID,
		Type:         LedgerRelease,
		Amount:       amount,
		BusinessType: reservation.BusinessType,
		BusinessID:   &businessID,
		Reason:       reason,
		ReferenceNo:  reservation.BusinessID,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	released := reservation.ReleasedAmount + amount
	status := ReservationActive
	if released+reservation.SettledAmount == reservation.Amount {
		status = ReservationReleased
	}
	err = tx.Model(reservation).Updates(map[string]any{
		"released_amount": released,
		"released_at":     &now,
		"status":          status,
		"version":         gorm.Expr("version + 1"),
	}).Error
	return entry, err
}

func (s *Service) RefundTx(
	tx *gorm.DB,
	reservationID string,
	amount int,
	reason string,
) (*model.CreditLedgerEntry, error) {
	reservation, err := lockReservation(tx, reservationID)
	if err != nil {
		return nil, err
	}
	refundable := reservation.SettledAmount - reservation.RefundedAmount
	if amount <= 0 || amount > refundable {
		return nil, apierror.Invalid("退款次数超出可退款范围", map[string]int{
			"refundable": refundable,
			"requested":  amount,
		})
	}

	businessID := reservation.BusinessID
	entry, err := s.AddTx(tx, Mutation{
		UserID:       reservation.UserID,
		Type:         LedgerRefund,
		Amount:       amount,
		BusinessType: reservation.BusinessType,
		BusinessID:   &businessID,
		Reason:       reason,
		ReferenceNo:  reservation.BusinessID,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	refunded := reservation.RefundedAmount + amount
	status := reservation.Status
	if refunded == reservation.SettledAmount &&
		reservation.SettledAmount+reservation.ReleasedAmount == reservation.Amount {
		status = ReservationRefunded
	}
	err = tx.Model(reservation).Updates(map[string]any{
		"refunded_amount": refunded,
		"refunded_at":     &now,
		"status":          status,
		"version":         gorm.Expr("version + 1"),
	}).Error
	return entry, err
}

func lockAccount(tx *gorm.DB, userID string) (*model.CreditAccount, error) {
	var account model.CreditAccount
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&account).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.New(
			http.StatusForbidden,
			apierror.CodeEntitlementMissing,
			"请先兑换次数",
			nil,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("lock credit account: %w", err)
	}
	return &account, nil
}

func lockReservation(tx *gorm.DB, reservationID string) (*model.CreditReservation, error) {
	var reservation model.CreditReservation
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&reservation, "id = ?", reservationID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apierror.Invalid("次数预占不存在", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("lock credit reservation: %w", err)
	}
	return &reservation, nil
}
