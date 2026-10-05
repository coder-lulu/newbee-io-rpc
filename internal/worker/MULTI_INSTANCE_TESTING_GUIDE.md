# Week 1 多实例集成测试指南

## 测试目标

验证分布式锁和任务去重机制在多实例环境下的正确性：
1. ✅ CronScheduler在多实例下只触发一次InputTask
2. ✅ TaskWorker不会重复处理同一个InputTask
3. ✅ 锁过期后能正确接管
4. ✅ 测量Redis锁对性能的影响

## 环境准备

### 1. Redis服务
```bash
# 启动Redis（如果尚未运行）
docker run -d --name redis-test -p 6379:6379 redis:7-alpine

# 或使用现有Redis
# 确保配置文件中Redis地址正确
```

### 2. 数据库准备
```bash
# 确保数据库已初始化
# 创建测试用的CronTask
```

### 3. 配置文件
创建3个配置文件用于3个实例：

**etc/io-instance1.yaml**:
```yaml
Name: unified-io-rpc-instance1
ListenOn: 0.0.0.0:9001

DatabaseConf:
  Type: mysql
  Path: "root:password@tcp(localhost:3306)/unified_io?parseTime=true"

RedisConf:
  Host: localhost:6379
  Type: node

TaskWorker:
  Enabled: true
  PullInterval: 5s
  BatchSize: 10
  MaxConcurrent: 5
```

**etc/io-instance2.yaml** (端口改为9002)
**etc/io-instance3.yaml** (端口改为9003)

## 测试场景

### 场景1: CronScheduler多实例去重测试

**目的**: 验证多个实例中只有一个会为同一个CronTask创建InputTask

**步骤**:
```bash
# Terminal 1: 启动实例1
cd /opt/code/newbee/unified-io/rpc
go run io.go -f etc/io-instance1.yaml

# Terminal 2: 启动实例2
go run io.go -f etc/io-instance2.yaml

# Terminal 3: 启动实例3
go run io.go -f etc/io-instance3.yaml

# Terminal 4: 创建一个每分钟触发的CronTask
mysql -u root -p unified_io -e "
INSERT INTO input_cron_tasks (
    task_name, cron_expression, input_source,
    source_config, status, tenant_id
) VALUES (
    'Multi-Instance Test Task',
    '*/1 * * * *',  -- 每分钟触发
    'mysql',
    '{\"query\":\"SELECT 1\"}',
    1,
    1
);
"
```

**验证**:
1. 观察3个实例的日志
2. 每分钟应该只有**一个**实例打印：
   ```
   Lock acquired successfully, executing cron task
   Creating InputTask for CronTask ID=X
   ```
3. 另外两个实例应该打印：
   ```
   Lock is held by another instance, skipping execution
   ```

**检查数据库**:
```sql
-- 查看生成的InputTask，每分钟应该只有一个
SELECT id, task_name, cron_task_id, created_at
FROM input_tasks
WHERE cron_task_id = <刚创建的CronTask ID>
ORDER BY created_at DESC;
```

### 场景2: TaskWorker任务去重测试

**目的**: 验证多个Worker不会重复处理同一个InputTask

**步骤**:
```bash
# 1. 确保3个实例都在运行
# 2. 批量创建测试任务
mysql -u root -p unified_io -e "
INSERT INTO input_tasks (
    task_name, input_source, source_config,
    task_status, scheduled_at, tenant_id
)
SELECT
    CONCAT('Batch Test Task ', n) as task_name,
    'mysql' as input_source,
    '{\"query\":\"SELECT 1\"}' as source_config,
    'pending' as task_status,
    NOW() as scheduled_at,
    1 as tenant_id
FROM (
    SELECT @row := @row + 1 as n
    FROM (SELECT 0 UNION SELECT 1 UNION SELECT 2 UNION SELECT 3) t1,
         (SELECT 0 UNION SELECT 1 UNION SELECT 2 UNION SELECT 3) t2,
         (SELECT @row:=0) r
    LIMIT 50
) numbers;
"
```

**验证**:
1. 观察3个实例的日志
2. 每个任务应该只被一个实例处理
3. 检查日志中的"Task already being processed, skipping"消息

**性能验证**:
```sql
-- 所有任务应该在合理时间内完成（考虑BatchSize=10, MaxConcurrent=5）
SELECT
    task_status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, scheduled_at, completed_at)) as avg_duration_sec
FROM input_tasks
WHERE task_name LIKE 'Batch Test Task %'
GROUP BY task_status;
```

### 场景3: 锁过期接管测试

**目的**: 验证锁过期后其他实例能接管

**步骤**:
```bash
# 1. 修改锁过期时间为5秒（仅用于测试）
# 在distributed_lock.go的CronScheduler锁获取代码中：
# lockExpiry := 5 * time.Second  // 原来是30秒

# 2. 启动实例1
go run io.go -f etc/io-instance1.yaml

# 3. 在实例1获取锁后，立即kill掉进程
# Terminal 1执行：
pkill -f "io.go -f etc/io-instance1.yaml"

# 4. 观察实例2和3的行为
# 应该在5秒后有一个实例获取到锁并继续执行
```

**验证**:
- 查看日志，确认在5秒左右有另一个实例打印：
  ```
  Lock acquired successfully (after previous holder expired)
  ```

### 场景4: 性能影响测试

**目的**: 测量Redis分布式锁对性能的影响

**基准测试（无锁）**:
```bash
# 1. 临时禁用分布式锁
# 在service_context.go中：
# taskWorker = worker.NewTaskWorker(db, workerConfig, nil)  // 传nil禁用锁

# 2. 运行性能测试
time mysql -u root -p unified_io -e "
-- 插入1000个任务
-- 测量处理时间
"
```

**对比测试（有锁）**:
```bash
# 1. 启用分布式锁
# 恢复原始代码

# 2. 运行相同的性能测试
# 3. 对比处理时间
```

**预期结果**:
- 单实例：锁的开销应该 < 5%
- 多实例：总吞吐量应该接近线性扩展（3实例 ≈ 3倍吞吐量）

## 监控指标

### Redis监控
```bash
# 实时查看Redis命令
redis-cli MONITOR | grep -E "SETNX|DEL|EXPIRE"

# 查看锁相关的key
redis-cli KEYS "cron:task:*"
redis-cli KEYS "task:process:*"
```

### 数据库监控
```sql
-- 实时查看任务处理情况
SELECT
    task_status,
    COUNT(*) as count,
    MAX(updated_at) as last_update
FROM input_tasks
GROUP BY task_status;

-- 查看重复处理（不应该有）
SELECT
    task_id,
    COUNT(*) as process_count
FROM (
    SELECT id as task_id FROM input_tasks WHERE task_status = 'completed'
) t
GROUP BY task_id
HAVING COUNT(*) > 1;
```

### 日志分析
```bash
# 统计每个实例获取锁的次数
grep "Lock acquired successfully" instance*.log | wc -l

# 统计锁被跳过的次数
grep "Lock is held by another instance" instance*.log | wc -l

# 查看是否有错误
grep -i "error\|failed" instance*.log
```

## 成功标准

### ✅ CronScheduler去重
- [ ] 3个实例运行时，每个Cron触发只生成1个InputTask
- [ ] 日志显示只有1个实例获取到锁
- [ ] 无重复的InputTask创建

### ✅ TaskWorker去重
- [ ] 50个任务分布在3个实例处理
- [ ] 无任务被重复处理
- [ ] 所有任务都被处理完成

### ✅ 锁过期接管
- [ ] 实例崩溃后，其他实例能在锁过期时间内接管
- [ ] 无任务丢失

### ✅ 性能影响
- [ ] 单实例性能下降 < 5%
- [ ] 3实例吞吐量 > 2.5倍单实例

## 故障排查

### 问题1: 多个实例都获取到了锁
**可能原因**:
- Redis未正确配置
- 时钟不同步
- 锁key命名冲突

**解决方案**:
```bash
# 检查Redis连接
redis-cli PING

# 检查锁key
redis-cli KEYS "cron:task:*"

# 检查系统时间
date
```

### 问题2: 任务未被处理
**可能原因**:
- 所有实例都获取锁失败
- TaskWorker未启动
- 数据库连接问题

**解决方案**:
```bash
# 检查TaskWorker状态
grep "TaskWorker initialized" instance*.log

# 检查数据库连接
mysql -u root -p unified_io -e "SELECT 1"
```

### 问题3: 性能下降严重
**可能原因**:
- Redis响应慢
- 锁竞争过多
- 网络延迟高

**解决方案**:
```bash
# 测试Redis延迟
redis-cli --latency

# 减少锁竞争：增加BatchSize
# 优化锁过期时间
```

## 下一步

测试通过后：
1. ✅ 确认Week 1 P0任务完成
2. 📝 记录测试结果和性能数据
3. 🚀 进入Week 2：配置中心建设

## 参考

- 分布式锁实现: `/opt/code/newbee/unified-io/rpc/internal/lock/distributed_lock.go`
- CronScheduler集成: `/opt/code/newbee/unified-io/rpc/internal/worker/cron_scheduler.go:217-248`
- TaskWorker去重: `/opt/code/newbee/unified-io/rpc/internal/worker/task_worker.go:227-248`
- 单元测试: `/opt/code/newbee/unified-io/rpc/internal/lock/distributed_lock_test.go`
