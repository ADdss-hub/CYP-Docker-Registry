# 版本管理系统命令参考

## NPM 脚本命令

### 版本更新命令

#### `npm run version:patch`
递增补丁版本号（x.y.z → x.y.z+1）

**示例：** 1.15.7 → 1.15.8

**使用场景：**
- 修复 bug
- 小的改进
- 文档更新

---

#### `npm run version:minor`
递增次版本号（x.y.z → x.y+1.0）

**示例：** 1.15.7 → 1.16.0

**使用场景：**
- 新增功能
- 向后兼容的更改
- 功能增强

---

#### `npm run version:major`
递增主版本号（x.y.z → x+1.0.0）

**示例：** 1.15.7 → 2.0.0

**使用场景：**
- 破坏性变更
- 重大重构
- API 不兼容更新

---

#### `npm run version:update <version>`
更新到指定版本号

**示例：**
```bash
npm run version:update 2.0.0
npm run version:update 1.16.5
```

**使用场景：**
- 回滚版本
- 跳过版本号
- 手动指定版本

---

### 版本信息命令

#### `npm run version:info`
显示当前版本信息和版本建议

**输出：**
```
📊 版本信息:

当前版本: 1.15.7

版本建议:
  Patch: 1.15.8
  Minor: 1.16.0
  Major: 2.0.0
```

---

#### `npm run version:validate`
验证版本系统的完整性

**检查项：**
- 版本号格式验证
- 硬编码检查
- 文件一致性验证

**输出：**
```
🔍 验证版本系统...

✅ 版本格式验证通过
✅ 硬编码检查通过

✅ 版本系统验证通过！
```

---

#### `npm run version:check-hardcode`
检查代码中的版本号硬编码问题

**检查文件：**
- `scripts/update-version.js`
- `scripts/version-manager.js`
- `unified-version-system/**/*.js`

**输出：**
```
🔍 开始检查版本号硬编码...

✅ 硬编码检查通过
```

---

### 版本历史命令

#### `npm run version:history`
生成版本历史 Markdown 文档

**生成文件：** `.version/VERSION_HISTORY.md`

**内容包括：**
- 所有版本的详细记录
- 变更内容分类
- 版本统计信息

**输出：**
```
✅ 版本历史记录已保存: .version/VERSION_HISTORY.md
```

---

#### `npm run version:history-stats`
显示版本统计信息

**输出：**
```
📊 版本统计信息:

总版本数: 58
主版本更新: 0 次
次版本更新: 9 次
补丁更新: 44 次
首个版本: v1.9.4
最新版本: v1.15.7
```

---

#### `npm run version:history-clean`
清理重复的历史记录

**功能：**
- 检测重复的版本记录
- 自动清理重复项
- 保留最新记录

**输出：**
```
✅ 已清理 2 条重复记录
```

---

## 直接模块命令

### 统一管理器

#### 更新版本
```bash
node unified-version-system/version-manager-unified.js update <version>
```

#### 递增版本
```bash
node unified-version-system/version-manager-unified.js increment <type>
# type: patch | minor | major
```

#### 查看信息
```bash
node unified-version-system/version-manager-unified.js info
```

#### 验证系统
```bash
node unified-version-system/version-manager-unified.js validate
```

---

### 验证模块

#### 验证版本格式
```bash
node unified-version-system/modules/version-validator.js <version>
```

**示例：**
```bash
node unified-version-system/modules/version-validator.js 1.15.8
```

---

### 写入模块

#### 写入版本号到所有文件
```bash
node unified-version-system/modules/version-writer.js <version>
```

**示例：**
```bash
node unified-version-system/modules/version-writer.js 1.15.8
```

**更新的文件：**
- `VERSION`
- `package.json`
- `frontend/package.json`
- `backend/package.json`
- `frontend/src/utils/version.ts`

---

### 递增模块

#### 计算递增后的版本号
```bash
node unified-version-system/modules/version-incrementer.js <version> <type>
```

**示例：**
```bash
node unified-version-system/modules/version-incrementer.js 1.15.7 patch
# 输出: 1.15.8

node unified-version-system/modules/version-incrementer.js 1.15.7 minor
# 输出: 1.16.0

node unified-version-system/modules/version-incrementer.js 1.15.7 major
# 输出: 2.0.0
```

---

### 历史模块

#### 生成历史文档
```bash
node unified-version-system/modules/version-history.js generate
```

#### 清理重复记录
```bash
node unified-version-system/modules/version-history.js clean
```

#### 显示统计信息
```bash
node unified-version-system/modules/version-history.js stats
```

---

## 命令组合示例

### 完整发布流程

```bash
# 1. 验证系统
npm run version:validate

# 2. 查看当前版本
npm run version:info

# 3. 递增版本
npm run version:patch

# 4. 查看历史统计
npm run version:history-stats

# 5. 构建项目
npm run build

# 6. 提交更改
git add .
git commit -m "chore: release v$(cat VERSION)"
git tag "v$(cat VERSION)"
git push && git push --tags
```

---

### 快速检查

```bash
# 一次性运行多个检查
npm run version:info && npm run version:validate && npm run version:history-stats
```

---

### 版本回滚

```bash
# 1. 查看历史
npm run version:history-stats

# 2. 回滚到指定版本
npm run version:update 1.15.6

# 3. 验证
npm run version:validate
```

---

### 清理和维护

```bash
# 1. 清理重复历史记录
npm run version:history-clean

# 2. 重新生成历史文档
npm run version:history

# 3. 检查硬编码
npm run version:check-hardcode
```

---

## 命令速查表

| 命令 | 功能 | 使用频率 |
|------|------|----------|
| `npm run version:patch` | 递增补丁版本 | ⭐⭐⭐⭐⭐ |
| `npm run version:minor` | 递增次版本 | ⭐⭐⭐⭐ |
| `npm run version:major` | 递增主版本 | ⭐⭐ |
| `npm run version:info` | 查看版本信息 | ⭐⭐⭐⭐⭐ |
| `npm run version:validate` | 验证系统 | ⭐⭐⭐⭐ |
| `npm run version:history` | 生成历史文档 | ⭐⭐⭐ |
| `npm run version:history-stats` | 查看统计 | ⭐⭐⭐ |
| `npm run version:history-clean` | 清理重复 | ⭐⭐ |
| `npm run version:check-hardcode` | 检查硬编码 | ⭐⭐⭐ |
| `npm run version:update` | 指定版本 | ⭐⭐ |

---

## 错误处理

### 版本格式错误

**错误信息：**
```
❌ 版本格式无效: 1.15.x
```

**解决方法：**
使用正确的语义化版本格式（x.y.z）

---

### 硬编码检测失败

**错误信息：**
```
❌ 发现硬编码版本号
```

**解决方法：**
1. 运行 `npm run version:check-hardcode` 查看详情
2. 修复硬编码问题
3. 重新运行版本更新命令

---

### 版本冲突

**警告信息：**
```
⚠️ 新版本 1.15.6 不大于当前版本 1.15.7
```

**解决方法：**
- 使用更大的版本号
- 或使用 `version:update` 强制更新

---

## 最佳实践

1. **更新前先验证**
   ```bash
   npm run version:validate
   ```

2. **定期查看统计**
   ```bash
   npm run version:history-stats
   ```

3. **保持历史清洁**
   ```bash
   npm run version:history-clean
   ```

4. **遵循语义化版本**
   - Patch: 修复
   - Minor: 功能
   - Major: 破坏性变更

---

**文档版本：** v1.0.0  
**最后更新：** 2025-12-30
