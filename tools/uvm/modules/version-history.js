#!/usr/bin/env node

/**
 * 版本历史记录模块
 * 负责管理和生成版本历史记录
 * 
 * @module version-history
 * @author CYP
 * @version v1.15.7
 */

const fs = require('fs');
const path = require('path');

class VersionHistory {
  constructor(options = {}) {
    this.projectRoot = options.projectRoot || process.cwd();
    this.silent = options.silent || false;
    this.historyFile = path.join(this.projectRoot, '.version/changelog.json');
    this.historyMdFile = path.join(this.projectRoot, '.version/VERSION_HISTORY.md');
  }

  /**
   * 读取历史记录
   * @returns {Array} 历史记录列表
   */
  readHistory() {
    if (!fs.existsSync(this.historyFile)) {
      return [];
    }

    try {
      const data = JSON.parse(fs.readFileSync(this.historyFile, 'utf8'));
      return data.history || [];
    } catch (error) {
      if (!this.silent) {
        console.warn(`⚠️  读取历史记录失败: ${error.message}`);
      }
      return [];
    }
  }

  /**
   * 添加历史记录
   * @param {Object} record - 版本记录
   */
  addRecord(record) {
    const history = this.readHistory();
    
    // 检查是否已存在相同版本
    const existingIndex = history.findIndex(h => h.version === record.version);
    
    if (existingIndex >= 0) {
      // 更新现有记录
      history[existingIndex] = {
        ...history[existingIndex],
        ...record,
        timestamp: new Date().toISOString()
      };
    } else {
      // 添加新记录
      history.unshift({
        version: record.version,
        timestamp: new Date().toISOString(),
        author: record.author || 'Unknown',
        changes: record.changes || [],
        type: record.type || 'patch',
        previousVersion: record.previousVersion || null,
        metadata: record.metadata || {}
      });
    }

    // 保存历史记录
    this.saveHistory(history);
  }

  /**
   * 保存历史记录
   * @param {Array} history - 历史记录列表
   */
  saveHistory(history) {
    const versionDir = path.dirname(this.historyFile);
    
    if (!fs.existsSync(versionDir)) {
      fs.mkdirSync(versionDir, { recursive: true });
    }

    const data = {
      schema: '2.1.0',
      history: history,
      metadata: {
        updated: new Date().toISOString(),
        project: 'CYP-memo'
      }
    };

    fs.writeFileSync(this.historyFile, JSON.stringify(data, null, 2));
  }

  /**
   * 生成 Markdown 格式的历史记录
   * @returns {string} Markdown 内容
   */
  generateMarkdown() {
    const history = this.readHistory();
    
    if (history.length === 0) {
      return '# 版本历史记录\n\n暂无版本记录。\n';
    }

    let markdown = '# 版本历史记录\n\n';
    markdown += `> 自动生成于 ${new Date().toLocaleString('zh-CN')}\n\n`;
    markdown += `**总版本数：** ${history.length}\n\n`;
    markdown += '---\n\n';

    // 按版本分组
    history.forEach((record, index) => {
      markdown += `## ${record.version}\n\n`;
      markdown += `**发布时间：** ${new Date(record.timestamp).toLocaleString('zh-CN')}\n\n`;
      markdown += `**作者：** ${record.author}\n\n`;
      
      if (record.previousVersion) {
        markdown += `**上一版本：** ${record.previousVersion}\n\n`;
      }

      markdown += `**变更类型：** ${this.getTypeLabel(record.type)}\n\n`;

      // 变更列表
      if (record.changes && record.changes.length > 0) {
        markdown += '### 变更内容\n\n';
        
        // 按类型分组
        const changesByType = this.groupChangesByType(record.changes);
        
        Object.keys(changesByType).forEach(type => {
          const typeLabel = this.getChangeTypeLabel(type);
          markdown += `#### ${typeLabel}\n\n`;
          
          changesByType[type].forEach(change => {
            markdown += `- ${change.description}\n`;
          });
          
          markdown += '\n';
        });
      } else {
        markdown += '### 变更内容\n\n';
        markdown += '- 无详细变更记录\n\n';
      }

      markdown += '---\n\n';
    });

    // 添加统计信息
    markdown += '## 统计信息\n\n';
    
    const stats = this.calculateStats(history);
    markdown += `- **总版本数：** ${stats.total}\n`;
    markdown += `- **主版本更新：** ${stats.major} 次\n`;
    markdown += `- **次版本更新：** ${stats.minor} 次\n`;
    markdown += `- **补丁更新：** ${stats.patch} 次\n`;
    markdown += `- **首个版本：** ${stats.firstVersion}\n`;
    markdown += `- **最新版本：** ${stats.latestVersion}\n`;
    markdown += `- **首次发布：** ${stats.firstDate}\n`;
    markdown += `- **最后更新：** ${stats.lastDate}\n`;

    return markdown;
  }

  /**
   * 按类型分组变更
   * @param {Array} changes - 变更列表
   * @returns {Object} 分组后的变更
   */
  groupChangesByType(changes) {
    const grouped = {};
    
    changes.forEach(change => {
      const type = change.type || 'other';
      if (!grouped[type]) {
        grouped[type] = [];
      }
      grouped[type].push(change);
    });

    return grouped;
  }

  /**
   * 获取类型标签
   * @param {string} type - 类型
   * @returns {string} 标签
   */
  getTypeLabel(type) {
    const labels = {
      'major': '🚀 重大更新',
      'minor': '✨ 功能更新',
      'patch': '🐛 问题修复',
      'chore': '🔧 日常维护',
      'docs': '📝 文档更新',
      'style': '💄 样式调整',
      'refactor': '♻️ 代码重构',
      'perf': '⚡ 性能优化',
      'test': '✅ 测试相关'
    };
    
    return labels[type] || '📦 其他更新';
  }

  /**
   * 获取变更类型标签
   * @param {string} type - 变更类型
   * @returns {string} 标签
   */
  getChangeTypeLabel(type) {
    const labels = {
      'feat': '✨ 新增功能',
      'fix': '🐛 问题修复',
      'docs': '📝 文档',
      'style': '💄 样式',
      'refactor': '♻️ 重构',
      'perf': '⚡ 性能',
      'test': '✅ 测试',
      'chore': '🔧 其他',
      'build': '📦 构建',
      'ci': '👷 CI/CD'
    };
    
    return labels[type] || '📌 其他';
  }

  /**
   * 计算统计信息
   * @param {Array} history - 历史记录
   * @returns {Object} 统计信息
   */
  calculateStats(history) {
    if (history.length === 0) {
      return {
        total: 0,
        major: 0,
        minor: 0,
        patch: 0,
        firstVersion: 'N/A',
        latestVersion: 'N/A',
        firstDate: 'N/A',
        lastDate: 'N/A'
      };
    }

    const stats = {
      total: history.length,
      major: 0,
      minor: 0,
      patch: 0
    };

    // 统计各类型更新次数
    history.forEach(record => {
      if (record.type === 'major') stats.major++;
      else if (record.type === 'minor') stats.minor++;
      else if (record.type === 'patch') stats.patch++;
    });

    // 获取首个和最新版本
    const sortedHistory = [...history].sort((a, b) => 
      new Date(a.timestamp) - new Date(b.timestamp)
    );

    stats.firstVersion = sortedHistory[0].version;
    stats.latestVersion = sortedHistory[sortedHistory.length - 1].version;
    stats.firstDate = new Date(sortedHistory[0].timestamp).toLocaleString('zh-CN');
    stats.lastDate = new Date(sortedHistory[sortedHistory.length - 1].timestamp).toLocaleString('zh-CN');

    return stats;
  }

  /**
   * 保存 Markdown 文件
   */
  saveMarkdown() {
    const markdown = this.generateMarkdown();
    
    const versionDir = path.dirname(this.historyMdFile);
    if (!fs.existsSync(versionDir)) {
      fs.mkdirSync(versionDir, { recursive: true });
    }

    fs.writeFileSync(this.historyMdFile, markdown);

    if (!this.silent) {
      console.log(`✅ 版本历史记录已保存: ${this.historyMdFile}`);
    }
  }

  /**
   * 清理重复记录
   */
  cleanDuplicates() {
    const history = this.readHistory();
    const seen = new Set();
    const cleaned = [];

    history.forEach(record => {
      const key = `${record.version}-${record.timestamp}`;
      if (!seen.has(key)) {
        seen.add(key);
        cleaned.push(record);
      }
    });

    if (cleaned.length < history.length) {
      this.saveHistory(cleaned);
      
      if (!this.silent) {
        console.log(`✅ 已清理 ${history.length - cleaned.length} 条重复记录`);
      }
    }
  }

  /**
   * 获取版本统计
   * @returns {Object} 统计信息
   */
  getStats() {
    const history = this.readHistory();
    return this.calculateStats(history);
  }
}

module.exports = VersionHistory;

// CLI 支持
if (require.main === module) {
  const command = process.argv[2];
  const history = new VersionHistory();

  switch (command) {
    case 'generate':
      history.saveMarkdown();
      console.log('\n✅ 版本历史记录已生成！\n');
      break;

    case 'clean':
      history.cleanDuplicates();
      console.log('\n✅ 重复记录已清理！\n');
      break;

    case 'stats':
      const stats = history.getStats();
      console.log('\n📊 版本统计信息:\n');
      console.log(`总版本数: ${stats.total}`);
      console.log(`主版本更新: ${stats.major} 次`);
      console.log(`次版本更新: ${stats.minor} 次`);
      console.log(`补丁更新: ${stats.patch} 次`);
      console.log(`首个版本: ${stats.firstVersion}`);
      console.log(`最新版本: ${stats.latestVersion}`);
      console.log('');
      break;

    default:
      console.log('版本历史记录模块\n');
      console.log('用法:');
      console.log('  node version-history.js generate  - 生成 Markdown 历史记录');
      console.log('  node version-history.js clean     - 清理重复记录');
      console.log('  node version-history.js stats     - 显示统计信息');
      console.log('');
  }
}
