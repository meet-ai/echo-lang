#ifndef SCHEDULER_H
#define SCHEDULER_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "task.h"
#include "processor.h"
#include "machine.h"

// 调度器结构体
typedef struct Scheduler {
    uint32_t id;                    // 调度器ID
    struct Processor** processors;  // 处理器数组
    uint32_t num_processors;        // 处理器数量
    struct Machine** machines;      // 机器(OS线程)数组
    uint32_t num_machines;          // 机器数量
    Task* global_queue;             // 全局任务队列
    pthread_mutex_t global_lock;    // 全局队列锁
    bool is_running;                // 调度器是否运行中

    // 统计信息
    uint64_t tasks_scheduled;       // 已调度任务数
    uint64_t tasks_completed;       // 已完成任务数
} Scheduler;

// 调度器管理函数声明
Scheduler* scheduler_create(uint32_t num_processors);
void scheduler_destroy(Scheduler* scheduler);
bool scheduler_add_task(Scheduler* scheduler, Task* task);
Task* scheduler_get_work(Scheduler* scheduler, struct Processor* processor);
void scheduler_run(Scheduler* scheduler);
void scheduler_stop(Scheduler* scheduler);

// 工作窃取相关函数
bool scheduler_steal_work(Scheduler* scheduler, struct Processor* thief);

// 统计信息函数
void scheduler_print_stats(Scheduler* scheduler);

// 全局调度器访问函数
Scheduler* get_global_scheduler(void);

// 当前执行任务的全局变量
extern struct Task* current_task;

#endif // SCHEDULER_H
