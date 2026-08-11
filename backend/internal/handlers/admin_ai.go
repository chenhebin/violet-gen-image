package handlers

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"yingyan.local/backend/internal/aiconfig"
	"yingyan.local/backend/internal/auth"
	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/respond"
)

func (h *AdminHandler) ListProviders(c *gin.Context) {
	values, err := h.ai.ListProviders(c.Request.Context())
	if err != nil {
		respond.Error(c, err)
		return
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, providerDTO(value))
	}
	respond.OK(c, result)
}

func (h *AdminHandler) CreateProvider(c *gin.Context) {
	var input aiconfig.CreateProviderInput
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.ai.CreateProvider(c.Request.Context(), input)
	resourceID := ""
	if value != nil {
		resourceID = value.ID
	}
	h.audit(c, "ai_provider.create", "ai_provider", resourceID, input.Note, nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.Created(c, providerDTO(*value))
}

func (h *AdminHandler) UpdateProvider(c *gin.Context) {
	var input aiconfig.UpdateProviderInput
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.ai.UpdateProvider(c.Request.Context(), c.Param("providerId"), input)
	h.audit(c, "ai_provider.update", "ai_provider", c.Param("providerId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, providerDTO(*value))
}

func (h *AdminHandler) DeleteProvider(c *gin.Context) {
	providerID := c.Param("providerId")
	err := h.ai.DeleteProvider(c.Request.Context(), providerID)
	h.audit(c, "ai_provider.delete", "ai_provider", providerID, "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.NoData(c)
}

func (h *AdminHandler) TestProvider(c *gin.Context) {
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.ai.TestProvider(c.Request.Context(), c.Param("providerId"), principal.Admin.ID)
	h.audit(c, "ai_provider.test", "ai_provider", c.Param("providerId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, providerDTO(*value))
}

func (h *AdminHandler) RotateProviderKey(c *gin.Context) {
	var input struct {
		APIKey string `json:"apiKey"`
	}
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.ai.RotateKey(c.Request.Context(), c.Param("providerId"), input.APIKey)
	h.audit(c, "ai_provider.rotate_key", "ai_provider", c.Param("providerId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, providerDTO(*value))
}

func (h *AdminHandler) ListModels(c *gin.Context) {
	values, err := h.ai.ListModels(c.Request.Context(), c.Query("providerId"))
	if err != nil {
		respond.Error(c, err)
		return
	}
	providers, err := h.ai.ListProviders(c.Request.Context())
	if err != nil {
		respond.Error(c, err)
		return
	}
	providerNames := map[string]string{}
	for _, value := range providers {
		providerNames[value.ID] = value.Name
	}
	bindings, err := h.ai.Bindings(c.Request.Context())
	if err != nil {
		respond.Error(c, err)
		return
	}
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		result = append(result, modelDTO(value, providerNames[value.ProviderID], bindings))
	}
	respond.OK(c, result)
}

func (h *AdminHandler) CreateModel(c *gin.Context) {
	var input aiconfig.CreateModelInput
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.ai.CreateModel(c.Request.Context(), input)
	resourceID := ""
	if value != nil {
		resourceID = value.ID
	}
	h.audit(c, "ai_model.create", "ai_model", resourceID, "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	providerName := h.providerName(c, value.ProviderID)
	bindings, _ := h.ai.Bindings(c.Request.Context())
	respond.Created(c, modelDTO(*value, providerName, bindings))
}

func (h *AdminHandler) UpdateModel(c *gin.Context) {
	var input aiconfig.UpdateModelInput
	if !bindJSON(c, &input) {
		return
	}
	value, err := h.ai.UpdateModel(c.Request.Context(), c.Param("modelId"), input)
	h.audit(c, "ai_model.update", "ai_model", c.Param("modelId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	providerName := h.providerName(c, value.ProviderID)
	bindings, _ := h.ai.Bindings(c.Request.Context())
	respond.OK(c, modelDTO(*value, providerName, bindings))
}

func (h *AdminHandler) DeleteModel(c *gin.Context) {
	modelID := c.Param("modelId")
	err := h.ai.DeleteModel(c.Request.Context(), modelID)
	h.audit(c, "ai_model.delete", "ai_model", modelID, "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.NoData(c)
}

func (h *AdminHandler) TestModel(c *gin.Context) {
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.ai.TestModel(c.Request.Context(), c.Param("modelId"), principal.Admin.ID)
	h.audit(c, "ai_model.test", "ai_model", c.Param("modelId"), "", nil, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	providerName := h.providerName(c, value.ProviderID)
	bindings, _ := h.ai.Bindings(c.Request.Context())
	respond.OK(c, modelDTO(*value, providerName, bindings))
}

func (h *AdminHandler) GetBindings(c *gin.Context) {
	value, err := h.ai.Bindings(c.Request.Context())
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, bindingsDTO(value))
}

func (h *AdminHandler) BindModel(c *gin.Context) {
	var input aiconfig.BindInput
	if !bindJSON(c, &input) {
		return
	}
	principal, _ := auth.AdminPrincipalFrom(c)
	value, err := h.ai.Bind(
		c.Request.Context(), principal.Admin.ID, input, idempotencyKey(c),
	)
	h.audit(c, "platform_model.bind", "platform_model_binding", input.Type, "", map[string]any{"modelId": input.ModelID}, err)
	if err != nil {
		respond.Error(c, err)
		return
	}
	respond.OK(c, bindingsDTO(value))
}

func (h *AdminHandler) providerName(c *gin.Context, providerID string) string {
	values, err := h.ai.ListProviders(c.Request.Context())
	if err != nil {
		return ""
	}
	for _, value := range values {
		if value.ID == providerID {
			return value.Name
		}
	}
	return ""
}

func providerDTO(value model.AIProvider) map[string]any {
	var lastTest any
	if value.LastTestedAt != nil {
		lastTest = map[string]any{
			"testedAt": value.LastTestedAt, "success": value.ConnectionStatus == aiconfig.StatusHealthy,
			"message": value.LastTestSummary,
		}
	}
	return map[string]any{
		"id": value.ID, "name": value.Name, "code": value.Code,
		"protocol": value.Protocol, "baseUrl": value.BaseURL,
		"maskedApiKey": value.APIKeyMask, "enabled": value.Enabled,
		"connectionStatus": value.ConnectionStatus, "lastTest": lastTest,
		"note": value.Notes, "createdAt": value.CreatedAt, "updatedAt": value.UpdatedAt,
	}
}

func modelDTO(value model.AIModel, providerName string, bindings map[string]*string) map[string]any {
	capabilities := map[string]bool{}
	_ = json.Unmarshal(value.Capabilities, &capabilities)
	var lastTest any
	if value.LastTestedAt != nil {
		lastTest = map[string]any{
			"testedAt": value.LastTestedAt,
			"success":  value.TestStatus == aiconfig.StatusHealthy,
			"message":  value.LastTestSummary,
		}
	}
	isPlatform := false
	for _, modelID := range bindings {
		if modelID != nil && *modelID == value.ID {
			isPlatform = true
		}
	}
	return map[string]any{
		"id": value.ID, "providerId": value.ProviderID, "providerName": providerName,
		"displayName": value.DisplayName, "modelId": value.ModelID, "type": value.Type,
		"enabled": value.Enabled, "connectionStatus": value.TestStatus,
		"capabilities": capabilities, "lastTestAt": value.LastTestedAt, "lastTest": lastTest,
		"createdAt": value.CreatedAt, "updatedAt": value.UpdatedAt,
		"isPlatformModel": isPlatform,
	}
}

func bindingsDTO(value map[string]*string) map[string]*string {
	return map[string]*string{
		"chatModelId":  value["chat"],
		"imageModelId": value["image"],
	}
}

var _ = strings.TrimSpace
