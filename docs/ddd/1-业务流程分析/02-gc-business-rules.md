# GC核心业务规则清单

## 1. 规则分类与编号
规则格式：`GC-BR-[类别]-[序号]`，如 `GC-BR-PERF-001`

## 2. 性能规则

### GC-BR-PERF-001：暂停时间控制
- **描述**：单次GC暂停时间不得超过目标值
- **触发点**：GC执行过程中
- **规则逻辑**：
  ```
  IF gc_pause_time > target_pause_ms THEN
      enable_incremental_mode()
  ```
- **例外**：紧急GC情况下允许暂停时间延长至max_pause_ms

### GC-BR-PERF-002：吞吐量保障
- **描述**：GC开销不得超过应用程序吞吐量的5%
- **公式**：`吞吐量损失 = (gc_cpu_time / total_cpu_time) * 100%`
- **触发点**：GC周期结束时
- **规则逻辑**：`IF throughput_loss > 5% THEN reduce_gc_frequency()`

### GC-BR-PERF-003：内存开销限制
- **描述**：GC元数据开销不得超过堆总大小的20%
- **触发点**：堆大小调整时
- **规则逻辑**：`IF gc_overhead > 20% THEN trigger_compaction()`

## 3. 内存管理规则

### GC-BR-MEM-001：安全回收保证
- **描述**：只回收不可达对象，绝对禁止回收可达对象
- **触发点**：标记阶段
- **验证方法**：三色不变性检查
- **后果**：违反此规则将导致内存损坏

### GC-BR-MEM-002：堆大小动态调整
- **描述**：根据分配压力动态调整堆大小
- **公式**：
  ```
  target_heap_size = current_heap_size *
                     (allocation_rate / gc_throughput) *
                     safety_factor
  ```
- **触发点**：GC周期结束时

### GC-BR-MEM-003：大对象处理
- **描述**：超过32KB的对象直接分配到大对象区
- **触发点**：对象分配时
- **规则逻辑**：
  ```
  IF object_size > 32KB THEN
      allocate_in_large_object_space()
  ```

## 4. 并发安全规则

### GC-BR-CONC-001：写屏障一致性
- **描述**：写屏障必须保证三色不变性
- **触发点**：对象引用修改时
- **规则逻辑**：
  ```
  IF gc_phase == MARKING AND new_ref != NULL THEN
      mark_gray(new_ref)
  ```

### GC-BR-CONC-002：STW窗口最小化
- **描述**：Stop-The-World时间不得超过总GC时间的10%
- **触发点**：STW阶段
- **监控指标**：stw_duration / total_gc_duration < 0.1

## 5. 自适应规则

### GC-BR-ADAPT-001：工作负载检测
- **描述**：根据对象寿命分布识别应用类型
- **触发点**：GC周期结束时
- **规则逻辑**：
  ```
  IF short_lived_ratio > 0.7 THEN workload = TRANSACTIONAL
  IF long_lived_ratio > 0.6 THEN workload = CACHE_SERVER
  IF medium_lived_ratio > 0.5 THEN workload = BATCH_PROCESSING
  ```

### GC-BR-ADAPT-002：参数动态调整
- **描述**：使用PID控制器调整GC参数
- **控制目标**：暂停时间、堆使用率、GC频率
- **调整周期**：每秒检查一次

## 6. 模块化规则

### GC-BR-MOD-001：模块兼容性
- **描述**：增强模块必须与核心GC兼容
- **触发点**：模块加载时
- **验证方法**：接口契约检查、功能测试

### GC-BR-MOD-002：按需加载
- **描述**：增强模块只在需要时加载
- **触发点**：GC初始化时
- **规则逻辑**：
  ```
  IF heap_size > 1GB THEN enable_generational()
  IF target_pause < 10ms THEN enable_incremental()
  ```

## 7. 监控与诊断规则

### GC-BR-MON-001：指标完整性
- **描述**：必须收集所有关键性能指标
- **必需指标**：
  - 暂停时间分布（P50、P95、P99）
  - 堆使用率变化
  - 分配和回收速率
  - CPU开销比例

### GC-BR-MON-002：异常检测
- **描述**：自动检测GC异常情况
- **检测规则**：
  ```
  IF pause_time_p99 > 100ms THEN alert("High GC pause")
  IF heap_fragmentation > 0.5 THEN alert("High fragmentation")
  IF gc_cpu_percent > 30% THEN alert("High GC overhead")
  ```

## 8. 配置管理规则

### GC-BR-CONF-001：配置验证
- **描述**：启动时验证所有配置参数的合理性
- **验证规则**：
  ```
  0 < gc_trigger_ratio < 1
  target_pause_ms > 0
  initial_heap_size <= max_heap_size
  ```

### GC-BR-CONF-002：运行时配置
- **描述**：支持运行时动态调整部分配置
- **可调整参数**：触发阈值、并发度、模块开关
- **安全机制**：渐进式调整、回滚能力
