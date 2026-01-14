# 更新日志

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.5] - 2026-01-14

### 修复
- 🔧 优化注册界面，移除邮箱字段
  - 注册仅需账号和密码，简化注册流程
  - 移除前端邮箱输入框和验证规则
  - 移除后端邮箱验证逻辑
- 🔧 优化系统锁定界面
  - 确保锁定界面为完整独立的错误界面
  - 更新版本号显示

### 新增
- 🔑 注册自动生成个人访问令牌
  - 注册成功后自动生成个人访问令牌
  - 令牌仅显示一次，提示用户妥善保存
  - 添加令牌复制功能
  - 后端添加 `RegisterWithToken` 方法
- 🌐 DNS 解析服务
  - 新增 DNS 解析后端服务 (`internal/service/dns_service.go`)
  - 新增 DNS 解析 API 接口 `/api/v1/dns/resolve`
  - 支持 POST 和 GET 两种请求方式
  - 支持 A、AAAA、CNAME、MX、TXT、NS 记录查询
  - 新增 DNS 解析前端界面 (`web/src/views/DNS.vue`)
  - 显示解析耗时和记录数量
  - 支持复制解析结果

## [1.0.4] - 2026-01-14

### 修复
- 🔧 修复使用条款界面顶部标题字体不够明亮的问题
  - 将 `.el-dialog__title`、`h3`、`h4` 的颜色从 CSS 变量改为明确的 `#ffffff`
- 🔧 修复登录界面和底部视图中的英文版权信息
  - 将 "Copyright © 2026 CYP. All rights reserved." 改为 "版权所有 © 2026 CYP"
  - 修复 Login.vue、Register.vue、Locked.vue、Footer.vue、About.vue、TermsDialog.vue 中的版权信息
- 🔧 修复注册界面报错问题
  - 添加后端注册接口 `/api/v1/auth/register`
  - 添加 `GetUserByEmail` 数据库查询函数用于邮箱重复检测
  - 添加 `RegisterRequest` 结构体和 `Register` 方法
- 🔧 修复前端版本号显示错误
  - 修复 Login.vue、Register.vue、Locked.vue、Footer.vue 中的默认版本号
  - 修复 app store 默认版本号为 1.0.4

### 新增
- 📝 用户注册 API 接口
  - 支持用户名、邮箱、密码注册
  - 用户名和邮箱重复检测
  - 密码加密存储

## [1.0.3] - 2026-01-14

### 修复
- 🔧 修复生产环境使用条款字体不够明亮的问题
  - 将 CSS 变量颜色改为明确的颜色值 #e6edf3 和 #c9d1d9
- 🔧 修复界面底部版本显示为 v未知 的问题
  - 添加多个版本接口尝试逻辑和默认版本号
- 🔧 优化锁定界面设计
  - 统一与登录界面风格，使用卡片式布局
  - 添加锁定原因中文映射显示
- 🌐 将后端所有英文错误消息改为中文
  - 修复 auth_handler、lock_middleware、security_middleware 英文消息
  - 修复 token_handler、signature_handler 英文消息
  - 修复 registry/handler、registry/sync_handler 英文消息
  - 修复 share_handler、sbom_handler、org_handler、lock_handler 英文消息
- 🌐 修复前端所有 Vue 组件 console 英文消息为中文
  - 修复 Dashboard、Images、Accelerator、Tokens 等组件
  - 修复 System、Share、Settings、Org、Config、Audit 等组件

### 新增
- 📝 添加用户注册界面和功能
  - 创建 Register.vue 注册页面
  - 添加注册路由配置
  - 在登录页面添加注册链接

## [1.0.2] - 2026-01-14

### 修复
- 🔧 修复生产环境访问根路径返回404的问题
  - 问题原因：后端路由未配置静态文件服务，导致前端页面无法访问
  - 解决方案：在 gateway/router.go 中添加 setupStaticFiles() 函数
  - 支持多路径查找静态文件目录（开发环境和Docker容器环境）

### 新增
- 🌐 前端静态文件服务支持
  - 自动检测静态文件目录（./web/dist 或 /app/web/dist）
  - 支持 /assets 静态资源路由
  - 支持 favicon.ico、vite.svg 等根目录静态文件
  - 默认 robots.txt 响应

- 🔄 SPA 路由支持
  - 未匹配的非API路由自动返回 index.html
  - 保持 /api、/v2、/health 等后端路由正常工作

- 🐳 Watchtower 自动更新配置
  - docker-compose.yaml 添加 Watchtower 服务
  - 每小时自动检测 Docker Hub 镜像更新
  - 自动清理旧镜像

### 变更
- 📦 Docker 镜像源更新为 cyp97/cyp-docker-registry

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
