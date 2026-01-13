# 版本记录格式规范

## 概述

版本管理系统使用两种格式来记录和展示版本信息：
- **JSON 格式** - 机器可读，用于程序引入和更新
- **Markdown 格式** - 人类可读，用于文档记录和展示

---

## JSON 格式 (.version/changelog.json)

### 主要用途

1. **版本号的引入和更新**
   - 程序读取当前版本信息
   - 自动化脚本获取版本数据
   - 版本比较和验证

2. **版本变更记录的引入和更新**
   - 记录每个版本的详细变更
   - 存储版本元数据
   - 支持程序化查询和分析

3. **数据交换和集成**
   - 与其他工具集成
   - API 数据源
   - 自动化工作流数据

### 文件结构

```json
{
  "schema": "2.1.0",
  "history": [
    {
      "version": "v1.15.7",
      "timestamp": "2025-12-30T13:31:14.344Z",
      "author": "CYP",
      "changes": [
        {
          "type": "fix",
          "description": "修复版本号显示问题"
        },
        {
          "type": "feat",
          "description": "新增功能描述"
        }
      ],
      "type": "patch",
      "previousVersion": "v1.15.6",
      "metadata": {
        "previousVersion": "v1.15.6",
        "updated": true,
        "schema": "2.1.0"
      }
    }
  ],
  "metadata": {
    "updated": "2025-12-30T13:31:14.344Z",
    "project": "CYP-memo"
  }
}
```

### 字段说明

#### 根级别字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `schema` | string | 是 | 数据格式版本号 |
| `history` | array | 是 | 版本历史记录数组 |
| `metadata` | object | 是 | 项目元数据 |

#### history 数组元素字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `version` | string | 是 | 版本号（带 v 前缀） |
| `timestamp` | string | 是 | ISO 8601 格式时间戳 |
| `author` | string | 是 | 版本作者 |
| `changes` | array | 否 | 变更列表 |
| `type` | string | 是 | 版本类型（major/minor/patch/chore） |
| `previousVersion` | string | 否 | 上一个版本号 |
| `metadata` | object | 否 | 版本元数据 |

#### changes 数组元素字段

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `type` | string | 是 | 变更类型（feat/fix/docs/style/refactor/perf/test/chore/build/ci） |
| `description` | string | 是 | 变更描述 |
| `scope` | string | 否 | 变更范围 |
| `breaking` | boolean | 否 | 是否为破坏性变更 |

### 变更类型定义

| 类型 | 说明 | 示例 |
|------|------|------|
| `feat` | 新增功能 | 添加用户管理模块 |
| `fix` | 问题修复 | 修复登录失败问题 |
| `docs` | 文档更新 | 更新 API 文档 |
| `style` | 样式调整 | 优化界面布局 |
| `refactor` | 代码重构 | 重构版本管理系统 |
| `perf` | 性能优化 | 优化数据库查询 |
| `test` | 测试相关 | 添加单元测试 |
| `chore` | 日常维护 | 更新依赖包 |
| `build` | 构建相关 | 优化构建脚本 |
| `ci` | CI/CD | 更新 GitHub Actions |

### 版本类型定义

| 类型 | 说明 | 版本变化 |
|------|------|----------|
| `major` | 主版本更新 | 1.0.0 → 2.0.0 |
| `minor` | 次版本更新 | 1.0.0 → 1.1.0 |
| `patch` | 补丁更新 | 1.0.0 → 1.0.1 |
| `chore` | 日常维护 | 无版本变化或特殊情况 |

### 使用场景

#### 1. 程序读取版本信息

```javascript
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('.version/changelog.json', 'utf8'));

// 获取最新版本
const latestVersion = data.history[0].version;

// 获取版本变更
const changes = data.history[0].changes;
```

#### 2. 添加新版本记录

```javascript
const VersionHistory = require('./modules/version-history');
const history = new VersionHistory();

history.addRecord({
  version: 'v1.15.8',
  author: 'CYP',
  changes: [
    { type: 'fix', description: '修复 bug' }
  ],
  type: 'patch',
  previousVersion: 'v1.15.7'
});
```

#### 3. 查询版本历史

```javascript
const history = new VersionHistory();
const records = history.readHistory();

// 查找特定版本
const v1_15_7 = records.find(r => r.version === 'v1.15.7');

// 统计版本类型
const patchCount = records.filter(r => r.type === 'patch').length;
```

---

## Markdown 格式 (.version/VERSION_HISTORY.md)

### 主要用途

1. **项目版本历史文档记录**
   - 人类可读的版本历史
   - 项目文档的一部分
   - 便于查看和分享

2. **版本变更展示**
   - 清晰的版本时间线
   - 分类展示变更内容
   - 统计信息可视化

3. **团队协作和沟通**
   - 团队成员了解项目演进
   - 向用户展示更新内容
   - 生成 Release Notes

### 文件结构

```markdown
# 版本历史记录

> 自动生成于 2025/12/30 21:49:22

**总版本数：** 58

---

## v1.15.7

**发布时间：** 2025/12/30 21:31:14

**作者：** CYP

**上一版本：** v1.15.6

**变更类型：** 🐛 问题修复

### 变更内容

#### 🐛 问题修复

- 修复版本号显示问题

#### ✨ 新增功能

- 添加用户管理功能

---

## 统计信息

- **总版本数：** 58
- **主版本更新：** 0 次
- **次版本更新：** 9 次
- **补丁更新：** 44 次
- **首个版本：** v1.9.4
- **最新版本：** v1.15.7
- **首次发布：** 2025/12/28 15:20:25
- **最后更新：** 2025/12/30 21:31:14
```

### 格式规范

#### 1. 文档头部

```markdown
# 版本历史记录

> 自动生成于 [生成时间]

**总版本数：** [数量]

---
```

#### 2. 版本记录

```markdown
## [版本号]

**发布时间：** [时间]

**作者：** [作者名]

**上一版本：** [上一版本号]

**变更类型：** [类型图标] [类型名称]

### 变更内容

#### [变更类型图标] [变更类型名称]

- [变更描述 1]
- [变更描述 2]

---
```

#### 3. 统计信息

```markdown
## 统计信息

- **总版本数：** [数量]
- **主版本更新：** [数量] 次
- **次版本更新：** [数量] 次
- **补丁更新：** [数量] 次
- **首个版本：** [版本号]
- **最新版本：** [版本号]
- **首次发布：** [时间]
- **最后更新：** [时间]
```

### 图标映射

#### 版本类型图标

| 类型 | 图标 | 名称 |
|------|------|------|
| `major` | 🚀 | 重大更新 |
| `minor` | ✨ | 功能更新 |
| `patch` | 🐛 | 问题修复 |
| `chore` | 🔧 | 日常维护 |
| `docs` | 📝 | 文档更新 |
| `style` | 💄 | 样式调整 |
| `refactor` | ♻️ | 代码重构 |
| `perf` | ⚡ | 性能优化 |
| `test` | ✅ | 测试相关 |

#### 变更类型图标

| 类型 | 图标 | 名称 |
|------|------|------|
| `feat` | ✨ | 新增功能 |
| `fix` | 🐛 | 问题修复 |
| `docs` | 📝 | 文档 |
| `style` | 💄 | 样式 |
| `refactor` | ♻️ | 重构 |
| `perf` | ⚡ | 性能 |
| `test` | ✅ | 测试 |
| `chore` | 🔧 | 其他 |
| `build` | 📦 | 构建 |
| `ci` | 👷 | CI/CD |

### 使用场景

#### 1. 查看版本历史

直接打开 `.version/VERSION_HISTORY.md` 文件查看

#### 2. 生成文档

```bash
npm run version:history
```

#### 3. 集成到项目文档

可以将内容复制到：
- README.md
- CHANGELOG.md
- 项目文档网站
- Release Notes

#### 4. 分享给团队

- 通过 Git 提交共享
- 导出为 PDF
- 发布到文档平台

---

## 两种格式的关系

### 数据流向

```
版本更新
    ↓
写入 JSON (changelog.json)
    ↓
读取 JSON 数据
    ↓
生成 Markdown (VERSION_HISTORY.md)
```

### 同步机制

1. **自动同步**
   - 每次版本更新时自动更新 JSON
   - 自动生成 Markdown 文档

2. **手动同步**
   ```bash
   # 重新生成 Markdown
   npm run version:history
   ```

3. **数据一致性**
   - JSON 是数据源（Single Source of Truth）
   - Markdown 是 JSON 的可视化展示
   - 始终从 JSON 生成 Markdown

### 优先级

- **JSON 优先** - 所有数据写入 JSON
- **Markdown 只读** - 不直接编辑 Markdown
- **自动生成** - Markdown 由 JSON 自动生成

---

## 最佳实践

### JSON 格式

1. **保持结构化**
   - 使用标准字段
   - 遵循 schema 版本
   - 保持数据完整性

2. **详细记录变更**
   ```json
   {
     "changes": [
       {
         "type": "feat",
         "description": "添加用户管理功能",
         "scope": "user"
       },
       {
         "type": "fix",
         "description": "修复登录失败问题",
         "scope": "auth"
       }
     ]
   }
   ```

3. **使用元数据**
   ```json
   {
     "metadata": {
       "milestone": "v1.16",
       "branch": "main",
       "commit": "abc123"
     }
   }
   ```

### Markdown 格式

1. **定期生成**
   ```bash
   npm run version:history
   ```

2. **保持清洁**
   ```bash
   # 清理重复记录
   npm run version:history-clean
   ```

3. **版本控制**
   - 将 Markdown 提交到 Git
   - 作为项目文档的一部分
   - 便于查看历史变化

---

## 工具支持

### 读取 JSON

```javascript
const VersionHistory = require('./modules/version-history');
const history = new VersionHistory();

// 读取所有历史
const records = history.readHistory();

// 获取统计
const stats = history.getStats();
```

### 生成 Markdown

```javascript
const history = new VersionHistory();

// 生成并保存
history.saveMarkdown();

// 只生成内容
const markdown = history.generateMarkdown();
```

### CLI 命令

```bash
# 生成 Markdown
npm run version:history

# 查看统计
npm run version:history-stats

# 清理重复
npm run version:history-clean
```

---

## 扩展性

### 自定义 JSON 字段

可以在 `metadata` 中添加自定义字段：

```json
{
  "metadata": {
    "customField": "value",
    "tags": ["feature", "important"],
    "reviewers": ["user1", "user2"]
  }
}
```

### 自定义 Markdown 格式

修改 `version-history.js` 中的 `generateMarkdown()` 方法来自定义输出格式。

---

## 总结

| 特性 | JSON 格式 | Markdown 格式 |
|------|-----------|---------------|
| **主要用途** | 程序引入和更新 | 文档记录和展示 |
| **可读性** | 机器可读 | 人类可读 |
| **编辑方式** | 程序自动更新 | 自动生成，不手动编辑 |
| **数据完整性** | 完整的结构化数据 | 格式化的展示数据 |
| **使用场景** | API、自动化、集成 | 文档、分享、查看 |
| **优先级** | 数据源（高） | 展示层（低） |

**核心原则：** JSON 是唯一数据源，Markdown 是自动生成的展示文档。

---

**文档版本：** v1.0.0  
**最后更新：** 2025-12-30
