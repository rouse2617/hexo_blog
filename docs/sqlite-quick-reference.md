# SQLite WAL 模式快速参考卡片

## 核心配置

### WAL 模式参数速查表

| 参数 | 值 | 作用 | 内存/磁盘影响 |
|------|-----|------|---------------|
| journal_mode | WAL | 启用写前日志 | 创建 .db-wal 和 .db-shm 文件 |
| synchronous | NORMAL | 关键操作同步 | 平衡安全性和性能 |
| cache_size | -2000 (2MB) | 页面缓存 | 内存占用 2MB |
| wal_autocheckpoint | 1000 (4MB) | 检查点阈值 | WAL 文件最大 4MB |
| busy_timeout | 5000ms | 锁等待超时 | 最多等待 5 秒 |
| temp_store | MEMORY | 临时表存储 | 内存临时表 |
| mmap_size | 256MB | 内存映射 | 最多映射 256MB |
| page_size | 4096 | 页面大小 | 每页 4KB |
| foreign_keys | ON | 外键约束 | 启用完整性检查 |

## WAL 文件结构

```
ai-ops.db          # 主数据库文件
ai-ops.db-wal      # WAL 日志（未合并的写入）
ai-ops.db-shm      # 共享内存索引
```

**自动维护**：WAL 文件达到 4MB 时自动检查点清理

## 新增索引

| 索引名 | 表 | 字段 | 用途 |
|--------|-----|------|------|
| idx_messages_session_created | messages | session_id, created_at | 会话消息查询 |
| idx_hosts_status | hosts | status | 主机状态筛选 |
| idx_hosts_group_status | hosts | group, status | 组+状态组合查询 |
| idx_analysis_session_created | analysis | session_id, created_at | 会话分析历史 |
| idx_messages_role | messages | role | 消息角色筛选 |

## 性能预期

### 并发场景

| 操作 | 默认模式 | WAL 模式 | 提升 |
|------|----------|----------|------|
| 并发读取 | 串行 | 并行 | 5-10x |
| 写入时读取 | 阻塞 | 不阻塞 | ∞ |
| 读取时写入 | 阻塞 | 不阻塞 | ∞ |

### 查询性能

| 查询类型 | 无索引 | 有索引 | 提升 |
|----------|--------|--------|------|
| 按会话查消息 | 全表扫描 | 索引查找 | 10-100x |
| 按状态查主机 | 全表扫描 | 索引查找 | 5-50x |
| 按组+状态查主机 | 全表扫描 | 复合索引 | 10-100x |

## 常用命令

### 查看配置

```go
// 查看 WAL 模式
var journalMode string
db.Raw("PRAGMA journal_mode").Scan(&journalMode)

// 查看缓存大小
var cacheSize int
db.Raw("PRAGMA cache_size").Scan(&cacheSize)

// 查看同步模式
var synchronous string
db.Raw("PRAGMA synchronous").Scan(&synchronous)
```

### 性能分析

```go
// 分析查询计划
var plans []map[string]interface{}
db.Raw("EXPLAIN QUERY PLAN SELECT * FROM messages WHERE session_id = ?", sessionID).Scan(&plans)

// 查看表统计
db.Raw("SELECT * FROM sqlite_master WHERE type = 'table'")
```

### 维护操作

```go
// 手动检查点
db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

// 分析数据库
db.Exec("ANALYZE")

// 重建索引
db.Exec("REINDEX")

// 清理数据库
db.Exec("VACUUM")
```

## 故障排查速查

### 问题 1：WAL 文件过大

```go
// 解决方案：手动触发检查点
db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

// 或降低自动检查点阈值
db.Exec("PRAGMA wal_autocheckpoint=500")
```

### 问题 2：数据库锁定

```go
// 解决方案：增加超时时间
db.Exec("PRAGMA busy_timeout=10000")
```

### 问题 3：查询慢

```sql
-- 1. 检查查询计划
EXPLAIN QUERY PLAN SELECT ...

-- 2. 更新统计信息
ANALYZE;

-- 3. 重建索引
REINDEX;
```

## 备份命令

### 方法 1：检查点 + 复制

```bash
# 1. 执行检查点
sqlite3 ai-ops.db "PRAGMA wal_checkpoint(TRUNCATE);"

# 2. 复制文件
cp ai-ops.db backup.db
```

### 方法 2：SQLite 在线备份

```bash
sqlite3 ai-ops.db ".backup backup.db"
```

## 最佳实践

### 1. 事务处理

```go
// 推荐：短事务
db.Transaction(func(tx *gorm.DB) error {
    return tx.Create(&record).Error
})

// 避免：长事务（阻塞其他操作）
```

### 2. 批量操作

```go
// 推荐：批量插入
db.CreateInBatches(items, 100)

// 避免：逐条插入
for _, item := range items {
    db.Create(&item)  // 慢
}
```

### 3. 连接池配置

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)
```

## 性能测试

```bash
# 运行性能测试脚本
cd C:\Users\hrp\Downloads\ai-pro\scripts
go run test-sqlite-performance.go
```

## 关键优势总结

1. **并发性**：读写不阻塞，支持多用户同时访问
2. **性能**：查询速度提升 5-100 倍
3. **可靠性**：更好的崩溃恢复机制
4. **自动化**：自动检查点，无需手动维护
5. **兼容性**：完全兼容现有代码，透明优化

## 监控指标

```sql
-- 查看数据库大小
SELECT page_count * page_size as size
FROM pragma_page_count(), pragma_page_size();

-- 查看 WAL 文件大小
SELECT * FROM pragma_wal_checkpoint();

-- 查看索引使用情况
SELECT * FROM pragma_index_info('idx_messages_session_created');
```

## 注意事项

1. **备份**：使用 WAL 模式时需要先执行检查点再备份
2. **网络文件系统**：某些网络文件系统可能不支持 WAL
3. **长事务**：避免长时间运行的事务，会导致 WAL 文件增长
4. **内存使用**：cache_size 和 mmap_size 会增加内存占用

## 兼容性

- **最低 SQLite 版本**：3.7.0+
- **推荐 SQLite 版本**：3.30.0+
- **Go 版本**：1.16+
- **GORM 版本**：v1.20+

## 参考文档

- [完整优化说明](./sqlite-wal-optimization.md)
- [优化总结](./sqlite-optimization-summary.md)
- [SQLite 官方文档](https://www.sqlite.org/wal.html)
