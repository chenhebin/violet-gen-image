package handlers

import (
	"time"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/manage"
	"yingyan.local/backend/internal/redemption"
	"yingyan.local/backend/internal/respond"
)

func (h *AdminHandler) ListRedemptionCodes(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListCodes(c.Request.Context(), manage.CodeQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c),
		Status: c.Query("status"), BatchID: c.Query("batchId"),
		ProductCode: c.Query("productCode"), RedeemedBy: c.Query("redeemedBy"),
		ExpiringSoon: c.Query("expiringSoon") == "true",
	})
	write(c, value, err)
}

func (h *AdminHandler) GetRedemptionCode(c *gin.Context) {
	value, err := h.manage.GetCode(c.Request.Context(), c.Param("codeId"))
	write(c, value, err)
}

func (h *AdminHandler) ListRedemptionBatches(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListBatches(c.Request.Context(), manage.BatchQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c), ProductCode: c.Query("productCode"),
	})
	write(c, value, err)
}

func (h *AdminHandler) GetRedemptionBatch(c *gin.Context) {
	value, err := h.manage.GetBatch(c.Request.Context(), c.Param("batchId"))
	write(c, value, err)
}

func (h *AdminHandler) UpdateRedemptionBatch(c *gin.Context) {
	var input struct {
		Name string `json:"name"`
	}
	if !bindJSON(c, &input) {
		return
	}
	batchID := c.Param("batchId")
	principal, _ := auth.AdminPrincipalFrom(c)
	value, previousName, err := h.manage.UpdateBatchName(
		c.Request.Context(),
		principal.Admin.ID,
		batchID,
		input.Name,
		idempotencyKey(c),
	)
	afterName := input.Name
	if value != nil {
		afterName = value.Name
	}
	h.auditSnapshots(
		c,
		"redemption_batch.rename",
		"redemption_batch",
		batchID,
		"",
		map[string]any{"name": previousName},
		map[string]any{"name": afterName},
		err,
	)
	write(c, value, err)
}

func (h *AdminHandler) CreateRedemptionBatch(c *gin.Context) {
	var input struct {
		Name           string     `json:"name"`
		Quantity       int        `json:"quantity"`
		CreditsPerCode int        `json:"creditsPerCode"`
		ProductCode    string     `json:"productCode"`
		ExpiresAt      *time.Time `json:"expiresAt"`
		NeverExpires   bool       `json:"neverExpires"`
		Note           string     `json:"note"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	if input.NeverExpires {
		input.ExpiresAt = nil
	}
	value, err := h.redemptions.CreateBatch(c.Request.Context(), redemption.CreateBatchInput{
		Name: input.Name, Quantity: input.Quantity, CreditsPerCode: input.CreditsPerCode,
		ProductCode: input.ProductCode, ExpiresAt: input.ExpiresAt,
		Note: input.Note, CreatedBy: principal.Admin.ID,
	}, idempotencyKey(c))
	h.audit(c, "redemption_batch.create", "redemption_batch", "", input.Note, nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	batch, err := h.manage.GetBatch(c.Request.Context(), value.Batch.ID)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.Created(c, map[string]any{"batch": batch, "codes": value.Codes})
}

func (h *AdminHandler) RevealRedemptionCode(c *gin.Context) {
	codeID := c.Param("codeId")
	full, err := h.redemptions.Reveal(codeID)
	h.audit(c, "redemption_code.reveal", "redemption_code", codeID, "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{"id": codeID, "fullCode": full})
}

func (h *AdminHandler) RevealRedemptionBatch(c *gin.Context) {
	batchID := c.Param("batchId")
	value, err := h.manage.RevealBatch(c.Request.Context(), batchID)
	h.audit(c, "redemption_batch.reveal", "redemption_batch", batchID, "", map[string]any{"count": len(value)}, err)
	write(c, value, err)
}

func (h *AdminHandler) ExportRedemptionBatch(c *gin.Context) {
	batchID := c.Param("batchId")
	format := c.Query("format")
	value, err := h.manage.ExportBatch(c.Request.Context(), batchID, format)
	details := map[string]any{"format": format}
	if value != nil {
		details = map[string]any{"filename": value.Filename, "format": value.Format, "count": value.Count}
	}
	h.audit(c, "redemption_batch.export", "redemption_batch", batchID, "", details, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, value)
}

func (h *AdminHandler) DisableRedemptionCodes(c *gin.Context) {
	var input struct {
		CodeIDs []string `json:"codeIds"`
		BatchID string   `json:"batchId"`
		Reason  string   `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	affected, skipped, err := h.redemptions.Disable(
		c.Request.Context(), input.CodeIDs, input.BatchID, principal.Admin.ID,
		input.Reason, idempotencyKey(c),
	)
	h.audit(c, "redemption_code.disable", "redemption_code", input.BatchID, input.Reason, map[string]any{"affected": affected}, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{"affected": affected, "skipped": skipped, "failed": 0})
}

func (h *AdminHandler) ExtendRedemptionCodes(c *gin.Context) {
	var input struct {
		CodeIDs   []string   `json:"codeIds"`
		BatchID   string     `json:"batchId"`
		ExpiresAt *time.Time `json:"expiresAt"`
		Reason    string     `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	affected, skipped, err := h.redemptions.Extend(
		c.Request.Context(), input.CodeIDs, input.BatchID, input.ExpiresAt,
		principal.Admin.ID, input.Reason, idempotencyKey(c),
	)
	h.audit(c, "redemption_code.extend", "redemption_code", input.BatchID, input.Reason, map[string]any{"affected": affected}, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{"affected": affected, "skipped": skipped, "failed": 0})
}
