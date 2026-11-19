# 执行记录

## 2026-11-19 Gemini 4 Neon 兼容修复

- 请求：修复 `gemini-4-neon-high` / `xhigh` 的项目层兼容缺口。
- 修改：补充 Neon 模型目录、默认映射、旧账号运行时透传、推理模型识别、Gemini 4 thinkingLevel 转换、原生 Gemini fallback 和前端白名单。
- 保留：工作区原有 Codex 模型列表补丁。
- 验证：相关 Go 单测通过，`git diff --check` 通过。
- 未完成：未提交、未推送、未发版。前端 Vitest 因 `frontend/node_modules` 缺失未运行。
