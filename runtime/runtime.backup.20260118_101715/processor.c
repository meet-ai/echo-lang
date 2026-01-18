#include "processor.h"
#include "scheduler.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>

// 声明全局变量
extern struct Task* current_task;

// 创建处理器
Processor* processor_create(uint32_t id, uint32_t max_queue_size) {
    Processor* processor = (Processor*)malloc(sizeof(Processor));
    if (!processor) {
        fprintf(stderr, "ERROR: Failed to allocate memory for processor\n");
        return NULL;
    }

    memset(processor, 0, sizeof(Processor));

    // 初始化处理器属性
    processor->id = id;
    processor->local_queue = NULL;
    processor->local_queue_tail = NULL;
    processor->queue_size = 0;
    processor->max_queue_size = max_queue_size;

    // 初始化锁
    pthread_mutex_init(&processor->lock, NULL);

    printf("DEBUG: Created processor %u with max queue size %u\n", id, max_queue_size);
    return processor;
}

// 销毁处理器
void processor_destroy(Processor* processor) {
    if (!processor) return;

    // 注意：这里不清理任务队列，应该由调用者负责
    // 任务的生命周期由任务创建者管理

    // 清理锁
    pthread_mutex_destroy(&processor->lock);

    printf("DEBUG: Destroyed processor %u\n", processor->id);
    free(processor);
}

// 向本地队列推送任务（LIFO）
bool processor_push_local(Processor* processor, Task* task) {
    if (!processor || !task) return false;

    pthread_mutex_lock(&processor->lock);

    if (processor->queue_size >= processor->max_queue_size) {
        pthread_mutex_unlock(&processor->lock);
        return false; // 队列已满
    }

    // 将任务添加到队列头（LIFO）
    task->next = processor->local_queue;
    processor->local_queue = task;

    if (!processor->local_queue_tail) {
        processor->local_queue_tail = task;
    }

    processor->queue_size++;
    processor->tasks_processed++;

    printf("DEBUG: Processor %u pushed task %llu, queue size: %u\n",
           processor->id, task->id, processor->queue_size);

    pthread_mutex_unlock(&processor->lock);
    return true;
}

// 从本地队列弹出任务（LIFO）
Task* processor_pop_local(Processor* processor) {
    if (!processor) return NULL;

    pthread_mutex_lock(&processor->lock);

    if (processor->queue_size == 0) {
        pthread_mutex_unlock(&processor->lock);
        return NULL;
    }

    // 从队列头弹出（LIFO）
    Task* task = processor->local_queue;
    processor->local_queue = task->next;
    task->next = NULL; // 清除next指针

    if (processor->queue_size == 1) {
        processor->local_queue_tail = NULL;
    }

    processor->queue_size--;

    pthread_mutex_unlock(&processor->lock);
    return task;
}

// 从本地队列窃取任务（FIFO，用于工作窃取）
Task* processor_steal_local(Processor* processor) {
    if (!processor) return NULL;

    pthread_mutex_lock(&processor->lock);

    if (processor->queue_size == 0) {
        pthread_mutex_unlock(&processor->lock);
        return NULL;
    }

    // 从队列尾窃取（FIFO，避免与本地pop冲突）
    Task* task = processor->local_queue_tail;

    // 找到倒数第二个任务
    if (processor->queue_size == 1) {
        processor->local_queue = NULL;
        processor->local_queue_tail = NULL;
    } else {
        Task* prev = processor->local_queue;
        while (prev->next != task) {
            prev = prev->next;
        }
        prev->next = NULL;
        processor->local_queue_tail = prev;
    }

    task->next = NULL; // 清除next指针
    processor->queue_size--;

    printf("DEBUG: Processor %u stole task %llu, queue size: %u\n",
           processor->id, task->id, processor->queue_size);

    pthread_mutex_unlock(&processor->lock);
    return task;
}

// 获取队列大小
uint32_t processor_get_queue_size(Processor* processor) {
    if (!processor) return 0;

    pthread_mutex_lock(&processor->lock);
    uint32_t size = processor->queue_size;
    pthread_mutex_unlock(&processor->lock);

    return size;
}

// 处理器调度循环（暂时简化实现）
void processor_schedule(Processor* p) {
    // 暂时空实现，单线程调度器不需要这个
    printf("DEBUG: processor_schedule called but not implemented for single-threaded scheduler\n");
}

// 获取当前处理器（线程本地存储）
static __thread Processor* current_processor = NULL;

Processor* processor_get_current(void) {
    return current_processor;
}

void processor_set_current(Processor* processor) {
    current_processor = processor;
}

// 打印处理器统计信息
void processor_print_stats(Processor* processor) {
    if (!processor) return;

    pthread_mutex_lock(&processor->lock);
    printf("Processor %u stats:\n", processor->id);
    printf("  Tasks processed: %llu\n", processor->tasks_processed);
    printf("  Current queue size: %u\n", processor->queue_size);
    printf("  Steals attempted: %llu\n", processor->steals_attempted);
    printf("  Steals succeeded: %llu\n", processor->steals_succeeded);
    printf("  Steal success rate: %.2f%%\n",
           processor->steals_attempted > 0 ?
           (double)processor->steals_succeeded / processor->steals_attempted * 100 : 0);
    pthread_mutex_unlock(&processor->lock);
}
