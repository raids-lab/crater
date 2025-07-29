// scripts/install-git-hooks.mjs
// 使用 .mjs 扩展名以支持 ES 模块语法

import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { readFileSync, writeFileSync, chmodSync, existsSync, mkdirSync } from 'fs';
import { execSync } from 'child_process';

// 获取当前脚本的目录
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// 源钩子脚本的路径
const PRE_COMMIT_SCRIPT_SOURCE = join(__dirname, 'pre-commit');
// 目标钩子的文件名
const PRE_COMMIT_HOOK_DEST_NAME = 'pre-commit';

console.log("Attempting to install Git pre-commit hook...");

try {
  // 查找 Git 仓库的根目录
  // execSync 会返回命令的 stdout，如果不是 Git 仓库会抛出错误
  const gitRoot = execSync('git rev-parse --show-toplevel', { encoding: 'utf8', stdio: 'pipe' }).trim();
  const gitHooksDir = join(gitRoot, '.git', 'hooks');

  // 确保 .git/hooks 目录存在
  if (!existsSync(gitHooksDir)) {
    mkdirSync(gitHooksDir, { recursive: true });
    console.log(`Created Git hooks directory: ${gitHooksDir}`);
  }

  // 目标钩子文件的完整路径
  const destPath = join(gitHooksDir, PRE_COMMIT_HOOK_DEST_NAME);

  // 读取源脚本的内容
  const scriptContent = readFileSync(PRE_COMMIT_SCRIPT_SOURCE, 'utf8');

  // 将内容写入目标钩子文件
  writeFileSync(destPath, scriptContent);
  console.log(`Copied '${PRE_COMMIT_SCRIPT_SOURCE}' to '${destPath}'`);

  console.log("Git pre-commit hook installed successfully.");

} catch (error) {
  // 捕获 execSync 抛出的错误，特别是当不在 Git 仓库时
  if (error.message.includes('not a git repository') || error.message.includes('fatal: not a git repository')) {
    console.warn("Warning: Not in a Git repository. Skipping pre-commit hook installation.");
  } else {
    // 处理其他可能的错误
    console.error("Error installing Git pre-commit hook:", error.message);
    process.exit(1); // 非零退出码表示失败
  }
}