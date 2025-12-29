// 危险命令检测工具

// 危险命令关键词列表
const DANGEROUS_PATTERNS = [
  // 删除命令
  /\brm\s+(-rf|-r\s+-f|-fr)\s+/i,
  /\brm\s+-rf\s+/i,
  /\brm\s+.*\/\s*$/i, // rm ... /
  
  // 重启/关机命令
  /\breboot\b/i,
  /\bshutdown\b/i,
  /\bhalt\b/i,
  /\bpoweroff\b/i,
  /\binit\s+[06]\b/i,
  
  // 服务停止命令
  /\bsystemctl\s+stop\s+/i,
  /\bservice\s+.*\s+stop\s+/i,
  /\bkillall\s+/i,
  /\bkill\s+-9\s+/i,
  /\bpkill\s+-9\s+/i,
  
  // 格式化/文件系统操作
  /\bmkfs\s+/i,
  /\bfdisk\s+/i,
  /\bdd\s+if=/i,
  
  // 网络操作
  /\broute\s+del\s+/i,
  /\bip\s+route\s+del\s+/i,
  
  // 权限修改
  /\bchmod\s+000\s+/i,
  /\bchmod\s+-R\s+000\s+/i,
  
  // 数据库操作
  /\bdrop\s+database\s+/i,
  /\btruncate\s+table\s+/i,
  /\bdelete\s+from\s+.*\s+where\s+1\s*=\s*1/i,
]

// 高危险命令（需要强制确认）
const CRITICAL_PATTERNS = [
  /\brm\s+(-rf|-r\s+-f|-fr)\s+/i,
  /\brm\s+-rf\s+/i,
  /\breboot\b/i,
  /\bshutdown\s+(-h\s+now|now)\b/i,
  /\bhalt\b/i,
  /\bpoweroff\b/i,
  /\bmkfs\s+/i,
  /\bdd\s+if=/i,
]

/**
 * 检测命令是否危险
 * @param command 命令字符串
 * @returns 是否危险
 */
export function isDangerousCommand(command: string): boolean {
  if (!command || !command.trim()) {
    return false
  }
  
  const trimmed = command.trim()
  return DANGEROUS_PATTERNS.some(pattern => pattern.test(trimmed))
}

/**
 * 检测命令是否是高危险命令（需要强制确认）
 * @param command 命令字符串
 * @returns 是否高危险
 */
export function isCriticalCommand(command: string): boolean {
  if (!command || !command.trim()) {
    return false
  }
  
  const trimmed = command.trim()
  return CRITICAL_PATTERNS.some(pattern => pattern.test(trimmed))
}

/**
 * 获取危险命令的警告信息
 * @param command 命令字符串
 * @returns 警告信息
 */
export function getDangerousCommandWarning(command: string): string {
  if (isCriticalCommand(command)) {
    return '这是一个高危险命令，可能会造成不可逆的数据丢失或系统故障！'
  }
  
  if (isDangerousCommand(command)) {
    return '这是一个危险命令，请确认操作是否正确！'
  }
  
  return ''
}

/**
 * 高亮危险关键词（返回HTML字符串）
 * @param command 命令字符串
 * @returns 高亮后的HTML字符串
 */
export function highlightDangerousKeywords(command: string): string {
  if (!command) {
    return ''
  }
  
  let highlighted = command
  const dangerKeywords = ['rm -rf', 'reboot', 'shutdown', 'kill -9', 'mkfs', 'dd if=']
  
  dangerKeywords.forEach(keyword => {
    const regex = new RegExp(`(${keyword.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi')
    highlighted = highlighted.replace(regex, '<span style="color: #f56c6c; font-weight: bold;">$1</span>')
  })
  
  return highlighted
}



