package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/apierror"
	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/prompt"
	"yingyan.local/backend/internal/redemption"
	"yingyan.local/backend/internal/respond"
	"yingyan.local/backend/internal/retouch"
	"yingyan.local/backend/internal/user"
)

type UserHandler struct {
	users       *user.Service
	redemptions *redemption.Service
	assets      *asset.Service
	prompts     *prompt.Service
	generations *generation.Service
	retouches   *retouch.Service
}

func NewUserHandler(
	users *user.Service,
	redemptions *redemption.Service,
	assets *asset.Service,
	prompts *prompt.Service,
	generations *generation.Service,
	retouches *retouch.Service,
) *UserHandler {
	return &UserHandler{
		users: users, redemptions: redemptions, assets: assets,
		prompts: prompts, generations: generations, retouches: retouches,
	}
}

func (h *UserHandler) Me(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	respond.OK(c, principal.User)
}

func (h *UserHandler) Entitlement(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.users.Entitlement(c.Request.Context(), principal.User.ID)
	write(c, value, err)
}

func (h *UserHandler) Ledger(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.users.Ledger(c.Request.Context(), principal.User.ID)
	write(c, value, err)
}

func (h *UserHandler) Quote(c *gin.Context) {
	var input struct {
		Action      string `json:"action"`
		OutputCount int    `json:"outputCount"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Action != "generate" ||
		input.OutputCount < 1 || input.OutputCount > 4 {
		respond.Error(c, apierror.Invalid("报价参数无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.users.Quote(c.Request.Context(), principal.User.ID, input.OutputCount)
	write(c, value, err)
}

func (h *UserHandler) ClaimRedemption(c *gin.Context) {
	var input struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("请输入兑换码", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.redemptions.Claim(
		principal.User.ID,
		input.Code,
		idempotencyKey(c),
	)
	write(c, value, err)
}

func (h *UserHandler) UploadAsset(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	header, err := c.FormFile("file")
	if err != nil {
		respond.Error(c, apierror.Invalid("请选择图片文件", nil))
		return
	}
	value, err := h.assets.UploadUser(
		c.Request.Context(),
		principal.User.ID,
		header,
		c.PostForm("kind"),
		c.PostForm("role"),
	)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.Created(c, value)
}

func (h *UserHandler) DeleteAsset(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	err := h.assets.DeleteOwned(c.Request.Context(), principal.User.ID, c.Param("assetId"))
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.NoData(c)
}

func (h *UserHandler) OptimizePrompt(c *gin.Context) {
	var input prompt.OptimizeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("提示词优化参数无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.prompts.Optimize(c.Request.Context(), principal.User.ID, input)
	write(c, value, err)
}

func (h *UserHandler) ConfirmPrompt(c *gin.Context) {
	var input prompt.ConfirmInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("提示词确认参数无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.prompts.Confirm(c.Request.Context(), principal.User.ID, input)
	write(c, value, err)
}

func (h *UserHandler) CreateGeneration(c *gin.Context) {
	var input generation.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("生成任务参数无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.generations.Create(
		c.Request.Context(),
		principal.User.ID,
		input,
		idempotencyKey(c),
	)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.Success(c, http.StatusAccepted, value)
}

func (h *UserHandler) ListTasks(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.generations.List(c.Request.Context(), principal.User.ID)
	write(c, value, err)
}

func (h *UserHandler) GetTask(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.generations.Get(c.Request.Context(), principal.User.ID, c.Param("taskId"))
	write(c, value, err)
}

func (h *UserHandler) CancelTask(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.generations.Cancel(c.Request.Context(), principal.User.ID, c.Param("taskId"))
	write(c, value, err)
}

func (h *UserHandler) ListRetouch(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.retouches.ListUser(c.Request.Context(), principal.User.ID)
	write(c, value, err)
}

func (h *UserHandler) GetRetouch(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.retouches.GetUser(c.Request.Context(), principal.User.ID, c.Param("ticketId"))
	write(c, value, err)
}

func (h *UserHandler) CreateRetouch(c *gin.Context) {
	var input retouch.CreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("人工修图需求参数无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.retouches.Create(
		c.Request.Context(), principal.User.ID, c.Param("taskId"), input, idempotencyKey(c),
	)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.Created(c, value)
}

func (h *UserHandler) AcceptRetouchQuote(c *gin.Context) {
	var input struct {
		QuoteID string `json:"quoteId"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.QuoteID) == "" {
		respond.Error(c, apierror.Invalid("报价 ID 无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	ticket, balance, err := h.retouches.AcceptQuote(
		c.Request.Context(), principal.User.ID, c.Param("ticketId"),
		input.QuoteID, idempotencyKey(c),
	)
	h.writeRetouchBalance(c, ticket, balance, err)
}

func (h *UserHandler) CancelRetouch(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	ticket, balance, err := h.retouches.Cancel(
		c.Request.Context(), principal.User.ID, c.Param("ticketId"), idempotencyKey(c),
	)
	h.writeRetouchBalance(c, ticket, balance, err)
}

func (h *UserHandler) ReviseRetouch(c *gin.Context) {
	var input struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.Error(c, apierror.Invalid("返修说明无效", nil))
		return
	}
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.retouches.RequestRevision(
		c.Request.Context(), principal.User.ID, c.Param("ticketId"),
		input.Message, idempotencyKey(c),
	)
	write(c, value, err)
}

func (h *UserHandler) ConfirmRetouch(c *gin.Context) {
	principal, _ := auth.UserPrincipalFrom(c)
	value, err := h.retouches.Confirm(
		c.Request.Context(), principal.User.ID, c.Param("ticketId"), idempotencyKey(c),
	)
	write(c, value, err)
}

func (h *UserHandler) writeRetouchBalance(c *gin.Context, ticket *retouch.TicketDTO, balance int, err error) {
	if err != nil {
		respond.Error(c, err)
		return
	}
	status := "empty"
	if balance > 0 {
		status = "active"
	}
	respond.OK(c, map[string]any{
		"ticket": ticket,
		"entitlement": map[string]any{
			"balance": balance, "canCreate": balance > 0, "status": status,
		},
	})
}

func idempotencyKey(c *gin.Context) string {
	return strings.TrimSpace(c.GetHeader("Idempotency-Key"))
}

func write(c *gin.Context, value any, err error) {
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, value)
}
