/**
 * @file future.h
 * @brief AsyncComputation 聚合根定义
 *
 * Future 是异步计算聚合的聚合根，负责维护异步操作的一致性和生命周期。
 * 
 * 使用场景：
 * - 创建Future：通过 future_factory_create() 创建新Future
 * - 解决Future：调用 future_resolve() 设置成功结果
 * - 拒绝Future：调用 future_reject() 设置错误结果
 * - 添加等待者：调用 future_add_waiter() 添加等待的Task（通过TaskID）
 * - 唤醒等待者：调用 future_wake_waiters() 唤醒所有等待的Task
 * 
 * 示例：
 * ```c
 * // 创建Future
 * Future* future = future_factory_create();
 * 
 * // 添加等待者（通过TaskID，不直接持有Task*）
 * future_add_waiter(future, task_id);
 * 
 * // 解决Future
 * future_resolve(future, result);
 * 
 * // Future内部会通过TaskRepository查找Task并唤醒
 * ```
 */

#ifndef ASYNC_COMPUTATION_AGGREGATE_FUTURE_H
#define ASYNC_COMPUTATION_AGGREGATE_FUTURE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include <time.h>

// 前向声明
struct DomainEvent;

// ============================================================================
// 值对象定义
// ============================================================================

// Future状态枚举（值对象）
typedef enum {
    FUTURE_PENDING,   // 异步操作未完成
    FUTURE_RESOLVED,  // 异步操作已完成（成功）
    FUTURE_REJECTED   // 异步操作已完成（失败）
} FutureState;

// Future优先级枚举（值对象）
typedef enum {
    FUTURE_PRIORITY_LOW,
    FUTURE_PRIORITY_NORMAL,
    FUTURE_PRIORITY_HIGH,
    FUTURE_PRIORITY_CRITICAL
} FuturePriority;

// FutureID类型（值对象）
typedef uint64_t FutureID;

// TaskID类型（值对象，用于引用TaskExecution聚合）
typedef uint64_t TaskID;

// 轮询状态枚举（值对象）
typedef enum {
    POLL_PENDING,   // 异步操作未完成
    POLL_READY,     // 异步操作已完成，结果可用
    POLL_COMPLETED  // Future已完成并被消费
} PollStatus;

// 轮询结果结构体（值对象）
typedef struct PollResult {
    PollStatus status;
    void* value;    // 结果值
} PollResult;

// ============================================================================
// 内部实体定义
// ============================================================================

/**
 * @brief WaitNode 内部实体（不对外暴露）
 * 
 * 注意：WaitNode只存储TaskID，不直接持有Task*指针，避免循环依赖。
 * 唤醒等待者时，通过TaskRepository根据TaskID查找Task。
 */
typedef struct WaitNode {
    TaskID task_id;          // 等待的Task ID（不直接持有Task*）
    struct WaitNode* next;   // 下一个节点
} WaitNode;

// ============================================================================
// 聚合根定义
// ============================================================================

/**
 * @brief Future 聚合根
 * 
 * 职责：
 * - 维护异步操作的状态一致性
 * - 管理等待队列（通过TaskID引用Task）
 * - 发布领域事件（FutureResolved, FutureRejected）
 * 
 * 不变条件：
 * - 等待队列中的TaskID必须有效
 * - Future状态转换必须符合业务规则（PENDING -> RESOLVED/REJECTED）
 * - 已解决的Future不能再次解决或拒绝
 */
typedef struct Future {
    // 聚合根标识
    FutureID id;               // Future唯一ID
    
    // 状态（值对象）
    FutureState state;         // Future状态
    FuturePriority priority;   // Future优先级
    
    // 结果数据
    void* result;              // 结果值（成功）或错误（失败）
    bool consumed;             // 是否已被消费
    bool result_needs_free;    // 结果是否需要释放
    
    // 等待队列（内部实体，只存储TaskID）
    WaitNode* wait_queue_head; // 等待队列头
    WaitNode* wait_queue_tail; // 等待队列尾
    int wait_count;            // 等待的任务数量
    
    // 同步机制
    pthread_mutex_t mutex;     // 保护Future状态
    pthread_cond_t cond;       // 条件变量，用于等待
    
    // 时间戳
    time_t created_at;         // 创建时间
    time_t resolved_at;        // 完成时间（成功）
    time_t rejected_at;        // 完成时间（失败）
    
    // 统计信息
    uint64_t poll_count;       // 轮询次数
    uint64_t wait_time_ms;     // 等待时间（毫秒）
    
    // 可选方法
    bool (*cancel)(struct Future* self);     // 取消异步操作（返回bool表示是否成功）
    void (*cleanup)(struct Future* self);    // 清理资源
    
    // 领域事件（待发布）
    struct DomainEvent** domain_events;      // 领域事件列表
    size_t domain_events_count;              // 事件数量
    size_t domain_events_capacity;           // 事件容量
    
    // 事件总线（可选，用于发布领域事件）
    struct EventBus* event_bus;              // 事件总线实例
} Future;

// ============================================================================
// 聚合根方法（业务操作）
// ============================================================================

/**
 * @brief 解决Future（设置成功结果）
 * 
 * 业务规则：
 * - 只有PENDING状态的Future可以解决
 * - 解决后发布FutureResolved事件
 * - 唤醒所有等待的Task（通过TaskID查找）
 * 
 * @param future Future聚合根
 * @param value 结果值
 * @return 成功返回true，失败返回false
 */
bool future_resolve(Future* future, void* value);

/**
 * @brief 拒绝Future（设置错误结果）
 * 
 * 业务规则：
 * - 只有PENDING状态的Future可以拒绝
 * - 拒绝后发布FutureRejected事件
 * - 唤醒所有等待的Task（通过TaskID查找）
 * 
 * @param future Future聚合根
 * @param error 错误信息
 * @return 成功返回true，失败返回false
 */
bool future_reject(Future* future, void* error);

/**
 * @brief 添加等待者（通过TaskID）
 * 
 * 业务规则：
 * - 只有PENDING状态的Future可以添加等待者
 * - 等待者通过TaskID引用，不直接持有Task*
 * 
 * @param future Future聚合根
 * @param task_id 等待的Task ID
 * @return 成功返回true，失败返回false
 */
bool future_add_waiter(Future* future, TaskID task_id);

/**
 * @brief 唤醒所有等待者（通过TaskID查找Task）
 * 
 * 业务规则：
 * - 通过TaskRepository根据TaskID查找Task
 * - 调用Task聚合根方法唤醒Task
 * - 清空等待队列
 * 
 * 注意：此方法需要TaskRepository依赖，通过依赖注入传入
 * 
 * @param future Future聚合根
 * @param task_repository TaskRepository（用于查找Task）
 * @return 成功返回true，失败返回false
 */
bool future_wake_waiters(Future* future, void* task_repository);

/**
 * @brief 轮询Future状态
 * 
 * 业务规则：
 * - 如果Future已完成，返回结果
 * - 如果Future未完成，将Task加入等待队列
 * 
 * @param future Future聚合根
 * @param task_id 轮询的Task ID（可选，如果提供则加入等待队列）
 * @return PollResult 轮询结果
 */
PollResult future_poll(Future* future, TaskID task_id);

/**
 * @brief 取消Future
 * 
 * 业务规则：
 * - 只有PENDING状态的Future可以取消
 * - 取消后状态变为REJECTED
 * - 唤醒所有等待者
 * 
 * @param future Future聚合根
 * @return 成功返回true，失败返回false
 */
bool future_cancel(Future* future);

// ============================================================================
// 查询方法（只读访问）
// ============================================================================

/**
 * @brief 获取Future ID
 */
FutureID future_get_id(const Future* future);

/**
 * @brief 获取Future状态
 */
FutureState future_get_state(const Future* future);

/**
 * @brief 检查Future是否已完成
 */
bool future_is_complete(const Future* future);

/**
 * @brief 检查Future是否已消费
 */
bool future_is_consumed(const Future* future);

/**
 * @brief 获取结果值（只读）
 */
const void* future_get_result(const Future* future);

// ============================================================================
// 不变条件验证
// ============================================================================

/**
 * @brief 验证Future不变条件
 * 
 * 验证规则：
 * - 等待队列中的TaskID必须有效
 * - Future状态必须有效
 * - 已解决的Future不能再次解决或拒绝
 * 
 * @param future Future聚合根
 * @return 验证通过返回true，失败返回false
 */
bool future_validate_invariants(const Future* future);

// ============================================================================
// 领域事件
// ============================================================================

/**
 * @brief 获取并清空领域事件列表
 * 
 * @param future Future聚合根
 * @param events 输出参数，事件数组
 * @param count 输出参数，事件数量
 * @return 成功返回true，失败返回false
 */
bool future_get_domain_events(Future* future, struct DomainEvent*** events, size_t* count);

// ============================================================================
// 工厂方法
// ============================================================================

/**
 * @brief 创建Future聚合根
 * 
 * @param priority Future优先级
 * @return Future* 新创建的Future，失败返回NULL
 */
Future* future_factory_create(FuturePriority priority, struct EventBus* event_bus);

/**
 * @brief 销毁Future聚合根
 * 
 * @param future Future聚合根
 */
void future_destroy(Future* future);

/**
 * @brief 生成Future ID
 * 
 * @return FutureID 新的Future ID
 */
FutureID future_generate_id(void);

// ============================================================================
// 工具函数（兼容现有代码）
// ============================================================================

/**
 * @brief 阻塞等待Future完成（用于同步等待）
 * 
 * 注意：此方法用于同步等待场景，异步场景应使用future_poll()
 * 
 * @param future Future聚合根
 * @return void* 结果值，失败返回NULL
 */
void* future_wait(Future* future);

#endif // ASYNC_COMPUTATION_AGGREGATE_FUTURE_H
