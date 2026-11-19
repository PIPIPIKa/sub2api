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
	"github.com/tidwall/gjson"
)

func TestForkPatch_CodexDuplicateBareAliases(t *testing.T) {
	got := FilterCodexModelIDsForGroup([]string{"gpt-6", "gpt-6-astra", "gpt-5.6", "gpt-5.6-sol"}, nil)
	require.Equal(t, []string{"gpt-6-astra", "gpt-5.6-sol"}, got)

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
}

func TestForkPatch_GeminiVariantsExposeCodexReasoningLevels(t *testing.T) {
	require.Equal(t, []string{"gemini-4-carbon"}, FilterCodexModelIDsForGroup([]string{
		"gemini-4-carbon", "gemini-4-carbon-low", "gemini-4-carbon-medium", "gemini-4-carbon-high",
	}, nil))
	require.Equal(t, []string{"gemini-4-carbon-high"}, FilterCodexModelIDsForGroup([]string{
		"gemini-4-carbon-high",
	}, nil))

	account := Account{
		ID:       39,
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{"model_mapping": map[string]any{
			"gemini-4-carbon-low":    "gemini-4-carbon-low",
			"gemini-4-carbon-medium": "gemini-4-carbon-medium",
			"gemini-4-carbon-high":   "gemini-4-carbon-high",
		}},
	}

	body, err := buildCodexModelsManifestForAccounts(
		PlatformAntigravity,
		[]string{"gemini-4-carbon"},
		[]Account{account},
		nil,
		nil,
		true,
	)
	require.NoError(t, err)
	models := decodeCodexManifestModels(t, body)
	require.Len(t, models, 1)
	require.Equal(t, "gemini-4-carbon", models[0]["slug"])
	require.Equal(t, "high", models[0]["default_reasoning_level"])

	levels, ok := models[0]["supported_reasoning_levels"].([]any)
	require.True(t, ok)
	require.Equal(t, []any{
		map[string]any{"effort": "low", "description": "Fast responses with lighter reasoning"},
		map[string]any{"effort": "medium", "description": "Balanced reasoning for most coding tasks"},
		map[string]any{"effort": "high", "description": "Greater reasoning depth for coding and agent tasks"},
	}, levels)

	for _, level := range []string{"low", "medium", "high"} {
		mapped, matched := resolveGeminiThinkingVariantForLevel(&account, "gemini-4-carbon", level)
		require.True(t, matched)
		require.Equal(t, "gemini-4-carbon-"+level, mapped)
	}

	hydrogen := account
	hydrogen.Credentials["model_mapping"].(map[string]any)["gemini-4-hydrogen-xhigh"] = "gemini-4-hydrogen-xhigh"
	mapped, matched := resolveGeminiThinkingVariantForLevel(&hydrogen, "gemini-4-hydrogen", "xhigh")
	require.True(t, matched)
	require.Equal(t, "gemini-4-hydrogen-xhigh", mapped)
}

func TestForkPatch_ServiceTiersIntersectAndOfficialOAuthDefault(t *testing.T) {
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

func TestForkPatch_AntigravityResponsesMaxOutputTokensDefault64000(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want int64
	}{
		{
			name: "omitted defaults to 64000 instead of 8192",
			body: `{"model":"gemini-4-carbon-high","input":"hello","stream":true}`,
			want: 64000,
		},
		{
			name: "explicit max_output_tokens preserved",
			body: `{"model":"gemini-4-carbon-high","input":"hello","stream":true,"max_output_tokens":16384}`,
			want: 16384,
		},
		{
			name: "above 64000 clamped to 64000",
			body: `{"model":"gemini-4-carbon-high","input":"hello","stream":true,"max_output_tokens":99999}`,
			want: 64000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{antigravityCompatSuccessResponse()}}
			svc := newAntigravityCompatService(config.GatewayConfig{MaxLineSize: defaultMaxLineSize}, upstream)
			raw := []byte(tc.body)
			c, _ := newAntigravityCompatContext(http.MethodPost, "/v1/responses", raw)
			account := newAntigravityCompatAccount(AccountTypeOAuth)
			account.Credentials["model_mapping"] = map[string]any{"gemini-4-carbon-high": "gemini-4-carbon-high"}

			result, err := svc.ForwardAsResponses(context.Background(), c, account, raw, nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, upstream.requestBodies, 1)
			require.Equal(t, tc.want, gjson.GetBytes(upstream.requestBodies[0], "request.generationConfig.maxOutputTokens").Int())
		})
	}
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
