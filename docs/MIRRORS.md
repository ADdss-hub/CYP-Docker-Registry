# 国内镜像加速配置指南

**测试时间**: 2026-01-13  
**测试环境**: Windows  
**状态**: ✅ 已配置并验证

## 当前配置

| 类型 | 镜像源 | 状态 |
|------|--------|------|
| Go Proxy | https://mirrors.aliyun.com/goproxy | ✅ 已配置 |
| NPM Registry | https://repo.huaweicloud.com/repository/npm | ✅ 已配置 |

## 测试结果

### Go 模块代理

| 镜像源 | URL | 状态 | 延迟 |
|--------|-----|------|------|
| 阿里云 | https://mirrors.aliyun.com/goproxy | ✅ 可用 | 210 ms |
| 七牛云 (goproxy.cn) | https://goproxy.cn | ❌ 不可用 | - |
| goproxy.io | https://goproxy.io | ❌ 不可用 | - |
| 百度 | https://goproxy.baidu.com | ❌ 不可用 | - |
| 官方 | https://proxy.golang.org | ❌ 不可用 | - |

### NPM 镜像

| 镜像源 | URL | 状态 | 延迟 |
|--------|-----|------|------|
| 华为云 | https://repo.huaweicloud.com/repository/npm | ✅ 可用 | 1459 ms |
| cnpm | https://r.cnpmjs.org | ✅ 可用 | 较慢 |
| npmmirror (淘宝) | https://registry.npmmirror.com | ❌ 不可用 | - |
| 腾讯云 | https://mirrors.cloud.tencent.com/npm | ❌ 不可用 | - |
| 官方 | https://registry.npmjs.org | ❌ 不可用 | - |

## 推荐配置

### Go 模块代理

```bash
# 设置阿里云代理
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy,direct
go env -w GOSUMDB=sum.golang.google.cn

# 验证配置
go env GOPROXY
```

### NPM 镜像

```bash
# 设置华为云镜像
npm config set registry https://repo.huaweicloud.com/repository/npm

# 验证配置
npm config get registry
```

### 备选配置

如果上述镜像不可用，可尝试以下备选：

**Go 代理备选**:
```bash
# 七牛云
go env -w GOPROXY=https://goproxy.cn,direct

# goproxy.io
go env -w GOPROXY=https://goproxy.io,direct
```

**NPM 备选**:
```bash
# 淘宝镜像
npm config set registry https://registry.npmmirror.com

# cnpm
npm config set registry https://r.cnpmjs.org
```

## 一键配置脚本

### Windows (PowerShell)

```powershell
# 配置 Go 代理
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy,direct
go env -w GOSUMDB=sum.golang.google.cn

# 配置 NPM 镜像
npm config set registry https://repo.huaweicloud.com/repository/npm

Write-Host "配置完成!" -ForegroundColor Green
```

### Linux/macOS

```bash
# 配置 Go 代理
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy,direct
go env -w GOSUMDB=sum.golang.google.cn

# 配置 NPM 镜像
npm config set registry https://repo.huaweicloud.com/repository/npm

echo "配置完成!"
```

## 安装依赖

配置完成后，执行以下命令安装项目依赖：

```bash
# Go 依赖
go mod tidy

# 前端依赖
cd web
npm install
```

## 恢复默认配置

如需恢复官方源：

```bash
# Go
go env -w GOPROXY=https://proxy.golang.org,direct
go env -w GOSUMDB=sum.golang.org

# NPM
npm config set registry https://registry.npmjs.org
```

---

**文档版本**: v1.0.0  
**最后更新**: 2026-01-13

## 验证结果

```
✅ Go 依赖安装成功
✅ NPM 依赖安装成功  
✅ 项目构建成功 (bin/cyp-registry.exe - 25.7 MB)
```
