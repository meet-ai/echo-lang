/**
 * @file task.h
 * @brief TaskExecution 聚合根定义
 *
 * Task 是任务执行聚合的聚合根，负责维护任务执行的一致性和生命周期。
 * 
 * 使用场景：
 * - 创建任务：通过 task_factory_create() 创建新任务
 * - 启动任务：调用 task_start() 开始执行
 * - 等待Future：调用 task_wait_for_future() 等待异步操作完成
 * - 处理Future完成：调用 task_handle_future_resolved() 处理Future结果
 * 
 * 示例：
 * ```c
 * // 创建任务
 * Task* task = task_factory_create(entry_func, arg, stack_size);
 * 
 * // 启动任务
 * task_start(task);
 * 
 * // 等待Future完成
 * task_wait_for_future(task, future_id);
 * 
 * // 处理Future结果
 * task_handle_future_resolved(task, future_id, result);
 * ```
 */

#ifndef TASK_EXECUTION_AGGREGATE_TASK_H
#define TASK_EXECUTION_AGGREGATE_TASK_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include <time.h>

// 前向声明
struct Coroutine;
struct DomainEvent;

// ============================================================================
// 值对象定义
// ============================================================================

// 任务状态枚举（值对象）
typedef enum {
    TASK_STATUS_CREATED,    // 已创建
    TASK_STATUS_READY,      // 准备执行
    TASK_STATUS_RUNNING,    // 正在执行
    TASK_STATUS_WAITING,    // 等待异步操作完成
    TASK_STATUS_COMPLETED,  // 执行完成
    TASK_STATUS_FAILED,     // 执行失败
    TASK_STATUS_CANCELLED   // 被取消
} TaskStatus;

// 任务优先级枚举（值对象）
typedef enum {
    TASK_PRIORITY_LOW,
    TASK_PRIORITY_NORMAL,
    TASK_PRIORITY_HIGH,
    TASK_PRIORITY_CRITICAL
} TaskPriority;

// 任务ID类型（值对象）
typedef uint64_t TaskID;

// TaskID列表节点（值对象，用于实现TaskID队列）
typedef struct TaskIDNode {
    TaskID task_id;              // 任务ID
    struct TaskIDNode* next;     // 下一个节点
} TaskIDNode;

// FutureID类型（值对象，用于引用Future聚合）
typedef uint64_t FutureID;

// ============================================================================
// TaskExecution 聚合根定义
// ============================================================================

/**
 * @brief Task 聚合根结构体
 * 
 * 职责：
 * - 维护任务执行状态的一致性
 * - 管理Coroutine内部实体的生命周期
 * - 通过FutureID引用Future聚合（避免循环依赖）
 * - 发布领域事件（状态变化时）
 */
typedef struct Task {
    // ==================== 聚合根标识 ====================
    TaskID id;                      // 任务唯一标识
    
    // ==================== 状态值对象 ====================
    TaskStatus status;              // 任务状态
    TaskPriority priority;          // 任务优先级
    
    // ==================== 内部实体 ====================
    struct Coroutine* coroutine;    // 关联的协程（内部实体，通过聚合根访问）
    
    // ==================== 外部聚合引用（通过ID）====================
    FutureID waiting_future_id;     // 当前等待的Future ID（替代Future*指针）
    
    // ==================== 执行参数 ====================
    void (*entry_point)(void*);     // 任务入口函数
    void* arg;                      // 函数参数
    size_t stack_size;              // 栈大小
    
    // ==================== 执行结果 ====================
    void* result;                   // 执行结果
    int exit_code;                  // 退出代码
    char error_message[1024];       // 错误信息
    
    // ==================== 时间戳 ====================
    time_t created_at;              // 创建时间
    time_t started_at;              // 开始时间
    time_t completed_at;            // 完成时间
    
    // ==================== 同步机制 ====================
    pthread_mutex_t mutex;          // 任务状态保护
    pthread_cond_t cond;            // 条件变量，用于等待
    
    // ==================== 领域事件列表 ====================
    struct DomainEvent** domain_events;  // 待发布的领域事件列表
    size_t domain_events_count;          // 事件数量
    size_t domain_events_capacity;       // 事件容量
    
    // ==================== 依赖注入 ====================
    struct EventBus* event_bus;          // 事件总线（可选，如果为NULL则只存储到domain_events列表）
    
    // ==================== 队列管理（用于调度器）====================
    struct Task* next;              // 下一个任务指针（用于队列）
    
    // ==================== 回调函数 ====================
    void (*on_complete)(struct Task*);  // 完成回调
    void* user_data;                // 用户数据
} Task;

// ============================================================================
// 聚合根工厂方法
// ============================================================================

/**
 * @brief 创建新任务（工厂方法）
 * 
 * 业务规则：
 * - entry_point不能为NULL
 * - stack_size必须大于等于4096字节
 * - 创建的任务初始状态为TASK_STATUS_CREATED
 * 
 * @param entry_point 任务入口函数
 * @param arg 函数参数
 * @param stack_size 栈大小（最小4096字节）
 * @param event_bus 事件总线（可选，如果为NULL则只存储到domain_events列表）
 * @return 新创建的任务，失败返回NULL
 */
Task* task_factory_create(void (*entry_point)(void*), void* arg, size_t stack_size, struct EventBus* event_bus);

// ============================================================================
// 聚合根业务方法
// ============================================================================

/**
 * @brief 启动任务执行
 * 
 * 业务规则：
 * - 只有TASK_STATUS_CREATED或TASK_STATUS_READY状态的任务可以启动
 * - 启动后会创建Coroutine内部实体
 * - 状态转换为TASK_STATUS_RUNNING
 * - 发布TaskStarted事件
 * 
 * @param task 任务聚合根
 * @return 0成功，-1失败
 */
int task_start(Task* task);

/**
 * @brief 挂起任务
 * 
 * 业务规则：
 * - 只有TASK_STATUS_RUNNING状态的任务可以挂起
 * - 状态转换为TASK_STATUS_WAITING
 * 
 * @param task 任务聚合根
 * @return 0成功，-1失败
 */
int task_suspend(Task* task);

/**
 * @brief 等待Future完成
 * 
 * 业务规则：
 * - 只有TASK_STATUS_RUNNING状态的任务可以等待Future
 * - 设置waiting_future_id
 * - 状态转换为TASK_STATUS_WAITING
 * 
 * @param task 任务聚合根
 * @param future_id 要等待的Future ID
 * @return 0成功，-1失败
 */
int task_wait_for_future(Task* task, FutureID future_id);

/**
 * @brief 处理Future完成事件
 * 
 * 业务规则：
 * - 只有TASK_STATUS_WAITING状态的任务可以处理Future完成
 * - waiting_future_id必须匹配
 * - 状态转换为TASK_STATUS_RUNNING
 * 
 * @param task 任务聚合根
 * @param future_id 完成的Future ID
 * @param result Future结果
 * @return 0成功，-1失败
 */
int task_handle_future_resolved(Task* task, FutureID future_id, void* result);

/**
 * @brief 完成任务
 * 
 * 业务规则：
 * - 只有TASK_STATUS_RUNNING状态的任务可以完成
 * - 状态转换为TASK_STATUS_COMPLETED
 * - 发布TaskCompleted事件
 * 
 * @param task 任务聚合根
 * @param result 执行结果
 * @return 0成功，-1失败
 */
int task_complete(Task* task, void* result);

/**
 * @brief 任务执行失败
 * 
 * 业务规则：
 * - 只有TASK_STATUS_RUNNING状态的任务可以失败
 * - 状态转换为TASK_STATUS_FAILED
 * - 发布TaskFailed事件
 * 
 * @param task 任务聚合根
 * @param error_message 错误信息
 * @return 0成功，-1失败
 */
int task_fail(Task* task, const char* error_message);

/**
 * @brief 取消任务
 * 
 * 业务规则：
 * - 只有TASK_STATUS_CREATED、TASK_STATUS_READY或TASK_STATUS_WAITING状态的任务可以取消
 * - 状态转换为TASK_STATUS_CANCELLED
 * - 发布TaskCancelled事件
 * 
 * @param task 任务聚合根
 * @return 0成功，-1失败
 */
int task_cancel(Task* task);

// ============================================================================
// 不变条件验证
// ============================================================================

/**
 * @brief 验证聚合不变条件
 * 
 * 不变条件：
 * - IC1：如果status为TASK_STATUS_WAITING，waiting_future_id必须有效（非0）
 * - IC2：如果status为TASK_STATUS_RUNNING，coroutine必须存在
 * - IC3：如果status为TASK_STATUS_COMPLETED或TASK_STATUS_FAILED，completed_at必须有效
 * 
 * @param task 任务聚合根
 * @return true如果不变条件满足，false否则
 */
bool task_validate_invariants(const Task* task);

/**
 * @brief 验证状态转换是否合法
 * 
 * 业务规则：
 * - TASK_STATUS_CREATED -> TASK_STATUS_READY, TASK_STATUS_CANCELLED
 * - TASK_STATUS_READY -> TASK_STATUS_RUNNING, TASK_STATUS_CANCELLED
 * - TASK_STATUS_RUNNING -> TASK_STATUS_WAITING, TASK_STATUS_COMPLETED, TASK_STATUS_FAILED
 * - TASK_STATUS_WAITING -> TASK_STATUS_RUNNING, TASK_STATUS_CANCELLED
 * - TASK_STATUS_COMPLETED, TASK_STATUS_FAILED, TASK_STATUS_CANCELLED -> 终态，不能转换
 * 
 * @param task 任务聚合根
 * @param new_status 新状态
 * @return true如果转换合法，false否则
 */
bool task_is_valid_state_transition(const Task* task, TaskStatus new_status);

// ============================================================================
// 领域事件管理
// ============================================================================

/**
 * @brief 获取并清空领域事件列表
 * 
 * 使用场景：
 * - 应用服务在保存聚合后调用此方法获取事件
 * - 然后发布到事件总线
 * 
 * @param task 任务聚合根
 * @param events 输出参数，事件数组指针
 * @param count 输出参数，事件数量
 * @return 0成功，-1失败
 */
int task_get_domain_events(Task* task, struct DomainEvent*** events, size_t* count);

// ============================================================================
// 查询方法（只读访问）
// ============================================================================

/**
 * @brief 获取任务ID
 */
TaskID task_get_id(const Task* task);

/**
 * @brief 获取任务状态
 */
TaskStatus task_get_status(const Task* task);

/**
 * @brief 获取等待的Future ID
 */
FutureID task_get_waiting_future_id(const Task* task);

/**
 * @brief 获取关联的Coroutine（只读访问）
 *
 * 使用场景：
 * - 调度器需要访问Coroutine进行上下文切换
 * - 只读访问，不修改Coroutine状态
 *
 * 注意：Coroutine是TaskExecution聚合的内部实体，只能通过聚合根访问
 *
 * @param task 任务聚合根
 * @return struct Coroutine* Coroutine指针，如果不存在返回NULL
 */
const struct Coroutine* task_get_coroutine(const Task* task);

/**
 * @brief 检查任务是否完成
 */
bool task_is_complete(const Task* task);

/**
 * @brief 检查任务是否失败
 */
bool task_is_failed(const Task* task);

/**
 * @brief 检查任务是否已取消
 */
bool task_is_cancelled(const Task* task);

// ============================================================================
// 资源管理
// ============================================================================

/**
 * @brief 销毁任务聚合根
 * 
 * 业务规则：
 * - 销毁时会销毁Coroutine内部实体
 * - 清理所有资源
 * 
 * @param task 任务聚合根
 */
void task_destroy(Task* task);

#endif // TASK_EXECUTION_AGGREGATE_TASK_H
