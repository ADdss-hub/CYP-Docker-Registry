# 统一版本管理系统 - 快速开始

## 简介

统一版本管理系统是一个模块化、自动化的版本号管理工具，提供完整的版本更新工作流。

## 快速使用

### 1. 查看当前版本信息

```bash
npm run version:info
```

**输出示例：**
```
📊 版本信息:

当前版本: 1.15.7

版本建议:
  Patch: 1.15.8
  Minor: 1.16.0
  Major: 2.0.0
```

### 2. 递增版本号

#### 递增补丁版本 (1.15.7 → 1.15.8)

```bash
npm run version:patch
```

#### 递增次版本 (1.15.7 → 1.16.0)

```bash
npm run version:minor
```

#### 递增主版本 (1.15.7 → 2.0.0)

```bash
npm run version:major
```

### 3. 更新到指定版本

```bash
npm run version:update 2.0.0
```

### 4. 验证版本系统

```bash
npm run version:validate
```

### 5. 版本历史管理

#### 生成版本历史文档

```bash
npm run version:history
```

#### 查看版本统计信息

```bash
npm run version:history-stats
```

**输出示例：**
```
📊 版本统计信息:

总版本数: 58
主版本更新: 0 次
次版本更新: 9 次
补丁更新: 44 次
首个版本: v1.9.4
最新版本: v1.15.7
```

#### 清理重复的历史记录

```bash
npm run version:history-clean
```

## 完整工作流示例

### 场景：发布一个补丁版本

```bash
# 1. 查看当前版本
npm run version:info

# 2. 验证系统状态
npm run version:validate

# 3. 递增补丁版本
npm run version:patch

# 4. 构建前端
npm run build:frontend

# 5. 复制到后端
# Windows:
Copy-Item -Path "frontend\dist\*" -Destination "backend\dist\" -Recurse -Force

# 6. 提交更改
git add .
git commit -m "chore: bump version to 1.15.8"
git tag v1.15.8
git push && git push --tags
```

## 执行流程

当你运行 `npm run version:patch` 时，系统会自动执行以下步骤：

```
🚀 开始版本更新流程...

📋 步骤 1/5: 验证版本号
   ✓ 格式验证通过
   ✓ 硬编码检查通过

📋 步骤 2/5: 检查版本冲突
   ✓ 1.15.7 → 1.15.8

📋 步骤 3/5: 写入版本号
   ✓ VERSION 文件: 1.15.8
   ✓ package.json: 1.15.8
   ✓ frontend/package.json: 1.15.8
   ✓ backend/package.json: 1.15.8
   ✓ 前端版本文件: 1.15.8

📋 步骤 4/5: 验证写入结果
   ✓ 验证通过

📋 步骤 5/5: 更新版本历史记录
   ✓ 版本历史已更新

✅ 版本更新完成！耗时 234ms

📊 更新摘要:
   版本变更: 1.15.7 → 1.15.8
   更新时间: 2025/12/30 21:45:30

📝 已更新的文件:
   ✓ VERSION
   ✓ package.json
   ✓ frontend/package.json
   ✓ backend/package.json
   ✓ frontend/src/utils/version.ts
   ✓ .version/changelog.json
   ✓ .version/VERSION_HISTORY.md
```

## 常见问题

### Q: 如何回滚版本？

A: 使用 `version:update` 命令指定旧版本号：

```bash
npm run version:update 1.15.7
```

### Q: 如何跳过某些步骤？

A: 直接使用模块命令：

```bash
# 只写入版本号，不验证
node unified-version-system/modules/version-writer.js 1.15.8

# 只验证，不写入
node unified-version-system/modules/version-validator.js 1.15.8
```

### Q: 如何检查硬编码问题？

A: 运行硬编码检查：

```bash
npm run version:check-hardcode
```

### Q: 版本更新失败怎么办？

A: 系统会自动检测并报告错误：

1. 查看错误信息
2. 根据提示修复问题
3. 重新运行命令

### Q: 如何自定义版本号格式？

A: 编辑 `unified-version-system/modules/version-validator.js` 中的验证规则。

## 模块独立使用

每个模块都可以独立使用：

### 验证模块

```bash
node unified-version-system/modules/version-validator.js 1.15.8
```

### 写入模块

```bash
node unified-version-system/modules/version-writer.js 1.15.8
```

### 递增模块

```bash
node unified-version-system/modules/version-incrementer.js 1.15.7 patch
```

### 历史模块

```bash
# 生成历史文档
node unified-version-system/modules/version-history.js generate

# 清理重复记录
node unified-version-system/modules/version-history.js clean

# 查看统计信息
node unified-version-system/modules/version-history.js stats
```

## 集成到 CI/CD

### GitHub Actions 示例

```yaml
name: Release

on:
  push:
    branches: [main]

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Node.js
        uses: actions/setup-node@v2
        with:
          node-version: '20'
      
      - name: Install dependencies
        run: npm install
      
      - name: Validate version system
        run: npm run version:validate
      
      - name: Bump version
        run: npm run version:patch
      
      - name: Build
        run: npm run build
      
      - name: Commit and tag
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          VERSION=$(cat VERSION)
          git add .
          git commit -m "chore: bump version to $VERSION"
          git tag "v$VERSION"
          git push && git push --tags
```

## 最佳实践

1. **更新前验证**
   ```bash
   npm run version:validate
   ```

2. **使用语义化版本**
   - Patch: 修复 bug
   - Minor: 新增功能（向后兼容）
   - Major: 破坏性变更

3. **更新后构建**
   ```bash
   npm run version:patch
   npm run build
   ```

4. **提交时包含版本号**
   ```bash
   git commit -m "chore: bump version to $(cat VERSION)"
   ```

5. **创建 Git 标签**
   ```bash
   git tag "v$(cat VERSION)"
   git push --tags
   ```

## 下一步

- 阅读 [架构文档](./ARCHITECTURE.md) 了解系统设计
- 查看 [API 文档](./API.md) 了解详细接口
- 参考 [示例](./examples/) 学习高级用法

## 获取帮助

如果遇到问题：

1. 运行 `npm run version:validate` 检查系统状态
2. 查看错误信息和建议
3. 阅读相关文档
4. 提交 Issue

---

**版本：** v1.15.7  
**更新时间：** 2025-12-30
