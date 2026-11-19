# 执行记录

## 2026-11-19 Gemini 4 Neon 兼容修复与发布

- 请求：修复 `gemini-4-neon-high` / `xhigh` 的项目层兼容缺口并发布。
- 提交：`044d80f2a`，`fix(gemini): add Gemini 4 Neon compatibility`。
- 发布：`v0.2.22-p5`。
- 修改：补充 Neon 模型目录、默认映射、旧账号运行时透传、推理模型识别、Gemini 4 thinkingLevel 转换、原生 Gemini fallback 和前端白名单。
- 保留：工作区原有 Codex 模型列表补丁。
- 本地验证：前端构建、嵌入式 Go 构建、Neon 相关 Go 测试、前端模型白名单测试、`git diff --check` 均通过。
- CI 验证：Release 前端构建、5 个平台二进制构建、镜像发布、GitHub Release 和 Security Scan 均成功。
- 发布地址：<https://github.com/PIPIPIKa/sub2api/releases/tag/v0.2.22-p5>
- 环境限制：Windows 本地 release matrix 辅助测试受 Python 3.9、Git Bash 路径和权限限制，GitHub Actions Linux 环境已完成对应发布验证。
