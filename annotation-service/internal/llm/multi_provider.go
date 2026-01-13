package llm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"annotation-service/internal/gemini"
	"annotation-service/internal/groq"
	"annotation-service/internal/openrouter"
	"annotation-service/internal/models"

	"go.uber.org/zap"
)

// ProviderType represents the type of LLM provider
type ProviderType string

const (
	ProviderGemini     ProviderType = "gemini"
	ProviderGroq       ProviderType = "groq"
	ProviderOpenRouter ProviderType = "openrouter"
)

// ProviderConfig holds configuration for a single provider instance
type ProviderConfig struct {
	Type              ProviderType  `yaml:"type"`
	APIKey            string        `yaml:"api_key"`
	ModelName         string        `yaml:"model_name"`
	MaxRetries        int           `yaml:"max_retries"`
	RetryDelay        time.Duration `yaml:"retry_delay"`
	// Rate limiting per provider
	RequestsPerMinute int           `yaml:"requests_per_minute"`
}

// Provider interface for any LLM provider
type Provider interface {
	Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error)
	Close() error
	GetModelInfo() map[string]interface{}
}

// RateLimitedProvider wraps a provider with rate limiting
type RateLimitedProvider struct {
	provider Provider
	limiter  *RateLimiter
	logger   *zap.Logger
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu               sync.Mutex
	tokens           int
	maxTokens        int
	refillRate       time.Duration
	lastRefill       time.Time
	requestsThisMin  int
	minuteResetTime  time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		tokens:          requestsPerMinute,
		maxTokens:       requestsPerMinute,
		refillRate:      time.Minute / time.Duration(requestsPerMinute),
		lastRefill:      time.Now(),
		minuteResetTime: time.Now().Add(time.Minute),
	}
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Reset counter every minute
	now := time.Now()
	if now.After(rl.minuteResetTime) {
		rl.requestsThisMin = 0
		rl.minuteResetTime = now.Add(time.Minute)
		rl.tokens = rl.maxTokens
		rl.lastRefill = now
	}

	// Refill tokens based on time passed
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)
	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}

	// If no tokens available, wait
	if rl.tokens <= 0 {
		waitTime := rl.refillRate
		rl.mu.Unlock()

		select {
		case <-time.After(waitTime):
			rl.mu.Lock()
			rl.tokens = 1
			rl.lastRefill = time.Now()
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Consume a token
	rl.tokens--
	rl.requestsThisMin++

	return nil
}

// NewRateLimitedProvider wraps a provider with rate limiting
func NewRateLimitedProvider(provider Provider, requestsPerMinute int, logger *zap.Logger) *RateLimitedProvider {
	return &RateLimitedProvider{
		provider: provider,
		limiter:  NewRateLimiter(requestsPerMinute),
		logger:   logger,
	}
}

func (p *RateLimitedProvider) Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error) {
	// Wait for rate limit
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait cancelled: %w", err)
	}

	return p.provider.Annotate(ctx, text)
}

func (p *RateLimitedProvider) Close() error {
	return p.provider.Close()
}

func (p *RateLimitedProvider) GetModelInfo() map[string]interface{} {
	return p.provider.GetModelInfo()
}

// MultiProviderClient manages multiple LLM providers with fallback
type MultiProviderClient struct {
	providers      []*RateLimitedProvider
	currentIndex   int
	mu             sync.RWMutex
	logger         *zap.Logger
	failureCount   map[int]int
	maxFailures    int
}

// MultiProviderConfig holds configuration for multiple providers
type MultiProviderConfig struct {
	Providers   []ProviderConfig
	MaxFailures int // Max consecutive failures before switching provider
}

// NewMultiProviderClient creates a new multi-provider client
func NewMultiProviderClient(cfg MultiProviderConfig, logger *zap.Logger) (*MultiProviderClient, error) {
	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("at least one provider is required")
	}

	if cfg.MaxFailures == 0 {
		cfg.MaxFailures = 3
	}

	providers := make([]*RateLimitedProvider, 0, len(cfg.Providers))

	for i, providerCfg := range cfg.Providers {
		var provider Provider
		var err error

		switch providerCfg.Type {
		case ProviderGemini:
			provider, err = gemini.NewClient(gemini.Config{
				APIKey:     providerCfg.APIKey,
				ModelName:  providerCfg.ModelName,
				MaxRetries: providerCfg.MaxRetries,
				RetryDelay: providerCfg.RetryDelay,
			}, logger)
		case ProviderGroq:
			provider, err = groq.NewClient(groq.Config{
				APIKey:     providerCfg.APIKey,
				ModelName:  providerCfg.ModelName,
				MaxRetries: providerCfg.MaxRetries,
				RetryDelay: providerCfg.RetryDelay,
			}, logger)
		case ProviderOpenRouter:
			provider, err = openrouter.NewClient(openrouter.Config{
				APIKey:     providerCfg.APIKey,
				ModelName:  providerCfg.ModelName,
				MaxRetries: providerCfg.MaxRetries,
				RetryDelay: providerCfg.RetryDelay,
			}, logger)
		default:
			logger.Warn("Unknown provider type, skipping",
				zap.String("type", string(providerCfg.Type)),
				zap.Int("index", i))
			continue
		}

		if err != nil {
			logger.Error("Failed to create provider",
				zap.String("type", string(providerCfg.Type)),
				zap.Int("index", i),
				zap.Error(err))
			continue
		}

		// Set default rate limit if not specified
		rateLimit := providerCfg.RequestsPerMinute
		if rateLimit == 0 {
			rateLimit = 8 // Conservative default for free tier
		}

		rateLimitedProvider := NewRateLimitedProvider(provider, rateLimit, logger)
		providers = append(providers, rateLimitedProvider)

		logger.Info("Provider initialized",
			zap.String("type", string(providerCfg.Type)),
			zap.String("model", providerCfg.ModelName),
			zap.Int("rate_limit", rateLimit),
			zap.Int("index", i))
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers could be initialized")
	}

	return &MultiProviderClient{
		providers:    providers,
		currentIndex: 0,
		logger:       logger,
		failureCount: make(map[int]int),
		maxFailures:  cfg.MaxFailures,
	}, nil
}

// getCurrentProvider returns the current provider and its index
func (c *MultiProviderClient) getCurrentProvider() (*RateLimitedProvider, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.providers[c.currentIndex], c.currentIndex
}

// switchToNextProvider switches to the next available provider
func (c *MultiProviderClient) switchToNextProvider() {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldIndex := c.currentIndex
	c.currentIndex = (c.currentIndex + 1) % len(c.providers)

	c.logger.Info("Switching provider",
		zap.Int("from_index", oldIndex),
		zap.Int("to_index", c.currentIndex),
		zap.Int("total_providers", len(c.providers)))
}

// recordFailure records a failure for a provider
func (c *MultiProviderClient) recordFailure(providerIndex int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount[providerIndex]++

	if c.failureCount[providerIndex] >= c.maxFailures {
		c.logger.Warn("Provider reached max failures",
			zap.Int("provider_index", providerIndex),
			zap.Int("failures", c.failureCount[providerIndex]))
		return true // Should switch
	}

	return false
}

// resetFailureCount resets failure count for a provider
func (c *MultiProviderClient) resetFailureCount(providerIndex int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failureCount[providerIndex] = 0
}

// Annotate tries to annotate using current provider, falls back to next on failure
func (c *MultiProviderClient) Annotate(ctx context.Context, text string) (*models.AnnotationResponse, error) {
	// Try all providers
	for attempts := 0; attempts < len(c.providers); attempts++ {
		provider, providerIndex := c.getCurrentProvider()

		c.logger.Debug("Attempting annotation",
			zap.Int("provider_index", providerIndex),
			zap.Int("attempt", attempts+1))

		result, err := provider.Annotate(ctx, text)

		if err == nil {
			// Success! Reset failure count
			c.resetFailureCount(providerIndex)
			return result, nil
		}

		// Record failure
		c.logger.Error("Provider failed",
			zap.Int("provider_index", providerIndex),
			zap.Error(err))

		shouldSwitch := c.recordFailure(providerIndex)

		// If reached max failures or rate limit error, switch immediately
		if shouldSwitch || isRateLimitError(err) {
			c.switchToNextProvider()
		}
	}

	return nil, fmt.Errorf("all providers failed")
}

// isRateLimitError checks if error is a rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "429") ||
	       contains(errStr, "quota") ||
	       contains(errStr, "rate limit")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
	       (s == substr || len(s) > len(substr) &&
	        (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
	         findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close closes all providers
func (c *MultiProviderClient) Close() error {
	var lastErr error
	for i, provider := range c.providers {
		if err := provider.Close(); err != nil {
			c.logger.Error("Failed to close provider",
				zap.Int("index", i),
				zap.Error(err))
			lastErr = err
		}
	}
	return lastErr
}

// GetModelInfo returns information about the current provider (for LLMClient interface)
func (c *MultiProviderClient) GetModelInfo() map[string]interface{} {
	provider, index := c.getCurrentProvider()
	info := provider.GetModelInfo()
	info["is_current"] = true
	info["provider_index"] = index
	info["total_providers"] = len(c.providers)
	info["failure_count"] = c.failureCount[index]
	return info
}

// GetProvidersInfo returns information about all providers
func (c *MultiProviderClient) GetProvidersInfo() []map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := make([]map[string]interface{}, len(c.providers))
	for i, provider := range c.providers {
		providerInfo := provider.GetModelInfo()
		providerInfo["is_current"] = (i == c.currentIndex)
		providerInfo["failure_count"] = c.failureCount[i]
		info[i] = providerInfo
	}
	return info
}
