# 数据库错误修复：resource_time_is_manual 字段

## 🐛 问题描述

### 错误信息
```
加载失败: HTTP 500: {"error":"no such column: resource_time_is_manual"}
```

### 发生位置
- **页面**：我的片库（Library）
- **操作**：加载已订阅剧集列表
- **API**：`GET /api/library`

## 🔍 原因分析

### 问题根源

数据库表 `tv_shows` 缺少 `resource_time_is_manual` 列。

### 详细说明

1. **模型定义**（`internal/models/models.go`）
   ```go
   type TVShow struct {
       ResourceTimeIsManual bool `json:"resource_time_is_manual"`
       // ... 其他字段
   }
   ```
   模型中定义了 `ResourceTimeIsManual` 字段

2. **数据库 Schema**（`internal/repository/sqlite.go`）
   ```sql
   CREATE TABLE IF NOT EXISTS tv_shows (
       ...
       resource_time_is_manual BOOLEAN DEFAULT FALSE,
       ...
   );
   ```
   新建数据库时会创建此列

3. **问题**：
   - 旧数据库在创建时没有此列
   - 代码更新后查询包含此字段
   - 导致 SQL 查询失败

## ✅ 解决方案

### 方案 1：自动迁移（推荐）

**已实现**：应用启动时自动检测并迁移数据库

**工作原理**：
1. 应用启动时检查列是否存在
2. 如果不存在，自动执行迁移
3. 无需手动操作

**使用方法**：
```bash
# 重启应用即可
docker compose restart tv-tracker

# 查看日志确认迁移成功
docker compose logs tv-tracker | grep -i migration
```

### 方案 2：手动迁移（备选）

如果自动迁移失败，可以手动执行 SQL：

#### 方法 A：使用 SQLite 命令行

```bash
# 进入容器
docker compose exec tv-tracker sh

# 安装 sqlite3（如果没有）
apk add sqlite3

# 连接数据库
sqlite3 /app/data/tv_tracker.db

# 执行迁移脚本
.read /migrations/add_resource_time_is_manual.sql

# 退出
.quit
```

#### 方法 B：使用 Docker 卷映射

```bash
# 停止容器
docker compose down

# 手动执行迁移
docker run --rm -v \
  "$(pwd)/data:/data" \
  nouchka/sqlite3:latest \
  /data/tv_tracker.db \
  < migrations/add_resource_time_is_manual.sql

# 重启容器
docker compose up -d
```

### 方案 3：重建数据库（最后手段）

⚠️ **警告**：此方法会丢失所有数据！

```bash
# 1. 备份现有数据（可选）
cp data/tv_tracker.db data/tv_tracker.db.backup

# 2. 删除旧数据库
rm data/tv_tracker.db

# 3. 重启容器（会自动创建新数据库）
docker compose up -d

# 4. 重新订阅剧集
```

## 🔧 验证修复

### 检查列是否存在

```bash
# 方法 1：使用 SQLite
docker compose exec tv-tracker sqlite3 /app/data/tv_tracker.db \
  "PRAGMA table_info(tv_shows);" | grep resource_time_is_manual

# 应该看到：
# 7|resource_time_is_manual|BOOLEAN|0||0
```

### 测试 API

```bash
# 测试片库 API
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:8318/api/library

# 应该返回剧集列表，不再报错
```

### 测试 Web 界面

1. 打开浏览器访问 `http://localhost:8318`
2. 点击"我的片库"标签
3. 应该正常显示已订阅的剧集

## 📊 迁移详情

### 迁移过程

```sql
-- 1. 创建新表（包含 resource_time_is_manual 列）
CREATE TABLE tv_shows_new (...);

-- 2. 复制数据（新列默认值为 FALSE）
INSERT INTO tv_shows_new (...) SELECT ... FROM tv_shows;

-- 3. 删除旧表
DROP TABLE tv_shows;

-- 4. 重命名新表
ALTER TABLE tv_shows_new RENAME TO tv_shows;

-- 5. 重建索引
CREATE INDEX idx_shows_tmdb_archived ON tv_shows(tmdb_id, is_archived);
```

### 数据完整性

- ✅ 所有现有数据保留
- ✅ `resource_time_is_manual` 默认值为 `FALSE`
- ✅ 不影响现有功能
- ✅ 事务保证原子性

## 🛡️ 预防措施

### 1. 版本控制

在数据库中添加版本表：

```go
// 未来可以实现版本追踪
type SchemaVersion struct {
    Version int
    AppliedAt time.Time
}
```

### 2. 迁移脚本规范

- ✅ 每次数据库变更都创建迁移脚本
- ✅ 迁移脚本应该是幂等的（可重复执行）
- ✅ 在 `migrations/` 目录下统一管理

### 3. 测试流程

```bash
# 本地测试迁移
docker compose down
rm data/tv_tracker.db
docker compose up -d

# 验证功能
curl http://localhost:8318/api/library
```

## 📝 相关文件

### 修改的文件

1. **internal/repository/sqlite.go**
   - 添加 `runMigrations()` 方法
   - 添加 `migrateResourceTimeIsManual()` 方法
   - 在 `InitSchema()` 中调用迁移

2. **migrations/add_resource_time_is_manual.sql**
   - 手动迁移脚本（备选方案）

3. **DATABASE_FIX_RESOURCE_TIME.md**
   - 本文档

### 相关模型

- **internal/models/models.go**
  - `TVShow.ResourceTimeIsManual` 字段定义

## 🎯 后续优化建议

### 1. 实现完整的迁移系统

```go
type Migration struct {
    Version     int
    Description string
    Up          string
    Down        string
}

var migrations = []Migration{
    {
        Version:     1,
        Description: "Add resource_time_is_manual column",
        Up:          "...",
        Down:        "...",
    },
}
```

### 2. 添加迁移日志

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 3. 自动回滚机制

如果迁移失败，自动回滚到之前的状态。

## ❓ 常见问题

### Q1: 迁移会丢失数据吗？

**A**: 不会。迁移只是添加新列，所有现有数据都会保留。

### Q2: 需要重启应用吗？

**A**: 是的。迁移在应用启动时执行，需要重启才能生效。

### Q3: 可以回滚迁移吗？

**A**: 可以。如果需要回滚，可以手动删除 `resource_time_is_manual` 列。

### Q4: 为什么不在建表时就包含此列？

**A**: 这是后期添加的功能，旧数据库没有此列。

### Q5: 迁移需要多长时间？

**A**: 通常 < 1 秒，取决于数据量。

## 📞 获取帮助

如果遇到问题：

1. 查看应用日志：`docker compose logs tv-tracker`
2. 检查数据库结构：`sqlite3 data/tv_tracker.db ".schema tv_shows"`
3. 提交 Issue：[GitHub Issues](https://github.com/xc9973/tv-tracker/issues)

## ✅ 修复确认清单

- [ ] 重启应用
- [ ] 检查日志确认迁移成功
- [ ] 测试 `/api/library` 接口
- [ ] 测试 Web 界面"我的片库"页面
- [ ] 验证数据完整性

---

**最后更新**: 2026-01-09
**版本**: 1.0.0