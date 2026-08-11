package ai

import (
	"fmt"
	"time"

	"yingyan.local/backend/internal/model"
	"yingyan.local/backend/internal/platform/security"
	"yingyan.local/backend/internal/provider"
)

type Factory struct {
	encryptionKey      string
	allowHTTP          bool
	allowPrivate       bool
	connectTimeout     time.Duration
	requestTimeout     time.Duration
	responseHeaderTime time.Duration
}

type FactoryConfig struct {
	EncryptionKey         string
	AllowHTTP             bool
	AllowPrivateNetwork   bool
	ConnectTimeout        time.Duration
	RequestTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
}

func NewFactory(config FactoryConfig) *Factory {
	return &Factory{
		encryptionKey:      config.EncryptionKey,
		allowHTTP:          config.AllowHTTP,
		allowPrivate:       config.AllowPrivateNetwork,
		connectTimeout:     config.ConnectTimeout,
		requestTimeout:     config.RequestTimeout,
		responseHeaderTime: config.ResponseHeaderTimeout,
	}
}

func (f *Factory) FromProvider(providerModel model.AIProvider) (provider.Adapter, error) {
	key, err := security.Decrypt(providerModel.APIKeyCiphertext, f.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt provider key: %w", err)
	}
	return provider.NewOpenAICompatible(provider.Config{
		BaseURL:               providerModel.BaseURL,
		APIKey:                string(key),
		AllowHTTP:             f.allowHTTP,
		AllowPrivateNetwork:   f.allowPrivate,
		ConnectTimeout:        f.connectTimeout,
		RequestTimeout:        f.requestTimeout,
		ResponseHeaderTimeout: f.responseHeaderTime,
	})
}

func (f *Factory) FromSnapshot(baseURL string, encryptedKey []byte) (provider.Adapter, error) {
	key, err := security.Decrypt(encryptedKey, f.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt provider snapshot key: %w", err)
	}
	return provider.NewOpenAICompatible(provider.Config{
		BaseURL:               baseURL,
		APIKey:                string(key),
		AllowHTTP:             f.allowHTTP,
		AllowPrivateNetwork:   f.allowPrivate,
		ConnectTimeout:        f.connectTimeout,
		RequestTimeout:        f.requestTimeout,
		ResponseHeaderTimeout: f.responseHeaderTime,
	})
}
