# 执行记录

## 2026-11-23 同步官方 v0.2.24

- 从 fork `v0.2.22-p7` 合入官方 `328db26f4`，包含 `v0.2.24` 和随后的 VERSION 回写。fork 发布版本为 `v0.2.24-p1`。
- 合并时保留旧账号的订阅等级识别、Gemini 推理档位和模型目录能力，采用上游的固定额度查询端点和 Codex 音频判定。已通过定向 Go 回归和前端生产构建。
- Release、发布提交 CI 和 Security Scan 已通过。已核对五份安装包摘要及 GHCR 双架构索引。主分支随后修正了两个测试文件，本地四个相关 Go 包的完整单测通过。
- [同步与验证记录](03-execution/20261123-upstream-v0.2.24.md)

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

## 2026-11-19 Production model mapping cleanup

- Target：BWH production Sub2API accounts 39 `KONG-GOOGLE` and 40 `GAO-GOOGLE`.
- Backup：`/root/sub2api-backups/accounts-model-mapping-20261119T085116Z.sql`。
- Changed: removed `gemini-4-hydrogen-high/low/medium/xhigh` and consolidated Carbon tier keys into `gemini-4-carbon`.
- Preserved: all Argon mappings.
- Verification: both accounts have Carbon base, no Carbon tier keys, and no Hydrogen tier keys. Application container is `running healthy`, `/health` returned `{"status":"ok"}`。
