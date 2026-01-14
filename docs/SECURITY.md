# CYP-Docker-Registry 安全指南

## 概述

CYP-Docker-Registry 采用零信任架构设计，所有访问必须经过认证，任何绕过尝试都会触发系统锁定。

## 核心安全特性

### 1. 强制登录认证

- 所有页面和 API 必须登录后访问
- 白名单仅包含：登录页、锁定页、健康检查
- JWT Token 有效期 24 小时
- 支持 IP 绑定，IP 变更需重新登录

### 2. 入侵检测系统 (IDS)

检测规则：
| 规则名称 | 描述 | 阈值 | 动作 |
|----------|------|------|------|
| direct_url_access | 直接访问 URL 绕过登录 | 1 | 锁定 |
| forged_jwt | 伪造 JWT 令牌 | 1 | 锁定 |
| token_replay | 重放已失效令牌 | 2 | 锁定 |
| login_failure | 登录失败 | 3 | 锁定 |
| ip_change_mid_session | 会话中 IP 变更 | 1 | 警告 |

### 3. 自动锁定机制

触发锁定后：
- **硬件锁定**: CPU 限制到 10%，内存限制到 10%
- **网络锁定**: 阻止所有入站连接
- **服务锁定**: 暂停所有工作流，启用只读模式

### 4. 审计日志

- 记录所有请求和认证事件
- 区块链哈希防篡改
- 不可变存储
- 保留期限：1 年

## 默认配置

```yaml
security:
  force_login:
    enabled: true
    mode: "strict"
  
  failed_attempts:
    max_login_attempts: 3
    max_token_attempts: 5
    lock_duration: "1h"
  
  auto_lock:
    enabled: true
    lock_on_bypass_attempt: true
```

## 解锁方法

### 方法 1: 使用解锁脚本

```bash
./scripts/unlock.sh
```

### 方法 2: CLI 工具

```bash
./cyp-docker-registry cli unlock --password <admin_password>
```

### 方法 3: Docker 环境

```bash
docker exec -it cyp-docker-registry /app/scripts/unlock.sh
```

### 方法 4: 紧急恢复

```bash
# 重启容器（如果未启用持久化锁定）
docker restart cyp-docker-registry

# 或使用恢复密钥
./cyp-docker-registry cli unlock --recovery-key <recovery_key>
```

## 最佳实践

1. **首次登录后立即修改默认密码**
2. **启用双因素认证（如果可用）**
3. **定期审查审计日志**
4. **配置通知渠道接收安全告警**
5. **定期备份配置和数据**

## 安全事件响应

1. 收到锁定通知后，立即检查审计日志
2. 确认是否为误触发
3. 如果是真实攻击，保留日志证据
4. 解锁后修改所有凭据
5. 审查并加强安全配置

## 联系方式

- 安全问题报告: security@cyp-docker-registry.com
- 邮箱: nasDSSCYP@outlook.com

---

**版本**: v1.0.6  
**最后更新**: 2026-01-14
