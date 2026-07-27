package extends

import (
	"github.com/Wei-Shaw/sub2api/extends/lifecycle"
	"github.com/Wei-Shaw/sub2api/extends/oauthsession"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

var ProviderSet = wire.NewSet(
	lifecycle.NewManager,
	ProvideClaudeOAuthSessionStore,
	ProvideOpenAIOAuthSessionStore,
	ProvideAntigravityOAuthSessionStore,
	ProvideGeminiOAuthSessionStore,
	ProvideGrokOAuthSessionStore,
)

func ProvideClaudeOAuthSessionStore(client *redis.Client) (service.ClaudeOAuthSessionStore, error) {
	return oauthsession.New[oauth.OAuthSession](client, "claude", oauth.SessionTTL)
}

func ProvideOpenAIOAuthSessionStore(client *redis.Client) (service.OpenAIOAuthSessionStore, error) {
	return oauthsession.New[openai.OAuthSession](client, "openai", openai.SessionTTL)
}

func ProvideAntigravityOAuthSessionStore(client *redis.Client) (service.AntigravityOAuthSessionStore, error) {
	return oauthsession.New[antigravity.OAuthSession](client, "antigravity", antigravity.SessionTTL)
}

func ProvideGeminiOAuthSessionStore(client *redis.Client) (service.GeminiOAuthSessionStore, error) {
	return oauthsession.New[geminicli.OAuthSession](client, "gemini", geminicli.SessionTTL)
}

func ProvideGrokOAuthSessionStore(client *redis.Client) (service.GrokOAuthSessionStore, error) {
	return oauthsession.New[xai.OAuthSession](client, "grok", xai.SessionTTL)
}
