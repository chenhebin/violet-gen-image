package manage

import (
	"context"
	"fmt"
	"time"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/redemption"
)

func (s *Service) Dashboard(ctx context.Context, retouchOnly bool) (map[string]any, error) {
	pending, err := s.ListRetouch(ctx, RetouchQuery{Page: 1, PageSize: 8, Pending: true})
	if err != nil {
		return nil, err
	}
	overdue, err := s.ListRetouch(ctx, RetouchQuery{Page: 1, PageSize: 1, SLA: "overdue"})
	if err != nil {
		return nil, err
	}
	dueSoon, err := s.ListRetouch(ctx, RetouchQuery{Page: 1, PageSize: 1, SLA: "due-soon"})
	if err != nil {
		return nil, err
	}
	if retouchOnly {
		return map[string]any{
			"metrics": []map[string]any{
				{"key": "pendingTickets", "label": "待处理工单", "value": pending.Total, "tone": "warning"},
				{"key": "overdueTickets", "label": "已逾期工单", "value": overdue.Total, "tone": dashboardTone(overdue.Total, "danger")},
				{"key": "dueSoonTickets", "label": "即将逾期工单", "value": dueSoon.Total, "tone": dashboardTone(dueSoon.Total, "warning")},
			},
			"currentModels":  map[string]any{},
			"alerts":         []map[string]any{},
			"pendingTickets": pending.Items,
			"recentBatches":  []RedemptionBatchDTO{},
		}, nil
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	var unused, expiring, redeemedToday, creditsToday, failedTasks int64
	if err := s.db.WithContext(ctx).Model(&model.RedemptionCode{}).
		Where("redeemed_at IS NULL AND disabled_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", now).
		Count(&unused).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.RedemptionCode{}).
		Where("redeemed_at IS NULL AND disabled_at IS NULL AND expires_at BETWEEN ? AND ?", now, now.Add(7*24*time.Hour)).
		Count(&expiring).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.RedemptionCode{}).
		Where("redeemed_at >= ?", today).Count(&redeemedToday).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.CreditLedgerEntry{}).
		Where("type = ? AND created_at >= ?", "redemption", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&creditsToday).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&model.GenerationTask{}).
		Where("status = ? AND created_at >= ?", "failed", today).Count(&failedTasks).Error; err != nil {
		return nil, err
	}
	batches, err := s.ListBatches(ctx, BatchQuery{Page: 1, PageSize: 5})
	if err != nil {
		return nil, err
	}
	models := map[string]any{}
	var bindings []model.PlatformModelBinding
	if err := s.db.WithContext(ctx).Find(&bindings).Error; err != nil {
		return nil, err
	}
	for _, binding := range bindings {
		var aiModel model.AIModel
		var providerModel model.AIProvider
		if s.db.WithContext(ctx).First(&aiModel, "id = ?", binding.ModelID).Error != nil ||
			s.db.WithContext(ctx).First(&providerModel, "id = ?", binding.ProviderID).Error != nil {
			continue
		}
		models[binding.BindingType] = map[string]any{
			"id": aiModel.ID, "displayName": aiModel.DisplayName, "modelId": aiModel.ModelID,
			"providerName": providerModel.Name, "type": aiModel.Type,
		}
	}
	alerts := make([]map[string]any, 0)
	var unhealthy []model.AIProvider
	if err := s.db.WithContext(ctx).Where("enabled = ? AND connection_status <> ?", true, "healthy").
		Find(&unhealthy).Error; err != nil {
		return nil, err
	}
	for _, providerModel := range unhealthy {
		alerts = append(alerts, map[string]any{
			"id": providerModel.ID, "type": "provider", "title": providerModel.Name + " 需要检查",
			"description": providerModel.LastTestSummary, "tone": "warning", "href": "/manage/ai-providers",
		})
	}
	if failedTasks > 0 {
		alerts = append(alerts, map[string]any{
			"id": "failed-tasks", "type": "task", "title": "今日存在失败任务",
			"description": fmt.Sprintf("%d 个任务需要检查", failedTasks),
			"tone":        "danger", "href": "/manage/generation-tasks",
		})
	}
	return map[string]any{
		"metrics": []map[string]any{
			{"key": "unusedCodes", "label": "未使用兑换码", "value": unused, "tone": "neutral"},
			{"key": "expiringCodes", "label": "即将过期", "value": expiring, "tone": "warning"},
			{"key": "redeemedToday", "label": "今日兑换", "value": redeemedToday, "tone": "positive"},
			{"key": "creditsGrantedToday", "label": "今日发放次数", "value": creditsToday, "tone": "positive"},
			{"key": "failedTasks", "label": "今日失败任务", "value": failedTasks, "tone": "danger"},
			{"key": "pendingTickets", "label": "待处理工单", "value": pending.Total, "tone": "warning"},
			{"key": "overdueTickets", "label": "已逾期工单", "value": overdue.Total, "tone": dashboardTone(overdue.Total, "danger")},
			{"key": "dueSoonTickets", "label": "即将逾期工单", "value": dueSoon.Total, "tone": dashboardTone(dueSoon.Total, "warning")},
		},
		"currentModels":  models,
		"alerts":         alerts,
		"pendingTickets": pending.Items,
		"recentBatches":  batches.Items,
	}, nil
}

func ExpiringSoon(code model.RedemptionCode) bool {
	now := time.Now().UTC()
	return redemption.Status(code, now) == redemption.StatusUnused &&
		code.ExpiresAt != nil && code.ExpiresAt.Before(now.Add(7*24*time.Hour))
}

func dashboardTone(total int64, alertTone string) string {
	if total > 0 {
		return alertTone
	}
	return "positive"
}
