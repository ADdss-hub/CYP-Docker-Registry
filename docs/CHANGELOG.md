# 更新日志

所有重要的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.2.0] - 2026-01-15

### 修复
- 🔧 修复组织管理、分享管理、访问令牌页面自动跳转登录界面的问题
  - 为 `/api/v1/orgs`、`/api/v1/share`、`/api/v1/tokens` 路由正确应用认证中间件
  - 创建 `createAuthCheckMiddleware` 方法统一处理认证检查
- 🔧 修复关于页面版本号显示不完全的问题
  - 新增 `/api/version/full` 接口返回完整版本信息
  - 优化前端版本 API 调用，正确处理响应数据格式
- 🔧 修复镜像加速命中率显示 NaN% 的问题
  - 增强 `hitRatePercent` 计算属性对 null/undefined/NaN 值的处理
- 🔧 全面检查并修复所有界面中的数据引入问题
  - 确保所有 API 响应数据正确解析
  - 统一数据格式处理逻辑
- 🔧 修复系统锁定允许手动解锁的安全漏洞
  - 系统锁定后不允许手动解锁
  - 只能联系管理员或重新安装系统解锁
- 🔧 修复 P2P 服务无法使用的问题
  - 在 router.go 中添加 P2P 服务初始化和路由注册
  - 配置文件添加 P2P 配置支持
- 🔧 修复 DNS 服务没有对系统自动应用和配置的问题
  - 系统启动时自动应用 DNS 配置到全局
  - 支持自定义 DNS 服务器列表

### 新增
- ✨ 新增使用方法界面 (`/usage`)
  - 提供完整的系统使用指南
  - 包含快速开始、访问令牌、镜像加速、组织管理等使用说明
  - 包含常见问题解答
- ✨ 新增密码安全保护服务 (`internal/service/security_service.go`)
  - 检测强制查询密码的行为
  - 超过阈值时立即删除所有数据库信息并锁定系统
  - 创建安全标记文件记录触发信息
- ✨ 新增全局服务管理器 (`internal/service/global_service.go`)
  - 镜像加速、DNS、P2P 服务自动应用到系统全局配置
  - 系统启动时自动初始化并应用配置
  - 自动生成 Docker daemon 镜像加速配置
  - 自动生成 DNS 配置文件
  - 自动生成 P2P 配置文件
  - 新增 API 接口：
    - `GET /api/v1/global/status` - 获取全局服务状态
    - `POST /api/v1/global/apply/accelerator` - 手动应用镜像加速配置
    - `POST /api/v1/global/apply/dns` - 手动应用 DNS 配置
    - `POST /api/v1/global/apply/p2p` - 手动应用 P2P 配置
- ✨ 镜像加速服务全局集成
  - DNS 解析器自动应用到镜像加速代理服务
  - P2P 服务自动集成到镜像加速代理服务
  - 拉取镜像时优先从 P2P 网络获取
  - 成功拉取后自动向 P2P 网络宣布
- ✨ Registry Handler 全局服务集成
  - 签名服务自动集成，推送镜像时自动签名
  - SBOM 服务自动集成，推送镜像时自动生成 SBOM
  - 压缩服务自动集成，支持自动压缩/解压（默认关闭）
  - 拉取镜像时自动验证签名

### 变更
- 🔧 版本号同步更新至 1.2.0
- 🔧 前端版本文件 `frontend/src/utils/version.ts` 同步更新

## [1.1.0] - 2026-01-15

### 修复
- 🔧 修复 Docker 容器中系统信息显示 NaN undefined 的问题
  - 优化内存信息获取，支持 `/proc/meminfo` 和 cgroup 限制读取
  - 优化磁盘信息获取，优先从根目录获取，支持 `/data` 挂载点
- 🔧 修复 Dashboard 磁盘使用显示问题
  - 添加 `diskUsedFormatted` 计算属性，避免模板中直接计算产生 NaN
- 🔧 修复 Accelerator 缓存统计 API 数据解析问题
  - 正确处理 `{ success: true, data: {...} }` 响应格式
- 🔧 修复 System 页面 API 数据解析问题
- 🔧 修复 P2P 页面卡片标题颜色不清晰问题
- 🔧 修复 DNS 解析页面表格行白色背景问题
- 🔧 修复所有对话框标题颜色为白色，增强可读性
- 🔧 修复 el-descriptions 组件深色主题适配
- 🔧 修复 el-table 全局深色主题样式
- 🔧 修复登录页面版本号只显示 "v" 的问题

### 新增
- ✨ 版本同步脚本添加前端代码版本号同步支持
  - 支持同步 `web/src/stores/app.ts` 中的 DEFAULT_VERSION
  - 支持同步 `web/src/views/Login.vue` 中的版本显示

### 变更
- 🔧 Dockerfile 添加 `coreutils` 和 `procps` 包支持系统信息获取

## [1.0.9] - 2026-01-14

### 修复
- 🔧 修复全局对话框标题颜色不够明亮的问题
  - 将 `.el-dialog__title` 颜色从 CSS 变量改为明确的白色 `#ffffff`
  - 修复使用条款对话框标题颜色
  - 添加全局对话框标题白色样式覆盖，确保所有对话框标题都清晰可见

## [1.0.8] - 2026-01-14

### 修复
- 🔧 全面优化所有对话框标题样式，增强深色主题下的可读性
  - el-dialog 全局样式增强，包括标题、头部、主体、底部样式
  - Images.vue 详情对话框样式优化
  - 对话框标题字体加粗 (font-weight: 600)
  - 对话框头部添加底部边框分隔线
  - 对话框底部添加顶部边框分隔线
  - 对话框关闭按钮颜色优化

## [1.0.7] - 2026-01-14

### 修复
- 🔧 修复全局字体颜色对比度不足问题
  - 调整 `--muted-text` 从 `#8b949e` 改为 `#c9d1d9`
  - 新增 `--label-text` 和 `--card-title-color` CSS 变量
- 🔧 修复 NaN undefined 显示问题
  - 增强 `formatBytes` 函数对 null/undefined/NaN 值的处理
  - 修复 Dashboard.vue、System.vue、Accelerator.vue 中的格式化函数
- 🔧 修复系统信息 formatDuration 函数错误
  - 将 `string(rune())` 改为 `fmt.Sprintf()`，修复数字转换为 Unicode 字符的问题
- 🔧 修复版本号显示为空问题
  - System.vue 版本信息添加默认值处理
  - Footer.vue 版本获取逻辑优化
- 🔧 修复 DNS 解析表格样式问题
  - 增强表格行背景色和边框样式
  - 修复深色主题下表格可读性
- 🔧 修复添加上游源对话框标题样式
- 🔧 修复兼容性检查信息显示问题
  - 支持对象和字符串两种格式的警告/错误信息
- 🔧 Element Plus 组件深色主题适配
  - el-dialog、el-descriptions、el-form-item、el-switch 组件样式优化
  - P2P 页面 el-descriptions 组件深色主题适配

## [1.0.6] - 2026-01-14

### 修复
- 🔧 修复 formatBytes 函数处理空值问题
  - Dashboard.vue 中 formatBytes 函数增加空值检查
  - System.vue 中 formatBytes 函数增加空值检查
  - Accelerator.vue 中 formatBytes 函数增加空值检查
- 🔧 修复磁盘使用百分比计算 NaN 问题
- 🔧 修复内存显示格式化问题

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
