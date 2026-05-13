# 更新日志

## v0.0.3

### ✨ 新功能

- **会话历史加载**
  - 继续或打开会话时显示会话信息（文件路径和消息数量）
  - 在 TUI 中加载并显示历史会话消息
  - 将历史消息加载到 Agent 上下文中以保持连续性
  - 中止时重置 Agent 以确保下次请求状态干净

### 🛠 改进

- **构建与分发系统**
  - 重构 Makefile，按平台划分构建和分发目标
  - 新增 `dist-linux`、`dist-darwin`、`dist-windows` 目标
  - 新增 `build-zip.sh` 用于 Windows zip 打包
  - 新增 `checksums` 目标用于发布校验
  - 更新 `build-deb.sh` 和 `build-tarball.sh` 支持全平台

### 📖 文档

- 文档网站右上角新增 GitHub 仓库跳转按钮
- 新增 v0.0.2 更新日志

---

## v0.0.2

### ✨ 新功能

- **一键安装脚本**
  - `install.sh` 适用于 Linux/macOS，自动从 GitHub Releases 下载
  - `install.ps1` 适用于 Windows PowerShell，支持通过 `VIBECODING_INSTALL_DIR` 自定义安装目录
  - 两个脚本均可自动检测平台/架构、校验完整性并配置 PATH

- **文档站重新设计**
  - 采用 Google Material Design 风格重新设计
  - 默认语言改为英文
  - 新增 Hash 路由，方便文档分享（如 `#/en/README`、`#/zh/configuration`）
  - 头部和 README 新增 Logo

- **品牌素材**
  - 新增 `docs/assets/icon.svg`（512×512）用于打包
  - 新增 `docs/assets/logo.svg`（128×128）用于 README 和小尺寸显示
  - 简洁专业的石板色调设计

- **构建系统**
  - 新增 `make build-windows` 目标（amd64 + arm64）
  - 新增 `make build-linux` 和 `make build-darwin` 目标
  - 更新 `make build-all` 使用平台专用目标

- **文档**
  - 新增 `docs/en/skills.md` 技能系统文档
  - 更新 README 和快速入门中的安装说明

### 🐛 问题修复

- 将素材移至 `docs/assets/` 以支持 GitHub Pages 部署

---

**完整变更日志**: https://github.com/fuckvibecoding/vibecoding/compare/v0.0.1...v0.0.3
