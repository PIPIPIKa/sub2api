//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/stretchr/testify/require"
)

func TestForkPatch_CodexDuplicateBareAliasesAndOfficialAPIKeyNormalize(t *testing.T) {
	got := FilterCodexModelIDsForGroup([]string{"gpt-6", "gpt-6-astra", "gpt-5.6", "gpt-5.6-sol"}, nil)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, got)

	// Keep bare alias when canonical model is absent or explicitly selected.
	require.Equal(t, []string{"gpt-6"}, FilterCodexModelIDsForGroup([]string{"gpt-6"}, nil))
	explicitGroup := &Group{ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"gpt-6", "gpt-6-astra"}}}
	require.Equal(t, []string{"gpt-6", "gpt-6-astra"}, FilterCodexModelIDsForGroup([]string{"gpt-6", "gpt-6-astra"}, explicitGroup))

	merged, changed, err := mergeConfiguredCodexModelsManifest(
		[]byte(`{"models":[{"slug":"gpt-6-astra"},{"slug":"gpt-5.6-sol"}]}`),
		[]string{"gpt-6", "gpt-6-astra", "gpt-5.6", "gpt-5.6-sol"},
		nil,
		false,
	)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, codexManifestModelSlugs(t, merged))

	officialKey := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://api.openai.com/v1"}}
	require.Equal(t, "gpt-6-astra", normalizeOpenAIModelForUpstream(officialKey, "gpt-6"))
	require.Equal(t, "gpt-5.6-sol", normalizeOpenAIModelForUpstream(officialKey, "gpt-5.6"))

	customRelay := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://relay.example.com/v1"}}
	require.Equal(t, "gpt-6", normalizeOpenAIModelForUpstream(customRelay, "gpt-6"))
}

func TestForkPatch_ServiceTiersIntersectAndOfficialOAuthDefault(t *testing.T) {
	// Official OAuth without synced snapshot + official API key with ultrafast -> keeps shared Fast ("priority").
	oauthUnsynced := Account{
		ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Credentials: map[string]any{"base_url": "https://chatgpt.com"},
	}
	officialKey := Account{
		ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"base_url": "https://api.openai.com/v1"},
	}
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{"gpt-6-astra"}, []Account{oauthUnsynced, officialKey}, nil, nil, true)
	require.NoError(t, err)
	models := decodeCodexManifestModels(t, body)
	require.Len(t, models, 1)
	rawTiers, err := json.Marshal(models[0]["service_tiers"])
	require.NoError(t, err)
	require.Contains(t, string(rawTiers), `"id":"priority"`)
	require.NotContains(t, string(rawTiers), `"id":"ultrafast"`)
}

func TestForkPatch_AntigravityQuotaPrefersDailyForPaidAccount(t *testing.T) {
	var hitOrder []string
	prod := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitOrder = append(hitOrder, "prod:"+r.URL.Path)
		switch r.URL.Path {
		case "/v1internal:loadCodeAssist":
			_, _ = w.Write([]byte(`{"currentTier":{"id":"free-tier"},"paidTier":{"id":"g1-ultra-lite-tier","name":"Google AI Ultra"}}`))
		case "/v1internal:fetchAvailableModels":
			_, _ = w.Write([]byte(`{"models":{"gemini-3.8-flash-high":{"quotaInfo":{"remainingFraction":1}}}}`))
		case "/v1internal:retrieveUserQuotaSummary":
			_, _ = w.Write([]byte(`{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":1}]}]}`))
		}
	}))
	defer prod.Close()

	daily := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitOrder = append(hitOrder, "daily:"+r.URL.Path)
		switch r.URL.Path {
		case "/v1internal:fetchAvailableModels":
			_, _ = w.Write([]byte(`{"models":{"gemini-4-carbon-high":{"quotaInfo":{"remainingFraction":0.6,"resetTime":"2026-11-16T18:26:27Z"}}}}`))
		case "/v1internal:retrieveUserQuotaSummary":
			_, _ = w.Write([]byte(`{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.6,"resetTime":"2026-11-16T18:26:27Z"},{"bucketId":"gemini-weekly","window":"weekly","remainingFraction":0.5,"resetTime":"2026-11-23T03:26:27Z"}]}]}`))
		}
	}))
	defer daily.Close()

	origURLs := antigravity.BaseURLs
	origBase := antigravity.BaseURL
	antigravity.BaseURLs = []string{prod.URL, daily.URL}
	antigravity.BaseURL = prod.URL
	t.Cleanup(func() {
		antigravity.BaseURLs = origURLs
		antigravity.BaseURL = origBase
	})
	fetcher := NewAntigravityQuotaFetcher(nil, &config.Config{})
	account := &Account{
		ID: 39, Platform: PlatformAntigravity, Type: AccountTypeOAuth,
		Credentials: map[string]any{"access_token": "tok", "project_id": "proj"},
	}

	res, err := fetcher.FetchQuota(context.Background(), account, "")
	require.NoError(t, err)
	require.Equal(t, "ULTRA", res.UsageInfo.SubscriptionTier)
	require.Contains(t, res.UsageInfo.AntigravityQuota, "gemini-4-carbon-high")
	require.Equal(t, 40, res.UsageInfo.AntigravityQuota["gemini-4-carbon-high"].Utilization)
	require.Len(t, res.UsageInfo.AntigravityQuotaGroups, 1)
	require.InDelta(t, 40.0, res.UsageInfo.AntigravityQuotaGroups[0].Windows[0].Utilization, 0.01)
	require.Equal(t, []string{
		"prod:/v1internal:loadCodeAssist",
		"daily:/v1internal:fetchAvailableModels",
		"daily:/v1internal:retrieveUserQuotaSummary",
	}, hitOrder)
}

func TestForkPatch_UpdateRepoAndPatchVersionComparison(t *testing.T) {
	require.Equal(t, "PIPIPIKa/sub2api", resolveUpdateGitHubRepo())
	t.Setenv("SUB2API_UPDATE_REPO", "custom/sub2api")
	require.Equal(t, "custom/sub2api", resolveUpdateGitHubRepo())

	require.Equal(t, -1, compareVersions("0.2.21", "0.2.21-p1"))
	require.Equal(t, -1, compareVersions("0.2.21-p1", "0.2.21-p2"))
	require.Equal(t, 0, compareVersions("v0.2.21-p1", "0.2.21-p1"))
	require.Equal(t, 1, compareVersions("0.2.21-p2", "0.2.21-p1"))
	require.Equal(t, -1, compareVersions("0.2.21-p2", "0.2.22"))
}
