#ifndef COROUTINE_H
#define COROUTINE_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#include "context.h"

// 协程状态枚举
typedef enum {
    COROUTINE_READY,      // 准备执行
    COROUTINE_RUNNING,    // 正在执行
    COROUTINE_SUSPENDED,  // 已挂起
    COROUTINE_COMPLETED,  // 已完成
    COROUTINE_CANCELLED   // 已取消
} CoroutineState;

// 协程结构体
typedef struct Coroutine {
    uint64_t id;              // 协程唯一ID
    CoroutineState state;     // 协程状态

    // 上下文切换相关
    coro_context_t context;   // 协程上下文
    char* stack;              // 栈内存
    size_t stack_size;        // 栈大小

    // 执行相关
    void (*entry_point)(void*);  // 协程入口函数
    void* arg;                  // 入口函数参数

    // 调度相关
    struct Processor* bound_processor;  // 绑定到的处理器
    struct Task* task;          // 关联的任务
    struct Coroutine* next;     // 用于队列的下一个协程

    // 同步机制
    pthread_mutex_t mutex;      // 协程状态保护
} Coroutine;

// 协程管理函数声明
Coroutine* coroutine_create(void (*entry_point)(void*), void* arg, size_t stack_size);
void coroutine_destroy(Coroutine* coroutine);
void coroutine_resume(Coroutine* coroutine);
void coroutine_suspend(Coroutine* coroutine);
bool coroutine_is_complete(Coroutine* coroutine);
uint64_t coroutine_generate_id(void);

// 协程包装函数（处理协程完成和清理）
void coroutine_wrapper(Coroutine* coroutine);

#endif // COROUTINE_H
