# SQLite WAL 模式优化说明

## 优化概述

本次优化启用了 SQLite 的 WAL (Write-Ahead Logging) 模式，并配置了多个性能参数以提升并发性能。

## WAL 模式优势

### 1. 并发性能提升
- **读写并发**：读者不会阻塞写者，写者不会阻塞读者
- **多读者**：支持多个并发读操作
- **性能提升**：写入操作不会立即同步到主数据库文件

### 2. 更好的崩溃恢复
- WAL 文件包含所有未提交的更改
- 崩溃恢复更快速、更可靠
- 减少数据损坏风险

### 3. 自动检查点
- 自动将 WAL 文件内容合并回主数据库
- 防止 WAL 文件无限增长
- 可配置的检查点策略

## 配置参数说明

### PRAGMA journal_mode=WAL
```sql
PRAGMA journal_mode=WAL;
```
- **作用**：启用 WAL 模式
- **文件结构**：
  - `数据库.db` - 主数据库文件
  - `数据库.db-wal` - WAL 日志文件
  - `数据库.db-shm` - 共享内存索引

### PRAGMA synchronous=NORMAL
```sql
PRAGMA synchronous=NORMAL;
```
- **作用**：设置同步模式为 NORMAL
- **安全性**：在 WAL 模式下提供良好的安全性
- **性能**：比 FULL 模式快得多
- **平衡**：在大多数情况下安全性与性能的最佳平衡

**同步模式对比**：
| 模式 | 安全性 | 性能 | 说明 |
|------|--------|------|------|
| FULL | 最高 | 最慢 | 每次写入都同步到磁盘 |
| NORMAL | 高 | 快 | 关键操作同步（WAL 模式推荐） |
| OFF | 低 | 最快 | 不同步，可能丢失数据 |

### PRAGMA cache_size=-2000
```sql
PRAGMA cache_size=-2000;
```
- **作用**：设置缓存大小为 2MB
- **说明**：
  - 负值表示 KB（-2000 = 2000KB = 2MB）
  - 正值表示页数（每页 4KB）
- **调整建议**：
  - 内存充足：可增加到 -10000（10MB）
  - 内存受限：保持 -2000（2MB）

### PRAGMA wal_autocheckpoint=1000
```sql
PRAGMA wal_autocheckpoint=1000;
```
- **作用**：当 WAL 文件达到 1000 页（约 4MB）时执行检查点
- **说明**：
  - 自动检查点触发阈值
  - 页大小通常为 4096 字节
- **调整建议**：
  - 写入频繁：增加阈值减少检查点频率
  - 读多写少：降低阈值加快检查点

### PRAGMA busy_timeout=5000
```sql
PRAGMA busy_timeout=5000;
```
- **作用**：数据库锁定时等待 5 秒
- **说明**：
  - 当数据库被其他事务锁定时，等待时间
  - 防止立即返回 SQLITE_BUSY 错误
- **调整建议**：
  - 高并发场景：增加到 10000（10 秒）
  - 低并发场景：保持 5000（5 秒）

### PRAGMA temp_store=MEMORY
```sql
PRAGMA temp_store=MEMORY;
```
- **作用**：将临时表和索引存储在内存中
- **优势**：
  - 更快地执行排序、分组等操作
  - 减少磁盘 I/O
- **代价**：增加内存使用

### PRAGMA mmap_size=268435456
```sql
PRAGMA mmap_size=268435456;
```
- **作用**：启用 256MB 的内存映射 I/O
- **优势**：
  - 减少 `read()` 系统调用
  - 操作系统管理缓存
  - 在 64 位系统上效果显著
- **说明**：268435456 = 256 * 1024 * 1024

### PRAGMA page_size=4096
```sql
PRAGMA page_size=4096;
```
- **作用**：设置页面大小为 4KB
- **说明**：
  - 大多数文件系统的最佳默认值
  - 与文件系统块大小对齐
- **注意**：只能在创建数据库时设置

### PRAGMA foreign_keys=ON
```sql
PRAGMA foreign_keys=ON;
```
- **作用**：启用外键约束
- **优势**：保证数据引用完整性
- **性能影响**：轻微性能开销

### PRAGMA optimize
```sql
PRAGMA optimize;
```
- **作用**：优化数据库统计信息
- **执行时机**：每次数据库初始化时
- **效果**：查询优化器选择更好的执行计划

## 新增索引

### 1. idx_messages_session_created
```sql
CREATE INDEX idx_messages_session_created
ON messages(session_id, created_at DESC);
```
- **优化场景**：按会话查询消息并按时间排序
- **查询示例**：
  ```sql
  SELECT * FROM messages
  WHERE session_id = ?
  ORDER BY created_at DESC;
  ```

### 2. idx_hosts_status
```sql
CREATE INDEX idx_hosts_status ON hosts(status);
```
- **优化场景**：按状态筛选主机
- **查询示例**：
  ```sql
  SELECT * FROM hosts WHERE status = 'online';
  ```

### 3. idx_hosts_group_status
```sql
CREATE INDEX idx_hosts_group_status
ON hosts(`group`, status);
```
- **优化场景**：按组和状态组合筛选
- **查询示例**：
  ```sql
  SELECT * FROM hosts
  WHERE `group` = 'production' AND status = 'online';
  ```

### 4. idx_analysis_session_created
```sql
CREATE INDEX idx_analysis_session_created
ON analysis(session_id, created_at DESC);
```
- **优化场景**：查询特定会话的分析历史
- **查询示例**：
  ```sql
  SELECT * FROM analysis
  WHERE session_id = ?
  ORDER BY created_at DESC;
  ```

### 5. idx_messages_role
```sql
CREATE INDEX idx_messages_role ON messages(role);
```
- **优化场景**：按角色筛选消息
- **查询示例**：
  ```sql
  SELECT * FROM messages WHERE role = 'assistant';
  ```

## 性能对比

### 预期性能提升

| 操作 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| 并发读取 | 串行 | 并发 | 5-10x |
| 写入操作 | 阻塞读取 | 不阻塞 | 2-3x |
| 读取操作 | 被写入阻塞 | 不阻塞 | 3-5x |
| 崩溃恢复 | 较慢 | 快速 | 2x |

### 实际应用场景

#### 场景 1：多用户同时访问
```
优化前：用户 A 写入时，用户 B 必须等待
优化后：用户 A 写入时，用户 B 可以同时读取
```

#### 场景 2：大量消息查询
```
优化前：全表扫描 messages 表
优化后：使用 idx_messages_session_created 索引
性能提升：10-100 倍（取决于数据量）
```

#### 场景 3：主机状态监控
```
优化前：全表扫描 hosts 表筛选状态
优化后：使用 idx_hosts_status 和 idx_hosts_group_status
性能提升：5-50 倍
```

## WAL 文件管理

### 文件说明

启用 WAL 模式后，数据库目录会生成三个文件：

```
ai-ops.db          # 主数据库文件
ai-ops.db-wal      # WAL 日志文件（未提交的更改）
ai-ops.db-shm      # 共享内存索引（-shm 文件）
```

### 文件大小控制

1. **自动检查点**：
   - WAL 文件达到 4MB 时自动触发
   - 将 WAL 内容合并到主数据库
   - 自动清空 WAL 文件

2. **手动检查点**（如需要）：
   ```go
   db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
   ```

3. **WAL 文件过大**：
   - 正常现象，表示有大量未合并的写入
   - 会自动在检查点时清理
   - 可通过 `wal_autocheckpoint` 调整

### 备份注意事项

使用 WAL 模式时，备份需要特别注意：

#### 方法 1：使用在线备份
```go
// GORM 不直接支持，需要使用 database/sql
db.Exec("VACUUM INTO 'backup.db'")
```

#### 方法 2：先执行检查点
```go
db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
// 然后复制 .db 文件
```

#### 方法 3：使用 SQLite 工具
```bash
# 使用 sqlite3 命令行工具
sqlite3 ai-ops.db ".backup backup.db"
```

## 监控和维护

### 查看当前配置

```go
// 查看 WAL 模式
var journalMode string
db.Raw("PRAGMA journal_mode").Scan(&journalMode)

// 查看页面大小
var pageSize int
db.Raw("PRAGMA page_size").Scan(&pageSize)

// 查看 WAL 文件大小
var walSize int
db.Raw("PRAGMA wal_checkpoint(PASSIVE)").Scan(&walSize)
```

### 性能监控

```sql
-- 查看表统计信息
SELECT * FROM sqlite_master WHERE type = 'table';

-- 查看索引统计信息
SELECT * FROM sqlite_master WHERE type = 'index';

-- 分析查询计划
EXPLAIN QUERY PLAN
SELECT * FROM messages
WHERE session_id = 'xxx'
ORDER BY created_at DESC;
```

### 定期维护

#### 1. 重建索引（可选）
```go
// 当数据大量变化后
db.Exec("REINDEX")
```

#### 2. 分析数据库（可选）
```go
// 更新统计信息帮助查询优化器
db.Exec("ANALYZE")
```

#### 3. 清理数据库（可选）
```go
// 回收空间并整理文件
db.Exec("VACUUM")
```

## 故障排查

### 问题 1：WAL 文件过大

**现象**：`.db-wal` 文件持续增长

**原因**：
- 写入频繁但检查点未触发
- 长事务未提交

**解决方案**：
```go
// 手动触发检查点
db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

// 或降低自动检查点阈值
db.Exec("PRAGMA wal_autocheckpoint=500")
```

### 问题 2：数据库锁定超时

**现象**：`database is locked` 错误

**原因**：
- 多个写入冲突
- 长事务未提交

**解决方案**：
```go
// 增加超时时间
db.Exec("PRAGMA busy_timeout=10000")

// 或使用更短的事务
db.Transaction(func(tx *gorm.DB) error {
    // 快速操作
    return nil
}, &sql.TxOptions{Timeout: 5 * time.Second})
```

### 问题 3：读性能未提升

**现象**：查询仍然缓慢

**原因**：
- 索引未生效
- 查询计划不优

**解决方案**：
```sql
-- 检查查询计划
EXPLAIN QUERY SELECT * FROM messages WHERE session_id = 'xxx';

-- 重新分析
ANALYZE;

-- 重建索引
REINDEX;
```

## 兼容性说明

### 支持的 SQLite 版本
- SQLite 3.7.0+（2010-07-21）开始支持 WAL 模式
- 推荐 SQLite 3.30.0+（2019-10-04）以获得最佳性能

### 网络文件系统注意事项
WAL 模式在某些网络文件系统上可能不稳定：
- NFS：需要支持锁定
- SMB：通常支持
- CIFS：可能有问题

如果使用网络文件系统，建议：
1. 测试 WAL 模式的稳定性
2. 如有问题，回退到默认的 DELETE 模式：
   ```go
   "PRAGMA journal_mode=DELETE;"
   ```

## 最佳实践

### 1. 事务处理
```go
// 使用事务提高性能
db.Transaction(func(tx *gorm.DB) error {
    // 多个写入操作
    for _, item := range items {
        if err := tx.Create(&item).Error; err != nil {
            return err // 自动回滚
        }
    }
    return nil // 自动提交
})
```

### 2. 批量操作
```go
// 批量插入比逐条插入快得多
db.CreateInBatches(items, 100)

// 或使用 Create
db.Create(items) // GORM 自动分批
```

### 3. 避免长事务
```go
// 好的做法：短事务
db.Transaction(func(tx *gorm.DB) error {
    return tx.Create(&record).Error
})

// 避免：长事务
db.Transaction(func(tx *gorm.DB) error {
    // 耗时操作
    time.Sleep(10 * time.Second)
    return tx.Create(&record).Error
})
```

### 4. 使用连接池
```go
sqlDB, _ := db.DB()
// 设置空闲连接池
sqlDB.SetMaxIdleConns(10)
// 设置最大连接数
sqlDB.SetMaxOpenConns(100)
// 设置连接最大存活时间
sqlDB.SetConnMaxLifetime(time.Hour)
```

## 参考资源

- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
- [SQLite PRAGMA Statements](https://www.sqlite.org/pragma.html)
- [SQLite Query Optimization](https://www.sqlite.org/queryoptimizer.html)
- [GORM SQLite Driver](https://gorm.io/docs/connecting_to_the_database.html#SQLite)

## 总结

本次优化通过以下措施显著提升 SQLite 性能：

1. ✅ 启用 WAL 模式 - 支持读写并发
2. ✅ 配置合理的同步模式 - 平衡安全性和性能
3. ✅ 优化缓存和内存使用 - 减少磁盘 I/O
4. ✅ 添加关键索引 - 加速常见查询
5. ✅ 启用内存映射 I/O - 减少系统调用

预期效果：
- 读操作性能提升 3-10 倍
- 写操作性能提升 2-3 倍
- 并发场景下性能提升更显著
- 崩溃恢复更快更可靠
