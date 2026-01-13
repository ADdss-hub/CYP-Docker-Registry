#!/usr/bin/env node
/**
 * 版本同步脚本
 * 读取 VERSION 文件并同步到所有需要版本号的文件
 * 
 * @author CYP
 */

const fs = require('fs');
const path = require('path');

// 读取当前版本
const versionFile = path.join(__dirname, '..', 'VERSION');
const version = fs.readFileSync(versionFile, 'utf8').trim();

console.log(`📦 同步版本号: ${version}`);

// 需要同步的文件配置
const syncFiles = [
  {
    file: 'web/package.json',
    type: 'json',
    key: 'version'
  },
  {
    file: '设计文档.md',
    type: 'regex',
    patterns: [
      { search: /\*\*版本\*\*: v[\d.]+/g, replace: `**版本**: v${version}` },
      { search: /version: "v[\d.]+"/g, replace: `version: "v${version}"` },
      { search: /\*\*文档版本\*\*: v[\d.]+/g, replace: `**文档版本**: v${version}` }
    ]
  },
  {
    file: 'docs/SECURITY.md',
    type: 'regex',
    patterns: [
      { search: /\*\*版本\*\*: v[\d.]+/g, replace: `**版本**: v${version}` }
    ]
  },
  {
    file: 'docs/DEPLOY.md',
    type: 'regex',
    patterns: [
      { search: /\*\*版本\*\*: v[\d.]+/g, replace: `**版本**: v${version}` }
    ]
  },
  {
    file: 'README.md',
    type: 'regex',
    patterns: [
      { search: /\*\*版本\*\*: v[\d.]+/g, replace: `**版本**: v${version}` }
    ]
  }
];

let updated = 0;

syncFiles.forEach(config => {
  const filePath = path.join(__dirname, '..', config.file);
  
  if (!fs.existsSync(filePath)) {
    console.log(`  ⚠️  跳过 ${config.file} (文件不存在)`);
    return;
  }

  try {
    if (config.type === 'json') {
      const json = JSON.parse(fs.readFileSync(filePath, 'utf8'));
      if (json[config.key] !== version) {
        json[config.key] = version;
        fs.writeFileSync(filePath, JSON.stringify(json, null, 2) + '\n');
        console.log(`  ✅ ${config.file}`);
        updated++;
      } else {
        console.log(`  ⏭️  ${config.file} (已是最新)`);
      }
    } else if (config.type === 'regex') {
      let content = fs.readFileSync(filePath, 'utf8');
      let changed = false;
      
      config.patterns.forEach(pattern => {
        if (pattern.search.test(content)) {
          content = content.replace(pattern.search, pattern.replace);
          changed = true;
        }
      });
      
      if (changed) {
        fs.writeFileSync(filePath, content);
        console.log(`  ✅ ${config.file}`);
        updated++;
      } else {
        console.log(`  ⏭️  ${config.file} (已是最新)`);
      }
    }
  } catch (err) {
    console.log(`  ❌ ${config.file}: ${err.message}`);
  }
});

console.log(`\n✨ 完成! 更新了 ${updated} 个文件`);
