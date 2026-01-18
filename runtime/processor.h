#ifndef PROCESSOR_H
#define PROCESSOR_H

#include <stdint.h>
#include <pthread.h>
#include "task.h"

// 前向声明
struct Machine;

// 处理器结构体（P - Processor）
typedef struct Processor {
    uint32_t id;                    // 处理器ID
    Task* local_queue;              // 本地任务队列头
    Task* local_queue_tail;         // 本地任务队列尾
    uint32_t queue_size;            // 队列大小
    uint32_t max_queue_size;        // 最大队列大小
    pthread_mutex_t lock;           // 队列锁
    struct Machine* bound_machine;   // 绑定的机器

    // 统计信息
    uint64_t tasks_processed;       // 已处理任务数
    uint64_t steals_attempted;      // 尝试窃取次数
    uint64_t steals_succeeded;      // 窃取成功次数
} Processor;

// 处理器管理函数声明
Processor* processor_create(uint32_t id, uint32_t max_queue_size);
void processor_destroy(Processor* processor);
bool processor_push_local(Processor* processor, Task* task);
Task* processor_pop_local(Processor* processor);
Task* processor_steal_local(Processor* processor);
uint32_t processor_get_queue_size(Processor* processor);

// 获取当前处理器（线程本地）
Processor* processor_get_current(void);
void processor_set_current(Processor* processor);

// 调度函数
void processor_schedule(Processor* p);

// 统计信息函数
void processor_print_stats(Processor* processor);

#endif // PROCESSOR_H
