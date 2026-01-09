# 任务：代码审查问题修复

## 目标
根据代码审查结论修复后端逻辑与一致性问题，提升数据一致性与时间判断准确性。

## 问题清单
1. `TaskBoardService.CompleteTask` 与 `PostponeTask` 缺少事务，可能导致任务完成与归档、或延期重建不一致。
2. `TaskRepository.GetByShowAndEpisode` 使用 `LIKE %SxxExx%` 可能误匹配（如 `S01E1` 匹配 `S01E10`）。
3. 任务生成中使用 `time.Parse`（UTC）与本地 `timeutil.Now()` 对比，存在时区误差。
4. `GetTodayEpisodes` 使用字符串哨兵值判断日期参数，存在极端覆盖风险。

## 计划修改
- 为任务完成/延期引入事务（`TaskRepository` 新增事务支持与原子接口）。
- 为 UPDATE 任务描述引入稳定前缀（如 `SxxExx|`），并提供精确匹配查询。
- 统一日期解析时区，使用本地时区解析 TMDB 的 `YYYY-MM-DD`。
- 简化今日日期参数处理逻辑。

## 预期影响范围
- `internal/service/task_board.go`
- `internal/repository/task_repository.go`
- `internal/service/task_generator.go`
- `internal/handler/http.go`
- `tests/property/task_board_test.go`（必要时新增测试）

## 验收标准
- 完成/延期任务操作具备事务一致性。
- 不再出现 `S01E1` 与 `S01E10` 误匹配导致的漏任务。
- 跨时区情况下“今日/已播出”判断一致。
- `/api/today` 日期参数处理逻辑清晰无哨兵。

## 创建时间
- 2026-01-09
