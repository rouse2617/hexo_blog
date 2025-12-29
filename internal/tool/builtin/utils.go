package builtin

// truncateOutput 截断输出，避免返回数据过大
func truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... (output truncated)"
}
