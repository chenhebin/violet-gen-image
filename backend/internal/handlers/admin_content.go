package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/manage"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/respond"
	"yingyan.local/backend/internal/retouch"
)

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListUsers(c.Request.Context(), manage.UserQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c), Status: c.Query("status"),
		MinBalance: optionalInt(c, "minBalance"), MaxBalance: optionalInt(c, "maxBalance"),
		HasTasks: optionalBool(c, "hasTasks"), HasTickets: optionalBool(c, "hasTickets"),
	})
	write(c, value, err)
}

func (h *AdminHandler) GetUser(c *gin.Context) {
	value, err := h.manage.GetUser(c.Request.Context(), c.Param("userId"))
	write(c, value, err)
}

func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	var input struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.manage.SetUserStatus(c.Request.Context(), c.Param("userId"), input.Status, input.Reason)
	h.audit(c, "user.status", "user", c.Param("userId"), input.Reason, value, err)
	write(c, value, err)
}

func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	password, expiresAt, err := h.manage.ResetPassword(c.Request.Context(), c.Param("userId"))
	h.audit(c, "user.reset_password", "user", c.Param("userId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{"temporaryPassword": password, "expiresAt": expiresAt})
}

func (h *AdminHandler) AdjustUserCredits(c *gin.Context) {
	var input struct {
		Amount      int    `json:"amount"`
		Reason      string `json:"reason"`
		ReferenceNo string `json:"referenceNo"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	user, ledger, err := h.manage.AdjustCredits(
		c.Request.Context(), c.Param("userId"), principal.Admin.ID,
		input.Amount, input.Reason, input.ReferenceNo, idempotencyKey(c),
	)
	h.audit(c, "user.adjust_credits", "user", c.Param("userId"), input.Reason, map[string]any{"amount": input.Amount}, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{"user": user, "ledger": ledger})
}

func (h *AdminHandler) ListUsageLedger(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListLedger(c.Request.Context(), c.Query("userId"), page, pageSize)
	write(c, value, err)
}

func (h *AdminHandler) ListTasks(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListTasks(c.Request.Context(), manage.TaskQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c), Status: c.Query("status"),
		Mode: c.Query("mode"), UserID: c.Query("userId"), ProviderID: c.Query("providerId"),
		ModelID: c.Query("modelId"), HasRetouchTicket: optionalBool(c, "hasRetouchTicket"),
	})
	write(c, value, err)
}

func (h *AdminHandler) GetTask(c *gin.Context) {
	value, err := h.manage.GetTask(c.Request.Context(), c.Param("taskId"))
	write(c, value, err)
}

func (h *AdminHandler) ListAssets(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListAssets(c.Request.Context(), manage.AssetQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c), Kind: c.Query("kind"),
		UserID: c.Query("userId"), TaskID: c.Query("taskId"), TicketID: c.Query("ticketId"),
		Retained: optionalBool(c, "retained"),
	})
	write(c, value, err)
}

func (h *AdminHandler) GetAsset(c *gin.Context) {
	value, err := h.manage.GetAsset(c.Request.Context(), c.Param("assetId"))
	write(c, value, err)
}

func (h *AdminHandler) SignAsset(c *gin.Context) {
	value, err := h.manage.GetAsset(c.Request.Context(), c.Param("assetId"))
	h.audit(c, "asset.signed_url", "asset", c.Param("assetId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, map[string]any{
		"url": value.PreviewURL, "expiresAt": time.Now().UTC().Add(15 * time.Minute),
	})
}

func (h *AdminHandler) RetainAsset(c *gin.Context) {
	var input struct {
		Retained bool   `json:"retained"`
		Reason   string `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	_, err := h.assets.SetRetained(
		c.Request.Context(), c.Param("assetId"), principal.Admin.ID, input.Retained, input.Reason,
	)
	h.audit(c, "asset.retain", "asset", c.Param("assetId"), input.Reason, map[string]any{"retained": input.Retained}, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	value, err := h.manage.GetAsset(c.Request.Context(), c.Param("assetId"))
	write(c, value, err)
}

func (h *AdminHandler) CleanupAsset(c *gin.Context) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	_, err := h.assets.CleanupManage(c.Request.Context(), c.Param("assetId"), input.Reason)
	h.audit(c, "asset.cleanup", "asset", c.Param("assetId"), input.Reason, nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	value, err := h.manage.GetAsset(c.Request.Context(), c.Param("assetId"))
	write(c, value, err)
}

func (h *AdminHandler) ListAudits(c *gin.Context) {
	page, pageSize := pageQuery(c)
	principal, _ := auth.AdminPrincipalFrom(c)
	operatorID := c.Query("operatorId")
	if principal.Admin.Role == "retouch_operator" {
		operatorID = principal.Admin.ID
	}
	value, err := h.manage.ListAudits(c.Request.Context(), manage.AuditQuery{
		Page: page, PageSize: pageSize, OperatorID: operatorID,
		Search: keywordQuery(c),
		Action: c.Query("action"), ResourceType: c.Query("resourceType"),
		Result: c.Query("result"), StartAt: optionalTime(c, "startAt"), EndAt: optionalTime(c, "endAt"),
	})
	write(c, value, err)
}

func (h *AdminHandler) ListRetouchTickets(c *gin.Context) {
	page, pageSize := pageQuery(c)
	value, err := h.manage.ListRetouch(c.Request.Context(), manage.RetouchQuery{
		Page: page, PageSize: pageSize, Search: keywordQuery(c), Status: c.Query("status"),
	})
	write(c, value, err)
}

func (h *AdminHandler) GetRetouchTicket(c *gin.Context) {
	value, err := h.retouches.GetManage(c.Request.Context(), c.Param("ticketId"))
	write(c, value, err)
}

func (h *AdminHandler) QuoteRetouch(c *gin.Context) {
	var input retouch.QuoteInput
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.retouches.Quote(
		c.Request.Context(), principal.Admin.ID, c.Param("ticketId"), input, idempotencyKey(c),
	)
	h.audit(c, "retouch.quote", "retouch_ticket", c.Param("ticketId"), input.Note, map[string]any{"credits": input.Credits}, err)
	write(c, value, err)
}

func (h *AdminHandler) StartRetouch(c *gin.Context) {
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.retouches.Start(
		c.Request.Context(), principal.Admin.ID, c.Param("ticketId"), idempotencyKey(c),
	)
	h.audit(c, "retouch.start", "retouch_ticket", c.Param("ticketId"), "", nil, err)
	write(c, value, err)
}

func (h *AdminHandler) DeliverRetouch(c *gin.Context) {
	principal, _ := auth.AdminPrincipalFrom(c)
	ticket, err := h.retouches.GetManage(c.Request.Context(), c.Param("ticketId"))
	if err != nil {
		respond.Error(c, err)
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		respond.Error(c, apierror.Invalid("请选择人工成片", nil))
		return
	}
	headers := form.File["files"]
	if len(headers) < 1 || len(headers) > 4 {
		respond.Error(c, apierror.Invalid("请上传 1 到 4 张人工成片", nil))
		return
	}
	note := strings.TrimSpace(c.PostForm("note"))
	digest, err := deliveryDigest(headers)
	if err != nil {
		respond.Error(c, err)
		return
	}
	request := retouch.DeliveryRequest{FileDigest: digest, Note: note}
	if replayed, ok, replayErr := h.retouches.ReplayDelivery(
		c.Request.Context(),
		principal.Admin.ID,
		ticket.ID,
		request,
		idempotencyKey(c),
	); replayErr != nil {
		respond.Error(c, replayErr)
		return
	} else if ok {
		write(c, replayed, nil)
		return
	}
	resultAssets := make([]model.Asset, 0, len(headers))
	for _, header := range headers {
		value, uploadErr := h.assets.UploadAdminResult(
			c.Request.Context(), principal.Admin.ID, ticket.User.ID, header, asset.KindRetouchResult,
		)
		if uploadErr != nil {
			h.cleanupUnlinkedDeliverables(c, resultAssets)
			respond.Error(c, uploadErr)
			return
		}
		resultAssets = append(resultAssets, *value)
	}
	value, err := h.retouches.Deliver(
		c.Request.Context(), principal.Admin.ID, ticket.ID,
		resultAssets, request, idempotencyKey(c),
	)
	if err != nil {
		h.cleanupUnlinkedDeliverables(c, resultAssets)
	} else {
		used := make(map[string]struct{}, len(value.Deliverables))
		for _, deliverable := range value.Deliverables {
			used[deliverable.ID] = struct{}{}
		}
		unused := make([]model.Asset, 0)
		for _, item := range resultAssets {
			if _, ok := used[item.ID]; !ok {
				unused = append(unused, item)
			}
		}
		h.cleanupUnlinkedDeliverables(c, unused)
	}
	h.audit(c, "retouch.deliver", "retouch_ticket", ticket.ID, note, map[string]any{"count": len(resultAssets)}, err)
	write(c, value, err)
}

func (h *AdminHandler) cleanupUnlinkedDeliverables(c *gin.Context, values []model.Asset) {
	for _, value := range values {
		_, _ = h.assets.CleanupManage(
			context.WithoutCancel(c.Request.Context()),
			value.ID,
			"人工成片上传未被工单采用，自动清理",
		)
	}
}

func deliveryDigest(headers []*multipart.FileHeader) (string, error) {
	digest := sha256.New()
	for _, header := range headers {
		if header.Size > 30<<20 {
			return "", apierror.Invalid("人工成片不能超过 30MB", nil)
		}
		_, _ = io.WriteString(
			digest,
			fmt.Sprintf(
				"%d\x00%s\x00%s\x00",
				header.Size,
				header.Filename,
				header.Header.Get("Content-Type"),
			),
		)
		file, err := header.Open()
		if err != nil {
			return "", apierror.Invalid("人工成片读取失败", nil)
		}
		written, copyErr := io.Copy(digest, io.LimitReader(file, (30<<20)+1))
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			return "", apierror.Invalid("人工成片读取失败", nil)
		}
		if written > 30<<20 {
			return "", apierror.Invalid("人工成片不能超过 30MB", nil)
		}
		_, _ = io.WriteString(digest, "\x00")
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func (h *AdminHandler) RejectRetouch(c *gin.Context) {
	h.closeRetouch(c, false)
}

func (h *AdminHandler) FailRetouch(c *gin.Context) {
	h.closeRetouch(c, true)
}

func (h *AdminHandler) closeRetouch(c *gin.Context, failed bool) {
	var input struct {
		Reason string `json:"reason"`
	}
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	var value *retouch.ManageTicketDTO
	var err error
	action := "retouch.reject"
	if failed {
		action = "retouch.fail"
		value, err = h.retouches.Fail(
			c.Request.Context(), principal.Admin.ID, c.Param("ticketId"), input.Reason, idempotencyKey(c),
		)
	} else {
		value, err = h.retouches.Reject(
			c.Request.Context(), principal.Admin.ID, c.Param("ticketId"), input.Reason, idempotencyKey(c),
		)
	}
	h.audit(c, action, "retouch_ticket", c.Param("ticketId"), input.Reason, nil, err)
	write(c, value, err)
}

var _ = http.StatusOK
