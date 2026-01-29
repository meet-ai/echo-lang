/**
 * @file scheduler.c
 * @brief Scheduler向后兼容实现（已废弃，功能已迁移到适配层）
 * 
 * ⚠️ 注意：此文件已被CMakeLists.txt排除，不会被编译
 * 
 * 所有功能已迁移到适配层：
 * - runtime/domain/task_scheduling/adapter/scheduler_adapter.c
 * 
 * 此文件保留仅用于参考，实际实现请使用适配层。
 * 
 * 如果需要在其他地方使用这些函数，应该：
 * 1. 直接使用适配层接口：scheduler_adapter.h
 * 2. 或者通过scheduler.h头文件（它已经指向适配层）
 */

#include "scheduler.h"
// 注意：这个文件提供向后兼容的实现，但应该通过适配层调用新的聚合根方法
// 新的代码应该使用适配层接口：runtime/domain/task_scheduling/adapter/scheduler_adapter.h
#include "../task_scheduling/adapter/scheduler_adapter.h"  // 使用适配层
#include "processor.h"
#include "../coroutine/coroutine.h"
#include "../task_execution/aggregate/task.h"  // 使用新的Task聚合根
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 全局调度器实例（单例）
static Scheduler* global_scheduler = NULL;
static pthread_mutex_t global_scheduler_lock = PTHREAD_MUTEX_INITIALIZER;

// 当前执行任务（使用新的Task聚合根类型）
Task* current_task = NULL;

// 获取全局调度器
Scheduler* get_global_scheduler(void) {
    pthread_mutex_lock(&global_scheduler_lock);
    Scheduler* scheduler = global_scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
    return scheduler;
}

// 设置全局调度器
void set_global_scheduler(Scheduler* scheduler) {
    pthread_mutex_lock(&global_scheduler_lock);
    global_scheduler = scheduler;
    pthread_mutex_unlock(&global_scheduler_lock);
}

// ⚠️ 已废弃：功能已迁移到适配层
// 请使用适配层的实现：runtime/domain/task_scheduling/adapter/scheduler_adapter.c
// 此函数不会被编译（CMakeLists.txt排除了scheduler目录）

// ⚠️ 以下函数已废弃，功能已迁移到适配层
// 请使用适配层的实现：runtime/domain/task_scheduling/adapter/scheduler_adapter.c
// 这些函数不会被编译（CMakeLists.txt排除了scheduler目录）

// 已废弃：请使用适配层的scheduler_destroy
// 已废弃：请使用适配层的scheduler_add_task
// 已废弃：请使用适配层的scheduler_get_work
// 已废弃：请使用适配层的scheduler_steal_work
// 已废弃：请使用适配层的scheduler_run
// 已废弃：请使用适配层的scheduler_stop

// ⚠️ 已废弃：请使用适配层的scheduler_notify_coroutine_completed
// ⚠️ 已废弃：请使用适配层的scheduler_print_stats
// 这些函数不会被编译（CMakeLists.txt排除了scheduler目录）
