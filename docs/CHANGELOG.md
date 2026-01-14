# 更新日志

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.1] - 2026-01-14

### 修复
- 🔧 修复 Docker 容器启动时数据库初始化失败的问题
  - 问题原因：go-sqlite3 是 CGO 库，需要 C 编译器支持，但 Dockerfile 使用 CGO_ENABLED=0 编译
  - 解决方案：将 go-sqlite3 替换为纯 Go 实现的 modernc.org/sqlite 驱动
  - 影响范围：所有 Docker 部署环境，包括飞牛 NAS、群晖、QNAP 等

### 新增
- 🔄 完善自动更新检测功能
  - 支持从 GitHub Releases 检测新版本
  - Docker 环境自动检测，提供 Watchtower 自动更新配置
  - 非 Docker 环境支持自动下载和应用更新
  - 新增 API: `/api/update/docker-command`、`/api/update/watchtower-config`
  - 后台定时检查更新（默认每小时）

### 优化
- 📦 数据库驱动升级
  - 从 github.com/mattn/go-sqlite3 v1.14.22 迁移到 modernc.org/sqlite v1.29.1
  - 无需 CGO 支持，生成静态链接的二进制文件
  - 提升跨平台兼容性，支持 Alpine 等精简镜像

- 🛠 版本更新工具增强
  - 自动更新 Docker 部署文件版本号
  - 自动更新所有脚本文件版本号
  - 自动更新项目文档版本号

### 变更
- 🔄 数据库连接参数调整
  - SQLite 驱动名从 "sqlite3" 改为 "sqlite"
  - WAL 模式和忙等待超时参数格式适配新驱动

## [1.0.0] - 2026-01-14

### 新增
- 🔐 零信任安全架构
  - 强制登录认证（所有页面/API 必须登录）
  - 入侵检测系统（IDS）
  - 自动锁定机制（硬件/网络/服务级别）
  - 区块链哈希审计日志

- 🚀 镜像加速功能
  - 多源镜像代理（Docker Hub、阿里云、腾讯云等）
  - 智能缓存（LRU 策略）
  - 带宽限制和 QoS

- 📦 供应链安全
  - 镜像签名（ECDSA-P256）
  - SBOM 生成（SPDX/CycloneDX 格式）
  - 漏洞扫描集成

- 🏢 团队协作
  - 组织管理
  - RBAC 权限控制
  - 分享链接（密码保护、有效期）
  - 个人访问令牌

- 🌍 全平台支持
  - Docker 部署
  - Kubernetes 部署
  - NAS 支持（群晖/QNAP）
  - 树莓派支持
  - 云环境支持（AWS/阿里云）

- 📊 监控与审计
  - 实时 WebSocket 推送
  - Prometheus 指标
  - 完整审计日志
  - 健康检查端点

- 🛠 CLI 工具
  - 系统状态查询
  - 锁定/解锁操作
  - 审计日志查看/导出

### 安全
- 默认启用强制登录
- 登录失败 3 次自动锁定
- JWT Token 有效期 24 小时
- IP 绑定会话
- 审计日志防篡改

### 文档
- 完整 API 文档
- 部署指南
- 安全指南
- 安装说明

---

## [未发布]

### 计划中
- OPA 策略引擎
- AI 异常检测
- 联邦学习威胁情报
- Windows/macOS 原生支持
- BI 分析仪表板

---

**作者**: CYP  
**邮箱**: nasDSSCYP@outlook.com
