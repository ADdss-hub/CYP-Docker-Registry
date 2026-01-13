/**
 * Git 操作模块
 * 提供 Git 相关操作：提交、标签、推送
 * 
 * @module git-operations
 * @author CYP
 * @version v1.0.0
 */

const { execSync } = require('child_process');
const path = require('path');

class GitOperations {
  constructor(options = {}) {
    this.projectRoot = options.projectRoot || process.cwd();
    this.silent = options.silent || false;
  }

  /**
   * 执行 Git 命令
   * @param {string} command - Git 命令
   * @param {Object} options - 选项
   * @returns {string|null} 命令输出
   */
  exec(command, options = {}) {
    try {
      return execSync(`git ${command}`, {
        cwd: this.projectRoot,
        encoding: 'utf-8',
        stdio: options.silent || this.silent ? 'pipe' : 'inherit',
        ...options,
      });
    } catch (error) {
      if (!options.ignoreError) {
        throw error;
      }
      return null;
    }
  }

  /**
   * 检查是否是 Git 仓库
   * @returns {boolean}
   */
  isGitRepo() {
    try {
      this.exec('rev-parse --git-dir', { silent: true });
      return true;
    } catch {
      return false;
    }
  }

  /**
   * 获取当前分支名
   * @returns {string}
   */
  getCurrentBranch() {
    const branch = this.exec('rev-parse --abbrev-ref HEAD', { silent: true });
    return branch ? branch.trim() : 'main';
  }

  /**
   * 检查是否有未提交的更改
   * @returns {string[]} 未提交的文件列表
   */
  getUncommittedChanges() {
    const status = this.exec('status --porcelain', { silent: true });
    return status ? status.trim().split('\n').filter(Boolean) : [];
  }

  /**
   * 检查 tag 是否已存在
   * @param {string} tagName - 标签名
   * @returns {boolean}
   */
  tagExists(tagName) {
    try {
      this.exec(`rev-parse ${tagName}`, { silent: true });
      return true;
    } catch {
      return false;
    }
  }

  /**
   * 暂存所有更改
   * @returns {boolean}
   */
  stageAll() {
    try {
      this.exec('add -A');
      if (!this.silent) {
        console.log('   ✓ 已暂存所有更改');
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 暂存失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 提交更改
   * @param {string} message - 提交信息
   * @returns {boolean}
   */
  commit(message) {
    try {
      this.exec(`commit -m "${message}"`);
      if (!this.silent) {
        console.log(`   ✓ 已提交: ${message}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 提交失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 创建标签
   * @param {string} tagName - 标签名
   * @param {string} message - 标签信息（可选，用于注释标签）
   * @returns {boolean}
   */
  createTag(tagName, message = null) {
    try {
      if (this.tagExists(tagName)) {
        if (!this.silent) {
          console.log(`   ⚠️  标签 ${tagName} 已存在，跳过创建`);
        }
        return false;
      }

      if (message) {
        this.exec(`tag -a ${tagName} -m "${message}"`);
      } else {
        this.exec(`tag ${tagName}`);
      }
      
      if (!this.silent) {
        console.log(`   ✓ 已创建标签: ${tagName}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 创建标签失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 删除本地标签
   * @param {string} tagName - 标签名
   * @returns {boolean}
   */
  deleteTag(tagName) {
    try {
      this.exec(`tag -d ${tagName}`);
      if (!this.silent) {
        console.log(`   ✓ 已删除本地标签: ${tagName}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 删除标签失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 推送到远程
   * @param {string} remote - 远程名称
   * @param {string} branch - 分支名
   * @returns {boolean}
   */
  push(remote = 'origin', branch = null) {
    try {
      const targetBranch = branch || this.getCurrentBranch();
      this.exec(`push ${remote} ${targetBranch}`);
      if (!this.silent) {
        console.log(`   ✓ 已推送到 ${remote}/${targetBranch}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 推送失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 推送标签到远程
   * @param {string} tagName - 标签名
   * @param {string} remote - 远程名称
   * @returns {boolean}
   */
  pushTag(tagName, remote = 'origin') {
    try {
      this.exec(`push ${remote} ${tagName}`);
      if (!this.silent) {
        console.log(`   ✓ 已推送标签: ${tagName}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 推送标签失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 推送所有标签到远程
   * @param {string} remote - 远程名称
   * @returns {boolean}
   */
  pushAllTags(remote = 'origin') {
    try {
      this.exec(`push ${remote} --tags`);
      if (!this.silent) {
        console.log(`   ✓ 已推送所有标签到 ${remote}`);
      }
      return true;
    } catch (error) {
      if (!this.silent) {
        console.log(`   ✗ 推送标签失败: ${error.message}`);
      }
      return false;
    }
  }

  /**
   * 完整的发布流程：暂存 -> 提交 -> 创建标签 -> 推送
   * @param {string} version - 版本号
   * @param {Object} options - 选项
   * @returns {Object} 操作结果
   */
  release(version, options = {}) {
    const {
      commitMessage = `release: v${version}`,
      tagName = `v${version}`,
      tagMessage = null,
      remote = 'origin',
      branch = null,
      skipCommit = false,
      skipTag = false,
      skipPush = false,
    } = options;

    const result = {
      success: false,
      steps: {
        stage: false,
        commit: false,
        tag: false,
        pushCode: false,
        pushTag: false,
      },
      errors: [],
    };

    if (!this.silent) {
      console.log('\n📤 Git 操作...');
    }

    // 检查是否是 Git 仓库
    if (!this.isGitRepo()) {
      result.errors.push('当前目录不是 Git 仓库');
      return result;
    }

    try {
      // 1. 暂存更改
      result.steps.stage = this.stageAll();

      // 2. 提交
      if (!skipCommit) {
        result.steps.commit = this.commit(commitMessage);
      } else {
        if (!this.silent) console.log('   ⊘ 跳过提交');
      }

      // 3. 创建标签
      if (!skipTag) {
        result.steps.tag = this.createTag(tagName, tagMessage);
      } else {
        if (!this.silent) console.log('   ⊘ 跳过创建标签');
      }

      // 4. 推送代码
      if (!skipPush) {
        if (!this.silent) console.log('\n🌐 推送到远程...');
        result.steps.pushCode = this.push(remote, branch);

        // 5. 推送标签
        if (!skipTag && result.steps.tag) {
          result.steps.pushTag = this.pushTag(tagName, remote);
        }
      } else {
        if (!this.silent) console.log('   ⊘ 跳过推送');
      }

      result.success = true;
    } catch (error) {
      result.errors.push(error.message);
    }

    return result;
  }

  /**
   * 获取最近的标签
   * @returns {string|null}
   */
  getLatestTag() {
    try {
      const tag = this.exec('describe --tags --abbrev=0', { silent: true });
      return tag ? tag.trim() : null;
    } catch {
      return null;
    }
  }

  /**
   * 获取所有标签
   * @returns {string[]}
   */
  getAllTags() {
    try {
      const tags = this.exec('tag -l', { silent: true });
      return tags ? tags.trim().split('\n').filter(Boolean) : [];
    } catch {
      return [];
    }
  }
}

module.exports = GitOperations;
