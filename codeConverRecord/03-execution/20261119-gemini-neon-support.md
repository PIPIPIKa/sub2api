# Gemini 4 Neon 支持修复

## 变更范围

- `backend/internal/domain/constants.go`
- `backend/internal/pkg/antigravity/claude_types.go`
- `backend/internal/pkg/antigravity/request_transformer.go`
- `backend/internal/pkg/gemini/models.go`
- `backend/internal/pkg/geminicli/models.go`
- `backend/internal/service/account.go`
- `backend/internal/service/gemini_compat_thinking.go`
- `frontend/src/composables/useModelWhitelist.ts`

Neon 支持裸模型和 `low`、`medium`、`high`、`xhigh` 档位。带档位的模型保持上游模型名选择的思考深度，裸模型根据请求档位生成 `thinkingLevel`，避免被当作普通 Gemini 模型追加预算参数和停止序列。

## 验证

```text
go test -C backend -tags unit ./internal/pkg/antigravity ./internal/pkg/gemini ./internal/pkg/geminicli ./internal/domain -count=1
通过

go test -C backend -tags unit ./internal/service -run "TestApplyGeminiCompatThinkingConfig_Neon|TestNeonGenerationThinkingLevels|TestNeonTierPreservesUpstreamSelection|TestGeminiCarbonPricingAliases|TestResolveGeminiThinkingVariantForLevel_CompatPathResolvesBareModel" -count=1
通过

git diff --check
通过
```

前端 Vitest 未运行，因为本地缺少 `frontend/node_modules`。完整 `internal/service` 测试仍有工作区原有的 `max/xhigh` 期望不一致，未在本次 Neon 修复中改动。

## 发布状态

未提交、未推送、未构建 release 构件、未发版。
