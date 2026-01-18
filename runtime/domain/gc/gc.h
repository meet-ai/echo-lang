#ifndef ECHO_RUNTIME_GC_H
#define ECHO_RUNTIME_GC_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// 前向声明
typedef struct heap_t heap_t;
typedef struct object_t object_t;
typedef struct gc_collector_t gc_collector_t;
typedef struct mark_bitmap_t mark_bitmap_t;
typedef struct allocator_t allocator_t;
typedef struct gc_cycle_t gc_cycle_t;
typedef struct mark_phase_t mark_phase_t;
typedef struct sweep_phase_t sweep_phase_t;
typedef struct write_barrier_t write_barrier_t;
typedef struct adaptive_decision_t adaptive_decision_t;
typedef struct gc_module_t gc_module_t;

// =============================================================================
// 值对象定义 (Value Objects)
// =============================================================================

// GC配置值对象
typedef struct {
    double trigger_threshold;    // 触发阈值 (0.0-1.0)
    int pause_target_ms;         // 目标暂停时间 (ms)
    size_t heap_limit_mb;        // 堆大小限制 (MB)
    bool concurrent_mark;        // 是否并发标记
    bool concurrent_sweep;       // 是否并发清扫
} gc_config_t;

// 工作负载特征值对象
typedef struct {
    double allocation_rate_mbps;     // 分配率 (MB/s)
    double object_lifetime_avg_sec;  // 对象平均寿命 (秒)
    int concurrency_level;           // 并发度
    double mutation_rate;            // 变异率 (0.0-1.0)
    size_t young_object_ratio;       // 年轻对象比例 (0-100)
} workload_profile_t;

// 性能指标值对象
typedef struct {
    int pause_time_p95_ms;       // P95暂停时间 (ms)
    int pause_time_max_ms;       // 最大暂停时间 (ms)
    double throughput_loss_pct;  // 吞吐量损失百分比
    size_t memory_overhead_mb;   // 内存开销 (MB)
    double gc_cpu_usage_pct;     // GC CPU使用率
    size_t promoted_bytes_mb;    // 晋升字节数 (MB)
} performance_metrics_t;

// 规则条件值对象
typedef struct {
    const char* operator;        // 运算符 ("gt", "lt", "eq", "ge", "le")
    double threshold;            // 阈值
    const char* metric_type;     // 指标类型 ("pause_time", "throughput", "memory")
    const char* unit;            // 单位 ("ms", "pct", "mb")
} rule_condition_t;

// PID参数值对象
typedef struct {
    double kp;                   // 比例系数
    double ki;                   // 积分系数
    double kd;                   // 微分系数
    double output_min;           // 输出最小值
    double output_max;           // 输出最大值
    double integral_limit;       // 积分限幅
} pid_parameters_t;

// GC统计值对象
typedef struct {
    size_t objects_collected;    // 回收对象数
    size_t bytes_collected;      // 回收字节数
    int64_t collection_time_ns;  // 回收耗时 (纳秒)
    size_t heap_size_before_mb;  // 回收前堆大小 (MB)
    size_t heap_size_after_mb;   // 回收后堆大小 (MB)
    double fragmentation_pct;    // 碎片率百分比
} gc_statistics_t;

// 内存布局值对象
typedef struct {
    size_t heap_size_mb;         // 堆总大小 (MB)
    size_t used_size_mb;         // 已使用大小 (MB)
    size_t free_size_mb;         // 空闲大小 (MB)
    double fragmentation_pct;    // 碎片率百分比
    size_t largest_free_block_mb; // 最大空闲块 (MB)
    int num_free_blocks;         // 空闲块数量
} memory_layout_t;

// 分配统计值对象
typedef struct {
    double allocation_rate_mbps; // 分配率 (MB/s)
    size_t avg_object_size_bytes; // 平均对象大小 (bytes)
    size_t small_objects_count;   // 小对象数量 (<1KB)
    size_t medium_objects_count;  // 中等对象数量 (1KB-1MB)
    size_t large_objects_count;   // 大对象数量 (>1MB)
    double survival_rate_pct;     // 幸存率百分比
} allocation_stats_t;

// 内存地址值对象
typedef struct {
    uintptr_t address;           // 内存地址
    const char* region;          // 内存区域 ("heap", "stack", "mmap")
} memory_address_t;

// =============================================================================
// 实体接口定义 (Entity Interfaces)
// =============================================================================

// 堆实体接口
typedef struct heap_t heap_t;

heap_t* heap_create(size_t initial_size_mb);
void heap_destroy(heap_t* heap);
uint64_t heap_get_id(const heap_t* heap);
size_t heap_get_size_mb(const heap_t* heap);
size_t heap_get_used_size_mb(const heap_t* heap);
size_t heap_get_free_size_mb(const heap_t* heap);
double heap_get_fragmentation_pct(const heap_t* heap);
void* heap_allocate(heap_t* heap, size_t size);
void heap_deallocate(heap_t* heap, void* ptr);
memory_layout_t heap_get_layout(const heap_t* heap);

// 对象实体接口
typedef struct object_t object_t;

uint64_t object_get_id(const object_t* object);
size_t object_get_size(const object_t* object);
memory_address_t object_get_address(const object_t* object);
bool object_is_marked(const object_t* object);
void object_mark(object_t* object);
void object_unmark(object_t* object);
uint64_t object_get_type_id(const object_t* object);
const char* object_get_type_name(const object_t* object);

// 垃圾回收器实体接口
typedef struct gc_collector_t gc_collector_t;

gc_collector_t* gc_collector_create(const gc_config_t* config);
void gc_collector_destroy(gc_collector_t* collector);
uint64_t gc_collector_get_id(const gc_collector_t* collector);
const gc_config_t* gc_collector_get_config(const gc_collector_t* collector);
void gc_collector_set_config(gc_collector_t* collector, const gc_config_t* config);
void gc_collector_trigger_collection(gc_collector_t* collector);
bool gc_collector_is_collecting(const gc_collector_t* collector);
performance_metrics_t gc_collector_get_metrics(const gc_collector_t* collector);
gc_statistics_t gc_collector_get_last_stats(const gc_collector_t* collector);

// 标记位图实体接口
typedef struct mark_bitmap_t mark_bitmap_t;

mark_bitmap_t* mark_bitmap_create(size_t heap_size_mb);
void mark_bitmap_destroy(mark_bitmap_t* bitmap);
uint64_t mark_bitmap_get_id(const mark_bitmap_t* bitmap);
size_t mark_bitmap_get_size_bytes(const mark_bitmap_t* bitmap);
bool mark_bitmap_is_marked(const mark_bitmap_t* bitmap, memory_address_t address);
void mark_bitmap_mark(mark_bitmap_t* bitmap, memory_address_t address);
void mark_bitmap_clear(mark_bitmap_t* bitmap, memory_address_t address);
void mark_bitmap_clear_all(mark_bitmap_t* bitmap);
size_t mark_bitmap_count_marked(const mark_bitmap_t* bitmap);

// 分配器实体接口
typedef struct allocator_t allocator_t;

allocator_t* allocator_create(heap_t* heap);
void allocator_destroy(allocator_t* allocator);
uint64_t allocator_get_id(const allocator_t* allocator);
uint64_t allocator_get_heap_id(const allocator_t* allocator);
void* allocator_allocate(allocator_t* allocator, size_t size, const char* type_name);
void allocator_free(allocator_t* allocator, void* ptr);
allocation_stats_t allocator_get_stats(const allocator_t* allocator);
void allocator_reset_stats(allocator_t* allocator);

// GC周期实体接口
typedef struct gc_cycle_t gc_cycle_t;

gc_cycle_t* gc_cycle_create(uint64_t collector_id, int cycle_number);
void gc_cycle_destroy(gc_cycle_t* cycle);
uint64_t gc_cycle_get_id(const gc_cycle_t* cycle);
uint64_t gc_cycle_get_collector_id(const gc_cycle_t* cycle);
int gc_cycle_get_number(const gc_cycle_t* cycle);
int64_t gc_cycle_get_start_time_ns(const gc_cycle_t* cycle);
int64_t gc_cycle_get_end_time_ns(const gc_cycle_t* cycle);
gc_statistics_t gc_cycle_get_statistics(const gc_cycle_t* cycle);
void gc_cycle_complete(gc_cycle_t* cycle, const gc_statistics_t* stats);

// 标记阶段实体接口
typedef struct mark_phase_t mark_phase_t;

mark_phase_t* mark_phase_create(uint64_t cycle_id);
void mark_phase_destroy(mark_phase_t* phase);
uint64_t mark_phase_get_id(const mark_phase_t* phase);
uint64_t mark_phase_get_cycle_id(const mark_phase_t* phase);
bool mark_phase_is_concurrent(const mark_phase_t* phase);
void mark_phase_start(mark_phase_t* phase);
void mark_phase_complete(mark_phase_t* phase);
int64_t mark_phase_get_duration_ns(const mark_phase_t* phase);
size_t mark_phase_get_objects_marked(const mark_phase_t* phase);

// 清扫阶段实体接口
typedef struct sweep_phase_t sweep_phase_t;

sweep_phase_t* sweep_phase_create(uint64_t cycle_id);
void sweep_phase_destroy(sweep_phase_t* phase);
uint64_t sweep_phase_get_id(const sweep_phase_t* phase);
uint64_t sweep_phase_get_cycle_id(const sweep_phase_t* phase);
bool sweep_phase_is_concurrent(const sweep_phase_t* phase);
void sweep_phase_start(sweep_phase_t* phase);
void sweep_phase_complete(sweep_phase_t* phase);
int64_t sweep_phase_get_duration_ns(const sweep_phase_t* phase);
size_t sweep_phase_get_bytes_collected(const sweep_phase_t* phase);

// 写屏障实体接口
typedef struct write_barrier_t write_barrier_t;

write_barrier_t* write_barrier_create(void);
void write_barrier_destroy(write_barrier_t* barrier);
uint64_t write_barrier_get_id(const write_barrier_t* barrier);
void write_barrier_install(write_barrier_t* barrier);
void write_barrier_uninstall(write_barrier_t* barrier);
bool write_barrier_is_active(const write_barrier_t* barrier);
size_t write_barrier_get_triggers_count(const write_barrier_t* barrier);
void write_barrier_on_write_reference(write_barrier_t* barrier, object_t* from, object_t* to);
void write_barrier_on_write_value(write_barrier_t* barrier, object_t* object, void* value);

// 自适应决策实体接口
typedef struct adaptive_decision_t adaptive_decision_t;

adaptive_decision_t* adaptive_decision_create(const pid_parameters_t* pid_params);
void adaptive_decision_destroy(adaptive_decision_t* decision);
uint64_t adaptive_decision_get_id(const adaptive_decision_t* decision);
const pid_parameters_t* adaptive_decision_get_pid_params(const adaptive_decision_t* decision);
void adaptive_decision_update_pid_params(adaptive_decision_t* decision, const pid_parameters_t* params);
gc_config_t adaptive_decision_make_decision(
    adaptive_decision_t* decision,
    const workload_profile_t* workload,
    const performance_metrics_t* metrics
);
void adaptive_decision_reset(adaptive_decision_t* decision);

// GC模块实体接口
typedef struct gc_module_t gc_module_t;

gc_module_t* gc_module_create(const char* module_type, const char* config);
void gc_module_destroy(gc_module_t* module);
uint64_t gc_module_get_id(const gc_module_t* module);
const char* gc_module_get_type(const gc_module_t* module);
const char* gc_module_get_config(const gc_module_t* module);
bool gc_module_is_enabled(const gc_module_t* module);
void gc_module_enable(gc_module_t* module);
void gc_module_disable(gc_module_t* module);
bool gc_module_supports_feature(const gc_module_t* module, const char* feature);

// =============================================================================
// 领域服务接口定义 (Domain Services)
// =============================================================================

// GC控制器服务接口
typedef struct gc_controller_service_t gc_controller_service_t;

gc_controller_service_t* gc_controller_service_create(gc_collector_t* collector);
void gc_controller_service_destroy(gc_controller_service_t* service);
void gc_controller_service_start(gc_controller_service_t* service);
void gc_controller_service_stop(gc_controller_service_t* service);
void gc_controller_service_trigger_collection(gc_controller_service_t* service);
void gc_controller_service_set_config(gc_controller_service_t* service, const gc_config_t* config);
gc_config_t gc_controller_service_get_config(const gc_controller_service_t* service);
bool gc_controller_service_is_running(const gc_controller_service_t* service);

// 自适应GC服务接口
typedef struct adaptive_gc_service_t adaptive_gc_service_t;

adaptive_gc_service_t* adaptive_gc_service_create(
    gc_controller_service_t* controller,
    adaptive_decision_t* decision_maker
);
void adaptive_gc_service_destroy(adaptive_gc_service_t* service);
void adaptive_gc_service_start_monitoring(adaptive_gc_service_t* service);
void adaptive_gc_service_stop_monitoring(adaptive_gc_service_t* service);
void adaptive_gc_service_update_workload(adaptive_gc_service_t* service, const workload_profile_t* workload);
workload_profile_t adaptive_gc_service_analyze_workload(adaptive_gc_service_t* service);
void adaptive_gc_service_adapt_config(adaptive_gc_service_t* service);
bool adaptive_gc_service_should_trigger_collection(adaptive_gc_service_t* service);

// GC模块管理服务接口
typedef struct gc_module_manager_service_t gc_module_manager_service_t;

gc_module_manager_service_t* gc_module_manager_service_create(void);
void gc_module_manager_service_destroy(gc_module_manager_service_t* service);
void gc_module_manager_service_register_module(gc_module_manager_service_t* service, gc_module_t* module);
void gc_module_manager_service_unregister_module(gc_module_manager_service_t* service, uint64_t module_id);
gc_module_t* gc_module_manager_service_get_module(gc_module_manager_service_t* service, uint64_t module_id);
void gc_module_manager_service_enable_module(gc_module_manager_service_t* service, uint64_t module_id);
void gc_module_manager_service_disable_module(gc_module_manager_service_t* service, uint64_t module_id);
size_t gc_module_manager_service_get_enabled_modules_count(const gc_module_manager_service_t* service);

// 规则引擎服务接口
typedef struct rule_engine_service_t rule_engine_service_t;

rule_engine_service_t* rule_engine_service_create(void);
void rule_engine_service_destroy(rule_engine_service_t* service);
void rule_engine_service_add_rule(rule_engine_service_t* service, const rule_condition_t* condition, const char* action);
void rule_engine_service_remove_rule(rule_engine_service_t* service, uint64_t rule_id);
bool rule_engine_service_evaluate_condition(
    const rule_engine_service_t* service,
    const rule_condition_t* condition,
    double current_value
);
const char* rule_engine_service_get_action_for_condition(
    const rule_engine_service_t* service,
    const rule_condition_t* condition,
    double current_value
);

// 内存管理服务接口
typedef struct memory_manager_service_t memory_manager_service_t;

memory_manager_service_t* memory_manager_service_create(heap_t* heap, allocator_t* allocator);
void memory_manager_service_destroy(memory_manager_service_t* service);
void* memory_manager_service_allocate(memory_manager_service_t* service, size_t size, const char* type_name);
void memory_manager_service_deallocate(memory_manager_service_t* service, void* ptr);
memory_layout_t memory_manager_service_get_layout(const memory_manager_service_t* service);
allocation_stats_t memory_manager_service_get_allocation_stats(const memory_manager_service_t* service);
void memory_manager_service_run_gc(memory_manager_service_t* service);
bool memory_manager_service_should_run_gc(const memory_manager_service_t* service);

// =============================================================================
// 仓储接口定义 (Repository Interfaces)
// =============================================================================

// 堆仓储接口
typedef struct heap_repository_t heap_repository_t;

heap_repository_t* heap_repository_create(void);
void heap_repository_destroy(heap_repository_t* repository);
void heap_repository_save(heap_repository_t* repository, const heap_t* heap);
heap_t* heap_repository_find_by_id(heap_repository_t* repository, uint64_t id);
void heap_repository_delete(heap_repository_t* repository, uint64_t id);

// 对象仓储接口
typedef struct object_repository_t object_repository_t;

object_repository_t* object_repository_create(void);
void object_repository_destroy(object_repository_t* repository);
void object_repository_save(object_repository_t* repository, const object_t* object);
object_t* object_repository_find_by_id(object_repository_t* repository, uint64_t id);
object_t* object_repository_find_by_address(object_repository_t* repository, memory_address_t address);
void object_repository_delete(object_repository_t* repository, uint64_t id);
size_t object_repository_count_marked(const object_repository_t* repository);

// GC周期仓储接口
typedef struct gc_cycle_repository_t gc_cycle_repository_t;

gc_cycle_repository_t* gc_cycle_repository_create(void);
void gc_cycle_repository_destroy(gc_cycle_repository_t* repository);
void gc_cycle_repository_save(gc_cycle_repository_t* repository, const gc_cycle_t* cycle);
gc_cycle_t* gc_cycle_repository_find_by_id(gc_cycle_repository_t* repository, uint64_t id);
void gc_cycle_repository_delete_old_cycles(gc_cycle_repository_t* repository, int64_t older_than_ns);
size_t gc_cycle_repository_count_cycles(const gc_cycle_repository_t* repository, uint64_t collector_id);

#endif // ECHO_RUNTIME_GC_H
