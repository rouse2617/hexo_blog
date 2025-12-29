package builtin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-ops/internal/tool"
)

// StorageCheckTool 存储系统检查工具
type StorageCheckTool struct {
	name        string
	description string
}

// NewStorageCheckTool 创建存储检查工具
func NewStorageCheckTool() *StorageCheckTool {
	return &StorageCheckTool{
		name:        "check_filesystem",
		description: "检查文件系统状态和健康情况，支持 ext4/XFS/Btrfs/ZFS 等多种文件系统",
	}
}

// Name 返回工具名称
func (t *StorageCheckTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *StorageCheckTool) Description() string {
	return t.description
}

// Parameters 返回工具参数
func (t *StorageCheckTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "主机名称",
			Required:    true,
		},
		{
			Name:        "mount",
			Type:        "string",
			Description: "挂载点路径，如 / /home /var，为空则检查所有",
			Required:    false,
		},
		{
			Name:        "device",
			Type:        "string",
			Description: "设备路径，如 /dev/sda1 /dev/mapper/vg0-lv",
			Required:    false,
		},
		{
			Name:        "detail",
			Type:        "boolean",
			Description: "是否显示详细信息",
			Required:    false,
		},
	}
}

// Execute 执行文件系统检查
func (t *StorageCheckTool) Execute(ctx *tool.Context, params map[string]any) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	mount := tool.GetStringParam(params, "mount", "")
	device := tool.GetStringParam(params, "device", "")
	detail := tool.GetBoolParam(params, "detail", false)

	if host == "" {
		return tool.NewErrorResult("host 参数必填"), nil
	}

	// 如果没有指定挂载点或设备，检查所有挂载的文件系统
	if mount == "" && device == "" {
		return t.checkAllFilesystems(ctx, host, detail)
	}

	// 检查指定的文件系统
	return t.checkSpecificFilesystem(ctx, host, mount, device, detail)
}

// checkAllFilesystems 检查所有挂载的文件系统
func (t *StorageCheckTool) checkAllFilesystems(ctx *tool.Context, host string, detail bool) (*tool.Result, error) {
	commands := map[string]string{
		"df":        "df -hT",
		"mount":     "mount | grep -E '(ext|xfs|btrfs|zfs)'",
		"findmnt":   "findmnt -t ext4,xfs,btrfs,zfs -o SOURCE,TARGET,FSTYPE,AVAIL,USE%",
	}

	results := make(map[string]string)
	for name, cmd := range commands {
		output, err := ctx.SSH.Exec(host, cmd)
		if err != nil {
			results[name] = fmt.Sprintf("执行失败: %v", err)
		} else {
			results[name] = output
		}
	}

	// 获取内核日志中的文件系统错误
	dmesgOutput, err := ctx.SSH.Exec(host, "dmesg -T -l err,warn | grep -iE '(ext4-fs|xfs|btrfs|zfs|filesystem)' | tail -20")
	if err == nil && dmesgOutput != "" {
		results["kernel_errors"] = dmesgOutput
	}

	// 获取 LVM 信息
	lvmOutput, err := ctx.SSH.Exec(host, "vgs && lvs -o lv_name,vg_name,lv_size,data_percent 2>/dev/null || echo 'LVM not configured'")
	if err == nil {
		results["lvm"] = lvmOutput
	}

	// 获取 RAID 信息
	raidOutput, err := ctx.SSH.Exec(host, "cat /proc/mdstat 2>/dev/null || echo 'No RAID configured'")
	if err == nil {
		results["raid"] = raidOutput
	}

	// 解析并汇总
	summary := t.analyzeAllFilesystems(results, detail)

	return tool.NewResult(map[string]interface{}{
		"host":       host,
		"raw_results": results,
		"summary":     summary,
		"timestamp":   time.Now().Format(time.RFC3339),
	}, summary), nil
}

// checkSpecificFilesystem 检查指定的文件系统
func (t *StorageCheckTool) checkSpecificFilesystem(ctx *tool.Context, host, mount, device string, detail bool) (*tool.Result, error) {
	// 首先确定文件系统类型
	fsType := ""
	var err error

	if mount != "" {
		fsType, err = t.getFilesystemType(ctx, host, mount)
	} else if device != "" {
		fsType, err = t.getFilesystemTypeByDevice(ctx, host, device)
	}

	if err != nil {
		return tool.NewErrorResult(fmt.Sprintf("无法确定文件系统类型: %v", err)), nil
	}

	// 根据文件系统类型执行特定的检查命令
	checks := t.getFSChecks(fsType, mount, device, detail)

	results := make(map[string]string)
	for _, check := range checks {
		output, err := ctx.SSH.Exec(host, check.cmd)
		if err != nil {
			results[check.name] = fmt.Sprintf("执行失败: %v", err)
		} else {
			results[check.name] = output
		}
	}

	// 分析结果
	analysis := t.analyzeFilesystemResults(fsType, results, mount, device)

	return tool.NewResult(map[string]interface{}{
		"host":           host,
		"filesystem_type": fsType,
		"mount":          mount,
		"device":         device,
		"checks":         results,
		"analysis":       analysis,
		"timestamp":      time.Now().Format(time.RFC3339),
	}, analysis), nil
}

type fsCheck struct {
	name string
	cmd  string
}

// getFSChecks 获取文件系统特定的检查命令
func (t *StorageCheckTool) getFSChecks(fsType, mount, device string, detail bool) []fsCheck {
	switch strings.ToLower(fsType) {
	case "ext4", "ext3", "ext2":
		return t.getExt4Checks(mount, device, detail)
	case "xfs":
		return t.getXFSChecks(mount, device, detail)
	case "btrfs":
		return t.getBtrfsChecks(mount, device, detail)
	case "zfs":
		return t.getZFSChecks(mount, device, detail)
	default:
		return []fsCheck{
			{name: "df", cmd: fmt.Sprintf("df -h %s", mount)},
			{name: "stat", cmd: fmt.Sprintf("stat -f %s", mount)},
		}
	}
}

// getExt4Checks 获取 ext4 检查命令
func (t *StorageCheckTool) getExt4Checks(mount, device string, detail bool) []fsCheck {
	checks := []fsCheck{
		{name: "tune2fs", cmd: "tune2fs -l " + device},
	}

	if mount != "" {
		checks = append(checks,
			fsCheck{name: "df", cmd: "df -h " + mount},
			fsCheck{name: "stat", cmd: "stat -f " + mount},
		)
	}

	if detail {
		checks = append(checks,
			fsCheck{name: "dumpe2fs", cmd: "dumpe2fs -h " + device},
		)
	}

	return checks
}

// getXFSChecks 获取 XFS 检查命令
func (t *StorageCheckTool) getXFSChecks(mount, device string, detail bool) []fsCheck {
	checks := []fsCheck{}

	if mount != "" {
		checks = append(checks,
			fsCheck{name: "xfs_info", cmd: "xfs_info " + mount},
			fsCheck{name: "df", cmd: "df -h " + mount},
		)
	}

	if detail {
		checks = append(checks,
			fsCheck{name: "xfs_db", cmd: "xfs_db -c 'sb 0' " + device},
		)
	}

	return checks
}

// getBtrfsChecks 获取 Btrfs 检查命令
func (t *StorageCheckTool) getBtrfsChecks(mount, device string, detail bool) []fsCheck {
	checks := []fsCheck{}

	if mount != "" {
		checks = append(checks,
			fsCheck{name: "btrfs_df", cmd: "btrfs filesystem df " + mount},
			fsCheck{name: "btrfs_device_stats", cmd: "btrfs device stats " + mount},
			fsCheck{name: "btrfs_subvolume", cmd: "btrfs subvolume list " + mount},
		)
	}

	if detail {
		checks = append(checks,
			fsCheck{name: "btrfs_scrub", cmd: "btrfs scrub status " + mount},
			fsCheck{name: "btrfs_usage", cmd: "btrfs filesystem usage " + mount},
		)
	}

	return checks
}

// getZFSChecks 获取 ZFS 检查命令
func (t *StorageCheckTool) getZFSChecks(mount, device string, detail bool) []fsCheck {
	pool := t.extractZFSPool(device)
	if pool == "" && mount != "" {
		pool = t.extractZFSPoolFromMount(mount)
	}

	checks := []fsCheck{
		{name: "zpool_status", cmd: "zpool status " + pool},
		{name: "zfs_list", cmd: "zfs list"},
	}

	if detail {
		checks = append(checks,
			fsCheck{name: "zpool_detail", cmd: "zpool status -v " + pool},
		)
	}

	return checks
}

// getFilesystemType 获取文件系统类型
func (t *StorageCheckTool) getFilesystemType(ctx *tool.Context, host, mount string) (string, error) {
	output, err := ctx.SSH.Exec(host, fmt.Sprintf("findmnt -n -o FSTYPE %s", mount))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// getFilesystemTypeByDevice 通过设备获取文件系统类型
func (t *StorageCheckTool) getFilesystemTypeByDevice(ctx *tool.Context, host, device string) (string, error) {
	output, err := ctx.SSH.Exec(host, fmt.Sprintf("blkid -o value -s TYPE %s 2>/dev/null", device))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// extractZFSPool 从设备路径提取 ZFS 池名称
func (t *StorageCheckTool) extractZFSPool(device string) string {
	if device == "" {
		return ""
	}
	// ZFS 通常使用池名而非设备路径
	if strings.HasPrefix(device, "/dev/zd") || strings.HasPrefix(device, "/dev/") {
		return ""
	}
	return device
}

// extractZFSPoolFromMount 从挂载点提取 ZFS 池名称
func (t *StorageCheckTool) extractZFSPoolFromMount(mount string) string {
	// 简化处理，实际应该执行命令查询
	return ""
}

// analyzeAllFilesystems 分析所有文件系统结果
func (t *StorageCheckTool) analyzeAllFilesystems(results map[string]string, detail bool) string {
	var sb strings.Builder

	sb.WriteString("## 文件系统检查结果\n\n")

	// 显示 df 结果
	if dfOutput, ok := results["df"]; ok {
		sb.WriteString("### 挂载的文件系统\n```\n")
		sb.WriteString(dfOutput)
		sb.WriteString("\n```\n\n")

		// 分析空间使用
		t.analyzeDiskSpace(dfOutput, &sb)
	}

	// 显示 LVM 信息
	if lvmOutput, ok := results["lvm"]; ok && !strings.Contains(lvmOutput, "LVM not configured") {
		sb.WriteString("### LVM 状态\n```\n")
		sb.WriteString(lvmOutput)
		sb.WriteString("\n```\n\n")

		t.analyzeLVM(lvmOutput, &sb)
	}

	// 显示 RAID 信息
	if raidOutput, ok := results["raid"]; ok && !strings.Contains(raidOutput, "No RAID") {
		sb.WriteString("### RAID 状态\n```\n")
		sb.WriteString(raidOutput)
		sb.WriteString("\n```\n\n")

		t.analyzeRAID(raidOutput, &sb)
	}

	// 检查内核错误
	if kernelErrors, ok := results["kernel_errors"]; ok && kernelErrors != "" {
		sb.WriteString("### ⚠️ 内核日志中的文件系统错误\n```\n")
		sb.WriteString(kernelErrors)
		sb.WriteString("\n```\n\n")
	}

	// 提供诊断建议
	t.provideRecommendations(results, &sb)

	return sb.String()
}

// analyzeDiskSpace 分析磁盘空间使用
func (t *StorageCheckTool) analyzeDiskSpace(dfOutput string, sb *strings.Builder) {
	lines := strings.Split(dfOutput, "\n")
	issues := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "/dev") || strings.HasPrefix(line, "tmpfs") {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				usePercent := strings.TrimSuffix(fields[5], "%")
				if usage, err := strconv.Atoi(usePercent); err == nil && usage > 80 {
					mountPoint := "/"
					if len(fields) > 6 {
						mountPoint = fields[6]
					}
					level := "⚠️"
					if usage > 90 {
						level = "🔴"
					}
					issues = append(issues, fmt.Sprintf("%s %s 使用率 %d%% (挂载点: %s)", level, fields[0], usage, mountPoint))
				}
			}
		}
	}

	if len(issues) > 0 {
		sb.WriteString("#### 空间使用警告\n")
		for _, issue := range issues {
			sb.WriteString(issue + "\n")
		}
		sb.WriteString("\n")
	}
}

// analyzeLVM 分析 LVM 状态
func (t *StorageCheckTool) analyzeLVM(lvmOutput string, sb *strings.Builder) {
	lines := strings.Split(lvmOutput, "\n")

	for _, line := range lines {
		// 检查 thin pool 使用率
		if strings.Contains(line, "%") {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.HasSuffix(f, "%") {
					percent := strings.TrimSuffix(f, "%")
					if usage, err := strconv.ParseFloat(percent, 64); err == nil && usage > 80 {
						sb.WriteString(fmt.Sprintf("⚠️ LVM thin pool 使用率: %.1f%%\n", usage))
					}
				}
			}
		}
	}
}

// analyzeRAID 分析 RAID 状态
func (t *StorageCheckTool) analyzeRAID(raidOutput string, sb *strings.Builder) {
	lines := strings.Split(raidOutput, "\n")

	for _, line := range lines {
		if strings.Contains(line, "md") {
			if strings.Contains(line, "active") {
				sb.WriteString("✅ RAID 状态: 活跃\n")
			} else if strings.Contains(line, "inactive") {
				sb.WriteString("❌ RAID 状态: 非活跃\n")
			}

			// 检查 RAID 级别
			if strings.Contains(line, "raid1") {
				sb.WriteString("**类型**: RAID 1 (镜像)\n")
			} else if strings.Contains(line, "raid5") {
				sb.WriteString("**类型**: RAID 5\n")
			} else if strings.Contains(line, "raid6") {
				sb.WriteString("**类型**: RAID 6\n")
			} else if strings.Contains(line, "raid10") {
				sb.WriteString("**类型**: RAID 10\n")
			}
		}

		// 检查重建进度
		if strings.Contains(line, "recovery") || strings.Contains(line, "resync") {
			sb.WriteString("🔄 **重建中**: " + strings.TrimSpace(line) + "\n")
		}

		// 检查故障磁盘
		if strings.Contains(line, "(F)") {
			sb.WriteString("⚠️ **警告**: 检测到故障磁盘\n")
		}
	}
	sb.WriteString("\n")
}

// analyzeFilesystemResults 分析特定文件系统结果
func (t *StorageCheckTool) analyzeFilesystemResults(fsType string, results map[string]string, mount, device string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## %s 文件系统分析\n\n", strings.ToUpper(fsType)))

	if mount != "" {
		sb.WriteString(fmt.Sprintf("**挂载点**: %s\n", mount))
	}
	if device != "" {
		sb.WriteString(fmt.Sprintf("**设备**: %s\n\n", device))
	}

	// 分析特定文件系统的结果
	switch strings.ToLower(fsType) {
	case "ext4", "ext3", "ext2":
		t.analyzeExt4Results(results, &sb)
	case "xfs":
		t.analyzeXFSResults(results, &sb)
	case "btrfs":
		t.analyzeBtrfsResults(results, &sb)
	case "zfs":
		t.analyzeZFSResults(results, &sb)
	}

	return sb.String()
}

// analyzeExt4Results 分析 ext4 结果
func (t *StorageCheckTool) analyzeExt4Results(results map[string]string, sb *strings.Builder) {
	if tune2fs, ok := results["tune2fs"]; ok {
		sb.WriteString("### 文件系统状态\n")
		lines := strings.Split(tune2fs, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Filesystem state") {
				if strings.Contains(line, "clean") {
					sb.WriteString("✅ **状态**: 健康\n")
				} else {
					sb.WriteString("⚠️ **状态**: " + line + "\n")
				}
			}
			if strings.Contains(line, "Last mounted") || strings.Contains(line, "Mount count") {
				sb.WriteString(line + "\n")
			}
		}
		sb.WriteString("\n")
	}
}

// analyzeXFSResults 分析 XFS 结果
func (t *StorageCheckTool) analyzeXFSResults(results map[string]string, sb *strings.Builder) {
	if xfsInfo, ok := results["xfs_info"]; ok {
		sb.WriteString("### 文件系统信息\n```\n")
		sb.WriteString(xfsInfo)
		sb.WriteString("\n```\n")
	}
}

// analyzeBtrfsResults 分析 Btrfs 结果
func (t *StorageCheckTool) analyzeBtrfsResults(results map[string]string, sb *strings.Builder) {
	if deviceStats, ok := results["btrfs_device_stats"]; ok {
		sb.WriteString("### 设备统计\n")
		if !strings.Contains(deviceStats, "0") || strings.Contains(deviceStats, "err") {
			sb.WriteString("⚠️ **检查设备错误**\n")
		} else {
			sb.WriteString("✅ **设备状态**: 无错误\n")
		}
		sb.WriteString("```\n" + deviceStats + "\n```\n\n")
	}

	if btrfsDf, ok := results["btrfs_df"]; ok {
		sb.WriteString("### 空间分布\n```\n")
		sb.WriteString(btrfsDf)
		sb.WriteString("\n```\n\n")
	}
}

// analyzeZFSResults 分析 ZFS 结果
func (t *StorageCheckTool) analyzeZFSResults(results map[string]string, sb *strings.Builder) {
	if zpoolStatus, ok := results["zpool_status"]; ok {
		sb.WriteString("### 池状态\n")
		if strings.Contains(zpoolStatus, "ONLINE") && !strings.Contains(zpoolStatus, "DEGRADED") {
			sb.WriteString("✅ **池状态**: 在线\n\n")
		} else if strings.Contains(zpoolStatus, "DEGRADED") {
			sb.WriteString("⚠️ **池状态**: 降级模式\n\n")
		}
		sb.WriteString("```\n" + zpoolStatus + "\n```\n\n")
	}

	if zfsList, ok := results["zfs_list"]; ok {
		sb.WriteString("### 数据集列表\n```\n")
		sb.WriteString(zfsList)
		sb.WriteString("\n```\n\n")
	}
}

// provideRecommendations 提供诊断建议
func (t *StorageCheckTool) provideRecommendations(results map[string]string, sb *strings.Builder) {
	sb.WriteString("### 诊断建议\n\n")

	// 检查是否有内核错误
	hasKernelErrors := false
	for _, output := range results {
		if strings.Contains(output, "I/O error") ||
			strings.Contains(output, "corruption") ||
			strings.Contains(output, "EXT4-fs error") {
			hasKernelErrors = true
			break
		}
	}

	if hasKernelErrors {
		sb.WriteString("⚠️ **发现内核错误**\n")
		sb.WriteString("1. 检查磁盘健康状况: `smartctl -a /dev/sdX`\n")
		sb.WriteString("2. 建议备份数据后运行文件系统检查\n\n")
	}

	sb.WriteString("### 通用建议\n")
	sb.WriteString("- 定期检查磁盘空间: `df -h`\n")
	sb.WriteString("- 监控内核日志: `dmesg -T -l err`\n")
	sb.WriteString("- 定期运行 SMART 检测: `smartctl -t short /dev/sdX`\n")
	sb.WriteString("- 检查文件系统: `tune2fs -l /dev/sdX1`\n")
}

// RAIDCheckTool RAID 阵列检查工具
type RAIDCheckTool struct {
	name        string
	description string
}

// NewRAIDCheckTool 创建 RAID 检查工具
func NewRAIDCheckTool() *RAIDCheckTool {
	return &RAIDCheckTool{
		name:        "check_raid",
		description: "检查软件 RAID (mdadm) 状态，包括阵列状态、重建进度、成员磁盘健康",
	}
}

// Name 返回工具名称
func (t *RAIDCheckTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *RAIDCheckTool) Description() string {
	return t.description
}

// Parameters 返回工具参数
func (t *RAIDCheckTool) Parameters() []tool.Parameter {
	return []tool.Parameter{
		{
			Name:        "host",
			Type:        "string",
			Description: "主机名称",
			Required:    true,
		},
		{
			Name:        "device",
			Type:        "string",
			Description: "RAID 设备，如 /dev/md0，为空则检查所有",
			Required:    false,
		},
	}
}

// Execute 执行 RAID 检查
func (t *RAIDCheckTool) Execute(ctx *tool.Context, params map[string]any) (*tool.Result, error) {
	host := tool.GetStringParam(params, "host", "")
	device := tool.GetStringParam(params, "device", "")

	if host == "" {
		return tool.NewErrorResult("host 参数必填"), nil
	}

	var sb strings.Builder
	sb.WriteString("## RAID 阵列检查结果\n\n")

	// 检查 /proc/mdstat
	mdstatOutput, err := ctx.SSH.Exec(host, "cat /proc/mdstat")
	if err == nil {
		sb.WriteString("### RAID 状态 (/proc/mdstat)\n```\n")
		sb.WriteString(mdstatOutput)
		sb.WriteString("\n```\n\n")

		t.analyzeMDStat(mdstatOutput, &sb)
	}

	// 如果指定了设备，显示详细信息
	if device != "" {
		detailOutput, err := ctx.SSH.Exec(host, fmt.Sprintf("mdadm --detail %s", device))
		if err == nil {
			sb.WriteString("### RAID 详细信息\n```\n")
			sb.WriteString(detailOutput)
			sb.WriteString("\n```\n\n")

			t.analyzeRAIDDetail(detailOutput, &sb)
		}
	} else {
		// 列出所有 RAID 设备
		detailOutput, err := ctx.SSH.Exec(host, "mdadm --detail --scan")
		if err == nil {
			sb.WriteString("### 所有 RAID 设备\n```\n")
			sb.WriteString(detailOutput)
			sb.WriteString("\n```\n\n")
		}
	}

	// 检查成员磁盘健康状态
	t.checkRAIDMemberHealth(ctx, host, &sb)

	// 提供建议
	t.provideRAIDRecommendations(&sb)

	return tool.NewResult(map[string]interface{}{
		"host":    host,
		"device":  device,
		"mdstat":  mdstatOutput,
		"summary": sb.String(),
		"timestamp": time.Now().Format(time.RFC3339),
	}, sb.String()), nil
}

// analyzeMDStat 分析 mdstat 输出
func (t *RAIDCheckTool) analyzeMDStat(mdstat string, sb *strings.Builder) {
	lines := strings.Split(mdstat, "\n")

	for _, line := range lines {
		// 检查 RAID 状态
		if strings.Contains(line, "md") {
			if strings.Contains(line, "active") {
				sb.WriteString("✅ **状态**: 活跃\n")
			} else if strings.Contains(line, "inactive") {
				sb.WriteString("❌ **状态**: 非活跃\n")
			}

			// 检查 RAID 级别
			if strings.Contains(line, "raid1") {
				sb.WriteString("**类型**: RAID 1 (镜像)\n")
			} else if strings.Contains(line, "raid5") {
				sb.WriteString("**类型**: RAID 5\n")
			} else if strings.Contains(line, "raid6") {
				sb.WriteString("**类型**: RAID 6\n")
			} else if strings.Contains(line, "raid10") {
				sb.WriteString("**类型**: RAID 10\n")
			}
		}

		// 检查重建进度
		if strings.Contains(line, "recovery") || strings.Contains(line, "resync") {
			sb.WriteString("🔄 **重建中**: " + strings.TrimSpace(line) + "\n")
		}

		// 检查降级状态
		if strings.Contains(line, "(F)") {
			sb.WriteString("⚠️ **警告**: 有故障磁盘\n")
		}
	}
	sb.WriteString("\n")
}

// analyzeRAIDDetail 分析 RAID 详细信息
func (t *RAIDCheckTool) analyzeRAIDDetail(detail string, sb *strings.Builder) {
	lines := strings.Split(detail, "\n")

	raidLevel := ""
	raidSize := ""
	raidState := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Raid Level") {
			raidLevel = strings.TrimPrefix(line, "Raid Level :")
		} else if strings.HasPrefix(line, "Array Size") {
			raidSize = strings.TrimPrefix(line, "Array Size :")
		} else if strings.HasPrefix(line, "State :") {
			raidState = strings.TrimPrefix(line, "State :")
		}
	}

	sb.WriteString("### RAID 配置\n")
	sb.WriteString(fmt.Sprintf("- **级别**: %s\n", raidLevel))
	sb.WriteString(fmt.Sprintf("- **大小**: %s\n", raidSize))
	sb.WriteString(fmt.Sprintf("- **状态**: %s\n", raidState))

	if strings.Contains(raidState, "clean") {
		sb.WriteString("✅ **阵列状态**: 健康\n")
	} else if strings.Contains(raidState, "degraded") {
		sb.WriteString("⚠️ **阵列状态**: 降级模式\n")
		sb.WriteString("**建议**: 立即更换故障磁盘并重建阵列\n")
	} else if strings.Contains(raidState, "recovering") {
		sb.WriteString("🔄 **阵列状态**: 正在重建\n")
	}
	sb.WriteString("\n")
}

// checkRAIDMemberHealth 检查 RAID 成员磁盘健康
func (t *RAIDCheckTool) checkRAIDMemberHealth(ctx *tool.Context, host string, sb *strings.Builder) {
	sb.WriteString("### 成员磁盘健康状态\n\n")

	// 获取所有块设备
	devicesOutput, err := ctx.SSH.Exec(host, "lsblk -d -o NAME,TYPE,SIZE,ROTA 2>/dev/null | grep -E 'disk|md' || echo '无法获取设备列表'")
	if err != nil {
		sb.WriteString("无法获取设备列表\n")
		return
	}

	lines := strings.Split(devicesOutput, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && (strings.Contains(fields[0], "md") || strings.Contains(fields[0], "sd") || strings.Contains(fields[0], "nvme")) {
			device := "/dev/" + fields[0]

			// 获取 SMART 状态
			smartOutput, err := ctx.SSH.Exec(host, fmt.Sprintf("smartctl -H %s 2>/dev/null || echo 'SMART not available'", device))
			if err == nil {
				if strings.Contains(smartOutput, "PASSED") {
					sb.WriteString(fmt.Sprintf("✅ %s: 健康\n", device))
				} else if strings.Contains(smartOutput, "FAILED") {
					sb.WriteString(fmt.Sprintf("❌ %s: **故障**\n", device))
				}
			}
		}
	}
	sb.WriteString("\n")
}

// provideRAIDRecommendations 提供 RAID 建议
func (t *RAIDCheckTool) provideRAIDRecommendations(sb *strings.Builder) {
	sb.WriteString("### RAID 维护建议\n\n")
	sb.WriteString("1. **监控阵列状态**\n")
	sb.WriteString("   ```bash\n   watch cat /proc/mdstat\n   ```\n\n")

	sb.WriteString("2. **故障磁盘更换流程**\n")
	sb.WriteString("   ```bash\n   mdadm --fail /dev/md0 /dev/sdX1    # 标记故障\n")
	sb.WriteString("   mdadm --remove /dev/md0 /dev/sdX1  # 移除\n")
	sb.WriteString("   mdadm --add /dev/md0 /dev/sdY1     # 添加新盘\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("3. **优化重建速度**\n")
	sb.WriteString("   ```bash\n   echo 100000 > /proc/sys/dev/raid/speed_limit_min\n")
	sb.WriteString("   echo 500000 > /proc/sys/dev/raid/speed_limit_max\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("4. **定期检查**\n")
	sb.WriteString("- 每月检查 SMART 状态: `smartctl -a /dev/sdX`\n")
	sb.WriteString("- 监控重建进度\n")
	sb.WriteString("- 备份重要数据\n")
}
