#ifndef MACHINE_H
#define MACHINE_H

#include <stdint.h>
#include <pthread.h>
#include "processor.h"
#include "../../coroutine/context.h"

// Machine (M) - OS线程结构体
typedef struct Machine {
    uint32_t id;                    // Machine ID
    pthread_t thread;              // OS线程
    struct Processor* processor;    // 绑定的处理器
    context_t* context;             // 线程上下文（用于上下文切换）
    bool is_running;                // 是否正在运行
    bool should_stop;               // 是否应该停止
    struct Scheduler* scheduler;    // 所属的调度器
} Machine;

// Machine管理函数声明
Machine* machine_create(uint32_t id, struct Scheduler* scheduler);
void machine_destroy(Machine* machine);
bool machine_start(Machine* machine);
void machine_stop(Machine* machine);
void* machine_thread_entry(void* arg);  // 线程入口函数

#endif // MACHINE_H
