#ifndef SCHEDULER_H
#define SCHEDULER_H

/**
 * @file scheduler.h
 * @brief Scheduler接口（向后兼容）
 * 
 * 注意：这个文件提供向后兼容的接口，实际实现通过适配层调用新的聚合根方法。
 * 新的代码应该直接使用新的聚合根：runtime/domain/task_scheduling/aggregate/scheduler.h
 */

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "../task_execution/aggregate/task.h"  // 使用新的Task聚合根
#include "concurrency/processor.h"
#include "concurrency/machine.h"

// ==================== 类型定义（向后兼容）====================
// 包含新的聚合根定义，这样旧的代码可以访问结构体字段
#include "../task_scheduling/aggregate/scheduler.h"

// 注意：新的聚合根已经定义了Scheduler类型
// 旧的代码可以直接使用这个类型，但应该通过适配层的函数来操作

// ==================== 向后兼容接口 ====================
// 这些函数通过适配层实现，内部调用新的聚合根方法

Scheduler* scheduler_create(uint32_t num_processors);
void scheduler_destroy(Scheduler* scheduler);
bool scheduler_add_task(Scheduler* scheduler, Task* task);
Task* scheduler_get_work(Scheduler* scheduler, struct Processor* processor);
void scheduler_run(Scheduler* scheduler);
void scheduler_stop(Scheduler* scheduler);
bool scheduler_steal_work(Scheduler* scheduler, struct Processor* thief);
void scheduler_print_stats(Scheduler* scheduler);
void scheduler_notify_coroutine_completed(Scheduler* scheduler, struct Coroutine* coroutine);
Scheduler* get_global_scheduler(void);

// 当前执行任务的全局变量（使用新的Task聚合根类型）
extern Task* current_task;

#endif // SCHEDULER_H
