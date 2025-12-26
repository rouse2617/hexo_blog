# API 接口设计

## 概述

RESTful API，供前端和外部系统调用。

## 基础规范

### 请求格式

```
Content-Type: application/json
```

### 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

### 错误码

| code | 含义 |
|------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 资源不存在 |
| 2001 | SSH 连接失败 |
| 2002 | 命令执行失败 |
| 3001 | LLM 调用失败 |
| 5000 | 内部错误 |

---

## 对话 API

### 发送消息

**POST** `/api/chat`

与 AI Agent 对话，Agent 会理解意图并执行相应操作。

**请求:**

```json
{
  "session_id": "xxx",        // 可选，不传则创建新会话
  "message": "查看 node-10 的 nginx 日志",
  "hosts": ["node-10"],       // 可选，指定目标主机
  "stream": false             // 是否流式返回
}
```

**响应:**

```json
{
  "code": 0,
  "data": {
    "session_id": "sess_abc123",
    "reply": "node-10 的 nginx 日志显示最近有 3 个 502 错误...",
    "tool_calls": [
      {
        "tool": "query_log",
        "params": {"host": "node-10", "log_type": "nginx"},
        "result": "..."
      }
    ],
    "thinking": "用户想查看日志，我需要调用 query_log 工具..."
  }
}
```

### 流式对话

**POST** `/api/chat/stream`

SSE (Server-Sent Events) 流式返回。

**事件类型:**

```
event: thinking
data: {"content": "正在分析用户意图..."}

event: tool_call
data: {"tool": "query_log", "params": {...}}

event: tool_result
data: {"tool": "query_log", "result": "..."}

event: reply
data: {"content": "根据日志分析..."}

event: done
data: {"session_id": "xxx"}
```

### 获取对话历史

**GET** `/api/chat/history?session_id=xxx`

**响应:**

```json
{
  "code": 0,
  "data": {
    "session_id": "sess_abc123",
    "messages": [
      {"role": "user", "content": "查看 node-10 的日志", "time": "..."},
      {"role": "assistant", "content": "...", "time": "..."}
    ]
  }
}
```

### 获取会话列表

**GET** `/api/chat/sessions`

**响应:**

```json
{
  "code": 0,
  "data": {
    "sessions": [
      {
        "id": "sess_abc123",
        "title": "查看 node-10 日志",
        "created_at": "2024-01-01T10:00:00Z",
        "updated_at": "2024-01-01T10:05:00Z"
      }
    ]
  }
}
```

---

## 主机管理 API

### 获取主机列表

**GET** `/api/hosts`

**查询参数:**

| 参数 | 说明 |
|------|------|
| group | 按分组过滤 |
| keyword | 搜索关键词 |

**响应:**

```json
{
  "code": 0,
  "data": {
    "hosts": [
      {
        "id": "host_001",
        "name": "node-10",
        "ip": "192.168.1.10",
        "port": 22,
        "user": "root",
        "group": "web",
        "tags": ["nginx", "production"],
        "status": "online",
        "last_check": "2024-01-01T10:00:00Z"
      }
    ],
    "total": 100
  }
}
```

### 添加主机

**POST** `/api/hosts`

**请求:**

```json
{
  "name": "node-11",
  "ip": "192.168.1.11",
  "port": 22,
  "user": "root",
  "group": "web",
  "tags": ["nginx"],
  "auth_type": "key",
  "password": "",
  "key_path": "/root/.ssh/id_rsa"
}
```

### 更新主机

**PUT** `/api/hosts/:id`

### 删除主机

**DELETE** `/api/hosts/:id`

### 测试主机连接

**POST** `/api/hosts/:id/test`

**响应:**

```json
{
  "code": 0,
  "data": {
    "success": true,
    "latency": "23ms",
    "message": "连接成功"
  }
}
```

### 批量导入主机

**POST** `/api/hosts/import`

**请求:**

```json
{
  "format": "csv",
  "data": "name,ip,port,user,group\nnode-1,192.168.1.1,22,root,web\n..."
}
```

---

## 主机分组 API

### 获取分组列表

**GET** `/api/groups`

**响应:**

```json
{
  "code": 0,
  "data": {
    "groups": [
      {"name": "web", "count": 20},
      {"name": "db", "count": 5},
      {"name": "cache", "count": 10}
    ]
  }
}
```

### 创建分组

**POST** `/api/groups`

### 删除分组

**DELETE** `/api/groups/:name`

---

## 工具 API

### 获取工具列表

**GET** `/api/tools`

**响应:**

```json
{
  "code": 0,
  "data": {
    "tools": [
      {
        "name": "query_log",
        "description": "查询指定节点的日志文件",
        "type": "builtin",
        "parameters": [
          {"name": "host", "type": "string", "required": true},
          {"name": "log_type", "type": "string", "required": true}
        ]
      },
      {
        "name": "check_redis",
        "description": "检查 Redis 状态",
        "type": "script",
        "parameters": [...]
      }
    ]
  }
}
```

### 手动调用工具

**POST** `/api/tools/:name/execute`

绕过 Agent，直接调用工具（调试用）。

**请求:**

```json
{
  "params": {
    "host": "node-10",
    "log_type": "nginx",
    "lines": 50
  }
}
```

---

## 脚本管理 API

### 获取脚本列表

**GET** `/api/scripts`

### 上传脚本

**POST** `/api/scripts`

**请求 (multipart/form-data):**

```
file: check_mysql.sh
meta: {
  "name": "check_mysql",
  "description": "检查 MySQL 状态",
  "parameters": [...]
}
```

### 删除脚本

**DELETE** `/api/scripts/:name`

### 更新脚本

**PUT** `/api/scripts/:name`

---

## 任务 API (异步任务)

### 创建任务

**POST** `/api/tasks`

用于长时间运行的任务。

**请求:**

```json
{
  "type": "batch_check",
  "params": {
    "hosts": ["node-1", "node-2", "..."],
    "command": "df -h"
  }
}
```

**响应:**

```json
{
  "code": 0,
  "data": {
    "task_id": "task_xyz789",
    "status": "running"
  }
}
```

### 获取任务状态

**GET** `/api/tasks/:id`

**响应:**

```json
{
  "code": 0,
  "data": {
    "task_id": "task_xyz789",
    "status": "completed",
    "progress": 100,
    "result": {...},
    "created_at": "...",
    "completed_at": "..."
  }
}
```

### 取消任务

**POST** `/api/tasks/:id/cancel`

---

## 系统 API

### 健康检查

**GET** `/api/health`

```json
{
  "status": "ok",
  "version": "1.0.0",
  "llm_status": "connected",
  "uptime": "24h"
}
```

### 获取系统配置

**GET** `/api/system/config`

### 更新系统配置

**PUT** `/api/system/config`

---

## WebSocket API

### 实时日志

**WS** `/ws/logs`

订阅实时日志流。

```json
// 订阅
{"action": "subscribe", "host": "node-10", "log_path": "/var/log/nginx/access.log"}

// 取消订阅
{"action": "unsubscribe"}

// 服务端推送
{"type": "log", "content": "..."}
```

### 任务进度

**WS** `/ws/tasks/:id`

实时获取任务执行进度。

---

## 认证 (可选，后续扩展)

### 登录

**POST** `/api/auth/login`

```json
{
  "username": "admin",
  "password": "xxx"
}
```

**响应:**

```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "..."
  }
}
```

### 请求头

```
Authorization: Bearer <token>
```
