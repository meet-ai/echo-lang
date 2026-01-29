/**
 * @file scheduler_adapter.h
 * @brief TaskScheduling适配层：用于向后兼容旧的Scheduler接口
 * 
 * 这个适配层将旧的Scheduler接口转换为新的Scheduler聚合根方法，
 * 提供向后兼容的接口。
 * 
 * 使用场景：
 * - 在迁移期间，旧的代码可以继续使用旧的接口
 * - 新的代码应该直接使用新的聚合根方法
 * 
 * TODO: 阶段5后续重构：完全迁移到新的聚合根方法后，可以移除这个适配层
 */

#ifndef TASK_SCHEDULING_ADAPTER_SCHEDULER_ADAPTER_H
#define TASK_SCHEDULING_ADAPTER_SCHEDULER_ADAPTER_H

#include "../aggregate/scheduler.h"  // 新的Scheduler聚合根
#include "../../task_execution/repository/task_repository.h"
#include "../../task_execution/aggregate/task.h"
#include "../../scheduler/concurrency/processor.h"
// 事件总线接口
#include "../../shared/events/bus.h"

// ==================== 全局TaskRepository管理（暂时方案）====================
// TODO: 阶段5后续重构：应该通过依赖注入传递TaskRepository，而不是使用全局变量

/**
 * @brief 设置全局TaskRepository（用于适配层）
 */
void scheduler_adapter_set_task_repository(TaskRepository* task_repo);

/**
 * @brief 获取全局TaskRepository
 */
TaskRepository* scheduler_adapter_get_task_repository(void);

// ==================== 全局EventBus管理（暂时方案）====================
// TODO: 阶段7后续重构：应该通过依赖注入传递EventBus，而不是使用全局变量

/**
 * @brief 设置全局EventBus（用于适配层）
 */
void scheduler_adapter_set_event_bus(EventBus* event_bus);

/**
 * @brief 获取全局EventBus
 */
EventBus* scheduler_adapter_get_event_bus(void);

// ==================== 向后兼容接口 ====================
// 这些函数提供旧的接口，内部调用新的聚合根方法

/**
 * @brief 创建调度器（向后兼容接口）
 * 
 * 内部调用scheduler_factory_create，使用全局TaskRepository
 */
Scheduler* scheduler_create(uint32_t num_processors);

/**
 * @brief 销毁调度器（向后兼容接口）
 */
void scheduler_destroy(Scheduler* scheduler);

/**
 * @brief 添加任务到调度器（向后兼容接口）
 * 
 * 接受Task*，内部提取TaskID，然后调用新的聚合根方法
 */
bool scheduler_add_task(Scheduler* scheduler, Task* task);

/**
 * @brief 从调度器获取任务（向后兼容接口）
 * 
 * 返回Task*，内部通过TaskID查找Task，然后返回
 */
Task* scheduler_get_work(Scheduler* scheduler, Processor* processor);

/**
 * @brief 运行调度器（向后兼容接口）
 */
void scheduler_run(Scheduler* scheduler);

/**
 * @brief 停止调度器（向后兼容接口）
 */
void scheduler_stop(Scheduler* scheduler);

/**
 * @brief 工作窃取（向后兼容接口）
 */
bool scheduler_steal_work(Scheduler* scheduler, Processor* thief);

/**
 * @brief 打印统计信息（向后兼容接口）
 */
void scheduler_print_stats(Scheduler* scheduler);

/**
 * @brief 协程完成通知（向后兼容接口）
 */
void scheduler_notify_coroutine_completed(Scheduler* scheduler, struct Coroutine* coroutine);

/**
 * @brief 获取全局调度器（向后兼容接口）
 */
Scheduler* get_global_scheduler(void);

// ==================== 辅助查询函数 ====================
// 这些函数提供对Scheduler内部字段的访问，用于向后兼容

/**
 * @brief 检查全局队列是否为空
 */
bool scheduler_has_work_in_global_queue(Scheduler* scheduler);

/**
 * @brief 获取处理器数量
 */
uint32_t scheduler_get_num_processors(Scheduler* scheduler);

/**
 * @brief 获取机器数量
 */
uint32_t scheduler_get_num_machines(Scheduler* scheduler);

/**
 * @brief 获取指定索引的处理器（用于向后兼容）
 * 
 * 注意：Processor是内部实体，应该通过聚合根方法访问
 * 这个函数仅用于向后兼容，新的代码应该使用聚合根方法
 * 
 * @param scheduler 调度器实例
 * @param index 处理器索引（从0开始）
 * @return Processor* 处理器实例，失败返回NULL
 */
Processor* scheduler_get_processor_by_index(Scheduler* scheduler, uint32_t index);

/**
 * @brief 将任务添加到全局队列（用于向后兼容）
 * 
 * 注意：新的聚合根使用TaskID列表，这个函数接受Task*并提取TaskID
 * 
 * @param scheduler 调度器实例
 * @param task 任务实例
 * @return int 成功返回0，失败返回-1
 */
int scheduler_add_task_to_global_queue(Scheduler* scheduler, Task* task);

#endif // TASK_SCHEDULING_ADAPTER_SCHEDULER_ADAPTER_H
