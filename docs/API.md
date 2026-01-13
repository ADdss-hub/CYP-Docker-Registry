# API 文档

CYP-Docker-Registry API 文档

**作者：** CYP | **联系方式：** nasDSSCYP@outlook.com

## 概述

本系统通过统一端口（默认 8080）提供所有 API 服务，包括：
- Docker Registry V2 API（兼容 Docker CLI）
- Web 管理 API
- 系统管理 API

## 基础信息

- **基础 URL**: `http://localhost:8080`
- **内容类型**: `application/json`
- **字符编码**: `UTF-8`

## 通用响应格式

### 成功响应

```json
{
  "success": true,
  "data": { ... }
}
```

### 错误响应

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "错误描述",
    "details": { ... }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 错误码

| 错误码 | HTTP 状态码 | 描述 |
|--------|------------|------|
| IMAGE_NOT_FOUND | 404 | 镜像不存在 |
| BLOB_NOT_FOUND | 404 | 镜像层不存在 |
| NOT_FOUND | 404 | 资源不存在 |
| INVALID_MANIFEST | 400 | 无效的镜像清单 |
| INVALID_REQUEST | 400 | 无效的请求 |
| AUTH_FAILED | 401 | 认证失败 |
| STORAGE_FULL | 507 | 存储空间不足 |
| UPSTREAM_ERROR | 502 | 上游仓库错误 |
| INTERNAL_ERROR | 500 | 内部错误 |

---

## 系统端点

### 健康检查

检查服务运行状态。

```
GET /health
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.0"
  }
}
```

### 获取版本信息

```
GET /api/version
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "full_version": "1.0.0 (unknown @ unknown)"
  }
}
```

---

## Docker Registry V2 API

兼容 Docker Registry V2 协议，支持 Docker CLI 直接操作。

### V2 基础端点

```
GET /v2/
```

**响应头：**
- `Docker-Distribution-API-Version: registry/2.0`

**响应示例：**

```json
{}
```

### 获取镜像清单

```
GET /v2/:name/manifests/:reference
```

**参数：**
- `name` - 镜像名称
- `reference` - 标签或摘要

**响应头：**
- `Docker-Distribution-API-Version: registry/2.0`
- `Content-Type: application/vnd.docker.distribution.manifest.v2+json`
- `Docker-Content-Digest: sha256:...`

### 推送镜像清单

```
PUT /v2/:name/manifests/:reference
```

**参数：**
- `name` - 镜像名称
- `reference` - 标签

**请求体：** 镜像清单 JSON

**响应：**
- 状态码：201 Created
- `Location: /v2/:name/manifests/:digest`

### 删除镜像清单

```
DELETE /v2/:name/manifests/:reference
```

**响应：** 202 Accepted

### 检查镜像清单

```
HEAD /v2/:name/manifests/:reference
```

**响应：** 200 OK（包含清单元数据头）

### 获取镜像层

```
GET /v2/:name/blobs/:digest
```

**参数：**
- `name` - 镜像名称
- `digest` - 层摘要（sha256:...）

**响应：** 二进制数据流

### 检查镜像层

```
HEAD /v2/:name/blobs/:digest
```

**响应：** 200 OK（包含层元数据头）

### 删除镜像层

```
DELETE /v2/:name/blobs/:digest
```

**响应：** 202 Accepted

### 开始上传镜像层

```
POST /v2/:name/blobs/uploads/
```

**查询参数：**
- `digest` - （可选）单次上传时的摘要

**响应：**
- 状态码：202 Accepted
- `Location: /v2/:name/blobs/uploads/:uuid`
- `Docker-Upload-UUID: :uuid`

### 上传镜像层数据

```
PATCH /v2/:name/blobs/uploads/:uuid
```

**请求体：** 二进制数据

**响应：**
- 状态码：202 Accepted
- `Range: 0-:size`

### 完成镜像层上传

```
PUT /v2/:name/blobs/uploads/:uuid?digest=sha256:...
```

**响应：**
- 状态码：201 Created
- `Location: /v2/:name/blobs/:digest`

### 列出镜像标签

```
GET /v2/:name/tags/list
```

**响应示例：**

```json
{
  "name": "myapp",
  "tags": ["latest", "v1.0.0", "v1.1.0"]
}
```

---

## 镜像管理 API

### 列出镜像

```
GET /api/images
```

**查询参数：**
- `page` - 页码（默认：1）
- `page_size` - 每页数量（默认：10）

**响应示例：**

```json
{
  "success": true,
  "data": {
    "images": [
      {
        "name": "myapp",
        "tag": "latest",
        "digest": "sha256:abc123...",
        "size": 52428800,
        "created_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
  }
}
```

### 搜索镜像

```
GET /api/images/search
```

**查询参数：**
- `q` - 搜索关键词
- `page` - 页码（默认：1）
- `page_size` - 每页数量（默认：10）

**响应示例：**

```json
{
  "success": true,
  "data": {
    "images": [...],
    "total": 5,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
  }
}
```

### 获取镜像详情

```
GET /api/images/:name
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "name": "myapp",
    "tags": [
      {
        "name": "myapp",
        "tag": "latest",
        "digest": "sha256:abc123...",
        "size": 52428800,
        "created_at": "2024-01-15T10:30:00Z",
        "layers": [...]
      }
    ]
  }
}
```

### 获取指定标签镜像

```
GET /api/images/:name/:tag
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "image": {
      "name": "myapp",
      "tag": "latest",
      "digest": "sha256:abc123...",
      "size": 52428800,
      "created_at": "2024-01-15T10:30:00Z"
    },
    "pull_cmd": "docker pull localhost:8080/myapp:latest"
  }
}
```

### 删除镜像

```
DELETE /api/images/:name/:tag
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "message": "镜像删除成功",
    "name": "myapp",
    "tag": "latest"
  }
}
```

---

## 镜像加速器 API

### 代理拉取镜像层

```
GET /api/accel/pull/:name/blobs/:digest
```

**响应：** 二进制数据流

### 代理拉取镜像清单

```
GET /api/accel/pull/:name/manifests/:reference
```

**响应：** 镜像清单 JSON

### 获取缓存统计

```
GET /api/accel/cache/stats
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "total_size": 1073741824,
    "max_size": 10737418240,
    "entry_count": 50,
    "hit_count": 1000,
    "miss_count": 100,
    "hit_rate": 0.909
  }
}
```

### 清空缓存

```
DELETE /api/accel/cache
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "message": "缓存已清空"
  }
}
```

### 删除缓存条目

```
DELETE /api/accel/cache/:digest
```

### 列出缓存条目

```
GET /api/accel/cache/entries
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "entries": [
      {
        "digest": "sha256:abc123...",
        "size": 10485760,
        "last_access": "2024-01-15T10:30:00Z",
        "access_count": 5
      }
    ],
    "count": 1
  }
}
```

### 列出上游源

```
GET /api/accel/upstreams
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "upstreams": [
      {
        "name": "Docker Hub",
        "url": "https://registry-1.docker.io",
        "priority": 1,
        "enabled": true
      }
    ],
    "count": 1
  }
}
```

### 添加上游源

```
POST /api/accel/upstreams
```

**请求体：**

```json
{
  "name": "阿里云",
  "url": "https://registry.cn-hangzhou.aliyuncs.com",
  "priority": 2
}
```

### 更新上游源

```
PUT /api/accel/upstreams/:name
```

**请求体：**

```json
{
  "url": "https://new-url.com",
  "priority": 1,
  "enabled": true
}
```

### 删除上游源

```
DELETE /api/accel/upstreams/:name
```

### 启用上游源

```
POST /api/accel/upstreams/:name/enable
```

### 禁用上游源

```
POST /api/accel/upstreams/:name/disable
```

### 检查上游源健康状态

```
GET /api/accel/upstreams/:name/health
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "name": "Docker Hub",
    "healthy": true,
    "status": "healthy"
  }
}
```

---

## 系统信息 API

### 获取系统信息

```
GET /api/system/info
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "os": "linux",
    "os_version": "Ubuntu 22.04",
    "arch": "amd64",
    "hostname": "server-01",
    "docker_version": "24.0.5",
    "containerd_version": "1.6.21",
    "cpu_cores": 4,
    "memory_total": 8589934592,
    "disk_total": 107374182400,
    "disk_free": 53687091200
  }
}
```

### 检查兼容性

```
GET /api/system/compatibility
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "compatible": true,
    "warnings": [],
    "errors": []
  }
}
```

### 刷新系统信息

```
GET /api/system/refresh
```

---

## 更新管理 API

### 检查更新

```
GET /api/update/check
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "current": "0.1.0",
    "latest": "0.2.0",
    "has_update": true,
    "release_at": "2024-01-20T00:00:00Z",
    "changelog": "新功能和修复..."
  }
}
```

### 获取更新状态

```
GET /api/update/status
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "status": "idle",
    "version_info": {
      "current": "0.1.0",
      "latest": "0.2.0",
      "has_update": true
    }
  }
}
```

### 下载更新

```
POST /api/update/download
```

**请求体：**

```json
{
  "version": "0.2.0"
}
```

### 应用更新

```
POST /api/update/apply
```

### 回滚更新

```
POST /api/update/rollback
```

---

## 凭证管理 API

### 列出凭证

```
GET /api/credentials
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "credentials": [
      {
        "registry": "docker.io",
        "username": "user",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

### 保存凭证

```
POST /api/credentials
```

**请求体：**

```json
{
  "registry": "docker.io",
  "username": "user",
  "password": "password"
}
```

### 获取凭证

```
GET /api/credentials/:registry
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "registry": "docker.io",
    "username": "user",
    "password": "********",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### 删除凭证

```
DELETE /api/credentials/:registry
```

---

## 同步管理 API

### 同步镜像

```
POST /api/sync
```

**请求体：**

```json
{
  "image_name": "myapp",
  "image_tag": "latest",
  "target_registry": "docker.io",
  "target_name": "username/myapp",
  "target_tag": "latest"
}
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "message": "同步任务已启动",
    "record": {
      "id": "sync-123",
      "status": "pending",
      "created_at": "2024-01-15T10:30:00Z"
    }
  }
}
```

### 获取同步历史

```
GET /api/sync/history
```

**查询参数：**
- `page` - 页码（默认：1）
- `page_size` - 每页数量（默认：10）

**响应示例：**

```json
{
  "success": true,
  "data": {
    "records": [...],
    "total": 10,
    "page": 1,
    "page_size": 10,
    "total_pages": 1
  }
}
```

### 获取同步记录

```
GET /api/sync/history/:id
```

### 重试同步

```
POST /api/sync/retry/:id
```

### 获取镜像同步历史

```
GET /api/sync/image/:name/:tag
```

---

## 使用示例

### 使用 Docker CLI 推送镜像

```bash
# 标记镜像
docker tag myapp:latest localhost:8080/myapp:latest

# 推送镜像
docker push localhost:8080/myapp:latest
```

### 使用 Docker CLI 拉取镜像

```bash
docker pull localhost:8080/myapp:latest
```

### 使用 curl 调用 API

```bash
# 获取镜像列表
curl http://localhost:8080/api/images

# 搜索镜像
curl "http://localhost:8080/api/images/search?q=myapp"

# 删除镜像
curl -X DELETE http://localhost:8080/api/images/myapp/latest
```

---

## 版权声明

Copyright © 2026 CYP. All rights reserved.


---

## 认证 API（安全增强）

### 用户登录

```
POST /api/v1/auth/login
```

**请求体：**

```json
{
  "username": "admin",
  "password": "your_password",
  "captcha": "abc123"
}
```

**成功响应：**

```json
{
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin",
    "is_active": true
  },
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "session": {
    "id": "sess_abc123",
    "ip": "192.168.1.100"
  },
  "must_change_password": false,
  "lock_warning": false
}
```

**失败响应：**

```json
{
  "error": "Invalid credentials",
  "code": "login_failure",
  "remaining_attempts": 2
}
```

### 用户登出

```
POST /api/v1/auth/logout
```

**响应：**

```json
{
  "message": "Logged out successfully"
}
```

### 验证令牌

```
POST /api/v1/auth/verify-token
```

**请求体：**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应：**

```json
{
  "valid": true,
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

### 获取当前用户

```
GET /api/v1/auth/me
```

**响应：**

```json
{
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin",
    "is_active": true
  }
}
```

### 心跳检测

```
GET /api/v1/auth/heartbeat
```

**响应：**

```json
{
  "status": "ok",
  "timestamp": 1705312200
}
```

---

## 系统锁定 API

### 获取锁定状态

```
GET /api/v1/system/lock/status
```

**响应：**

```json
{
  "is_locked": true,
  "lock_reason": "too_many_failed_attempts",
  "lock_type": "rule_triggered",
  "locked_at": "2026-01-13T10:30:00Z",
  "locked_by_ip": "192.168.1.50",
  "locked_by_user": "",
  "require_manual": true
}
```

### 解锁系统

```
POST /api/v1/system/lock/unlock
```

**请求体：**

```json
{
  "password": "admin_password",
  "recovery_key": "optional_recovery_key"
}
```

**响应：**

```json
{
  "message": "System unlocked successfully"
}
```

### 手动锁定系统

```
POST /api/v1/system/lock/lock
```

**请求体：**

```json
{
  "reason": "Maintenance mode"
}
```

**响应：**

```json
{
  "message": "System locked successfully"
}
```

---

## 审计日志 API

### 获取审计日志

```
GET /api/v1/audit/logs
```

**查询参数：**
- `page` - 页码（默认：1）
- `page_size` - 每页数量（默认：20）
- `event_type` - 事件类型过滤
- `start_date` - 开始日期
- `end_date` - 结束日期

**响应：**

```json
{
  "logs": [
    {
      "id": 1,
      "timestamp": "2026-01-13T10:30:00Z",
      "level": "info",
      "event": "login_success",
      "user_id": 1,
      "username": "admin",
      "ip_address": "192.168.1.100",
      "resource": "/api/v1/auth/login",
      "action": "login",
      "status": "success",
      "blockchain_hash": "abc123..."
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### 导出审计日志

```
GET /api/v1/audit/logs/export
```

**查询参数：**
- `start_date` - 开始日期
- `end_date` - 结束日期

**响应：** JSON 文件下载

---

## 安全相关错误码

| 错误码 | HTTP 状态码 | 描述 |
|--------|------------|------|
| no_auth_header | 401 | 缺少认证头 |
| invalid_jwt | 401 | 无效的 JWT 令牌 |
| invalid_token | 401 | 无效的访问令牌 |
| invalid_format | 401 | 无效的认证格式 |
| inactive_user | 401 | 用户已禁用 |
| ip_mismatch | 401 | IP 地址变更 |
| login_failure | 401 | 登录失败 |
| system_locked | 403 | 系统已锁定 |
| csrf_missing | 403 | 缺少 CSRF 令牌 |
| csrf_invalid | 403 | 无效的 CSRF 令牌 |
| rate_limit_exceeded | 429 | 请求过于频繁 |

---

## 认证请求示例

### 使用 JWT 令牌

```bash
# 登录获取令牌
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}' | jq -r '.token')

# 使用令牌访问 API
curl http://localhost:8080/api/images \
  -H "Authorization: Bearer $TOKEN"
```

### 使用个人访问令牌

```bash
curl http://localhost:8080/api/images \
  -H "Authorization: Token pat_abc123..."
```

---

## 安全最佳实践

1. **始终使用 HTTPS** - 在生产环境中配置 TLS
2. **定期轮换令牌** - 设置合理的令牌过期时间
3. **监控审计日志** - 定期检查异常访问
4. **配置 IP 白名单** - 限制管理接口访问
5. **启用双因素认证** - 增强账户安全

---

**文档版本**: v1.0.0  
**最后更新**: 2026-01-14
