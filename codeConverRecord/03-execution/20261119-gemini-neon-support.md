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

go test -C backend -tags unit ./internal/service -run "TestApplyGeminiCompatThinkingConfig_Neon|TestNeonGenerationThinkingLevels|TestNeonTierPreservesUpstreamSelection|TestGeminiCarbonPricingAliases|TestResolveGeminiThinkingVariantForLevel_CompatPathResolvesBareModel|TestForkPatch_GeminiVariantsExposeCodexReasoningLevels" -count=1
通过

pnpm run build
通过

pnpm exec vitest run src/composables/__tests__/useModelWhitelist.spec.ts
27 tests passed

go build -tags embed ./cmd/server
通过

git diff --check
通过
```

GitHub Actions `v0.2.22-p5` 的前端构建、5 个平台二进制构建、镜像发布、GitHub Release 和 Security Scan 均通过。

Windows 本地 release matrix 辅助测试使用 Python 3.9，且 Git Bash 路径受本机权限限制。该检查在 GitHub Actions Linux / Python 3.12 发布流程中通过。

## 发布状态

- Commit：`044d80f2a`
- Tag：`v0.2.22-p5`
- Release：<https://github.com/PIPIPIKa/sub2api/releases/tag/v0.2.22-p5>
- 发布构件：Windows amd64、Linux amd64、Linux arm64、Darwin amd64、Darwin arm64 及 checksums
