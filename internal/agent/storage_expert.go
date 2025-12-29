package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/tool"
)

// StorageExpertAgent 专门处理存储相关问题的专家 Agent
// 集成了文件系统、块存储、分布式存储、数据库存储等专业知识
//
// 使用方法：
// 1. 检测到存储相关问题时，自动激活存储专家
// 2. 存储专家使用专用的 Prompt 模板和工具集
// 3. 可以与 ReAct 模式结合使用
type StorageExpertAgent struct {
	llmClient    llm.Client
	toolRegistry *tool.Registry
	sshPool      SSHExecutor
	maxLoops     int
	timeout      time.Duration
}

// SSHExecutor SSH 执行接口（兼容 ssh.Pool）
type SSHExecutor interface {
	Exec(host string, cmd string) (string, error)
}

// NewStorageExpertAgent 创建存储专家 Agent
func NewStorageExpertAgent(llmClient llm.Client, toolRegistry *tool.Registry, sshPool SSHExecutor) *StorageExpertAgent {
	return &StorageExpertAgent{
		llmClient:    llmClient,
		toolRegistry: toolRegistry,
		sshPool:      sshPool,
		maxLoops:     10,
		timeout:      5 * time.Minute,
	}
}

// StorageProblem 存储问题分类
type StorageProblem struct {
	Category    string   `json:"category"`    // filesystem, block, distributed, database, network
	SubType     string   `json:"sub_type"`    // ext4, xfs, btrfs, zfs, lvm, raid, nfs, iscsi, ceph, etc.
	Severity    string   `json:"severity"`    // critical, warning, info
	DataAtRisk  bool     `json:"data_at_risk"` // 是否有数据丢失风险
	ServiceDown bool     `json:"service_down"`
	Hosts       []string `json:"hosts"`
}

// classifyStorageProblem 分类存储问题
func (a *StorageExpertAgent) classifyStorageProblem(message string, hosts []string) StorageProblem {
	msg := strings.ToLower(message)

	problem := StorageProblem{
		Hosts:       hosts,
		Severity:    "info",
		DataAtRisk:  false,
		ServiceDown: false,
	}

	// 检测严重程度
	if strings.Contains(msg, "紧急") || strings.Contains(msg, "critical") || strings.Contains(msg, "down") {
		problem.Severity = "critical"
		problem.ServiceDown = true
	} else if strings.Contains(msg, "警告") || strings.Contains(msg, "warning") || strings.Contains(msg, "慢") {
		problem.Severity = "warning"
	}

	// 检测数据风险
	if strings.Contains(msg, "数据丢失") || strings.Contains(msg, "损坏") || strings.Contains(msg, "corrupt") {
		problem.DataAtRisk = true
		problem.Severity = "critical"
	}

	// 分类问题类型
	switch {
	case strings.Contains(msg, "nfs") || strings.Contains(msg, "smb") || strings.Contains(msg, "iscsi"):
		problem.Category = "network"
	case strings.Contains(msg, "ceph") || strings.Contains(msg, "minio") || strings.Contains(msg, "s3"):
		problem.Category = "distributed"
	case strings.Contains(msg, "mysql") || strings.Contains(msg, "postgres") || strings.Contains(msg, "redis") || strings.Contains(msg, "mongo"):
		problem.Category = "database"
	case strings.Contains(msg, "lvm") || strings.Contains(msg, "raid") || strings.Contains(msg, "mdadm"):
		problem.Category = "block"
	case strings.Contains(msg, "ext4") || strings.Contains(msg, "xfs") || strings.Contains(msg, "btrfs") || strings.Contains(msg, "zfs"):
		problem.Category = "filesystem"
		problem.SubType = extractFilesystemType(msg)
	case strings.Contains(msg, "磁盘") || strings.Contains(msg, "disk") || strings.Contains(msg, "空间") || strings.Contains(msg, "space"):
		problem.Category = "filesystem"
	case strings.Contains(msg, "io") || strings.Contains(msg, "性能") || strings.Contains(msg, "慢"):
		problem.Category = "performance"
	default:
		problem.Category = "general"
	}

	return problem
}

// extractFilesystemType 提取文件系统类型
func extractFilesystemType(msg string) string {
	types := []string{"ext4", "xfs", "btrfs", "zfs", "ext3", "ext2"}
	for _, t := range types {
		if strings.Contains(msg, t) {
			return t
		}
	}
	return "unknown"
}

// buildStorageExpertPrompt 构建存储专家专用 Prompt
func (a *StorageExpertAgent) buildStorageExpertPrompt(message string, problem StorageProblem, hosts []string) string {
	prompt := `# 存储系统专家

你是一名专业的存储系统故障排查专家，精通文件系统、块存储、分布式存储、数据库存储等领域的诊断和修复。

## 核心原则

**数据安全第一 (DATA PRESERVATION FIRST)**
- 在执行任何破坏性操作前，必须先执行只读诊断
- 永远不要在没有备份的情况下运行 fsck -y, wipefs, dd 等危险命令
- 在不确定时，只执行只读诊断命令

## 工具使用指南

根据问题类型选择合适的工具：

- check_filesystem: 检查文件系统状态（支持 ext4/XFS/Btrfs/ZFS）
- check_raid: 检查 RAID 阵列状态
- check_disk: 检查磁盘使用情况
- execute_command: 执行自定义诊断命令

## 问题分类和优先级

**关键 (Critical)** - 需要立即处理
- 服务完全中断
- 数据丢失风险
- 文件系统只读重挂载
- RAID 阵列降级

**警告 (Warning)** - 需要尽快处理
- 性能严重下降
- 磁盘空间不足
- I/O 等待时间过长
- 设备错误增加

## 诊断流程

1. 收集系统信息（df, mount, 内核日志）
2. 分析问题严重程度
3. 执行只读诊断命令
4. 提供修复建议

`

	// 添加问题上下文
	context := fmt.Sprintf(`
## 当前诊断上下文

问题描述: %s
问题分类: %s
严重程度: %s
数据风险: %v
服务中断: %v
目标主机: %v
`,
		message,
		problem.Category,
		problem.Severity,
		problem.DataAtRisk,
		problem.ServiceDown,
		hosts,
	)

	// 根据问题类型添加专门的诊断指导
	switch problem.Category {
	case "filesystem":
		context += a.getFilesystemGuidance(problem.SubType)
	case "block":
		context += a.getBlockStorageGuidance()
	case "network":
		context += a.getNetworkStorageGuidance()
	}

	return prompt + context
}

// getFilesystemGuidance 获取文件系统诊断指导
func (a *StorageExpertAgent) getFilesystemGuidance(fsType string) string {
	guidance := `
## 文件系统诊断流程

1. 检查文件系统类型和挂载状态
2. 查看内核日志中的文件系统错误
3. 检查磁盘健康状况（SMART）
4. 执行文件系统特定的检查命令

`

	switch fsType {
	case "ext4":
		guidance += `
### ext4 特定命令
- 检查超级块: tune2fs -l /dev/sdX1
- 只读检查: e2fsck -n /dev/sdX1
- 查看碎片: e4defrag -c /mount/point
`
	case "xfs":
		guidance += `
### XFS 特定命令
- 文件系统信息: xfs_info /mount/point
- 只读检查: xfs_repair -n /dev/sdX1
- 检查分配组: xfs_growfs -n /mount/point
`
	case "btrfs":
		guidance += `
### Btrfs 特定命令
- 设备统计: btrfs device stats /mount/point
- 空间使用: btrfs filesystem df /mount/point
- 子卷列表: btrfs subvolume list /mount/point
`
	case "zfs":
		guidance += `
### ZFS 特定命令
- 池状态: zpool status -v
- 数据集属性: zfs get all pool/dataset
`
	}

	return guidance
}

// getBlockStorageGuidance 获取块存储诊断指导
func (a *StorageExpertAgent) getBlockStorageGuidance() string {
	return `
## 块存储诊断流程

### LVM 诊断
1. 查看 PV/VG/LV 状态: pvs, vgs, lvs
2. 检查 thin pool 使用率: lvs -o +thin_data_percent

### RAID 诊断
1. 查看 RAID 状态: cat /proc/mdstat
2. RAID 详细信息: mdadm --detail /dev/md0
3. 检查成员磁盘: mdadm --examine /dev/sdX1
`
}

// getNetworkStorageGuidance 获取网络存储诊断指导
func (a *StorageExpertAgent) getNetworkStorageGuidance() string {
	return `
## 网络存储诊断流程

### NFS 诊断
1. 客户端统计: nfsstat -c
2. 查看挂载: mount | grep nfs
3. 测试连接: showmount -e server

### SMB/CIFS 诊断
1. 测试配置: testparm -v
2. 查看共享: smbclient -L //server
3. 检查服务: systemctl status smbd nmbd
`
}

// Execute 执行存储专家诊断
func (a *StorageExpertAgent) Execute(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 1. 分析问题类型
	problem := a.classifyStorageProblem(req.Message, req.Hosts)

	// 2. 构建专用 Prompt
	prompt := a.buildStorageExpertPrompt(req.Message, problem, req.Hosts)

	// 3. 构建消息列表
	messages := []llm.Message{
		llm.NewSystemMessage(prompt),
		llm.NewUserMessage(req.Message),
	}

	// 添加历史消息
	messages = append(messages, req.History...)

	// 4. 调用 LLM
	resp, err := a.llmClient.Chat(ctx, messages)
	if err != nil {
		return &ChatResponse{
			Error: fmt.Sprintf("LLM 调用失败: %v", err),
		}, nil
	}

	// 5. 返回响应
	return &ChatResponse{
		SessionID: req.SessionID,
		Reply:     resp.Message.Content,
	}, nil
}

// GetStoragePrompt 获取存储专家 Prompt（供外部使用）
func GetStoragePrompt(message string, hosts []string) string {
	agent := &StorageExpertAgent{}
	problem := agent.classifyStorageProblem(message, hosts)
	return agent.buildStorageExpertPrompt(message, problem, hosts)
}
