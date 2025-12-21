// Agent 引用接口
export interface AgentReference {
  id: string;
  displayName: string;
}

// 日志条目接口
export interface LogEntry {
  id: string;
  timestamp: string;
  message: string;
  agentRefs?: AgentReference[];
}

// 上下文日志接口
export interface ContextLogs {
  agentId: string;
  contextEntries: LogEntry[];
}
