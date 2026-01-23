#ifndef TASK_H
#define TASK_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明，避免循环依赖
struct Future;
struct Coroutine;

// 任务状态枚举
typedef enum {
    TASK_READY,      // 准备执行
    TASK_RUNNING,    // 正在执行
    TASK_WAITING,    // 等待异步操作完成
    TASK_COMPLETED,  // 执行完成
    TASK_CANCELLED   // 被取消
} TaskStatus;

// 任务控制块结构体
typedef struct Task {
    uint64_t id;              // 任务唯一ID
    TaskStatus status;        // 任务状态
    struct Future* future;           // 当前等待的异步计算
    struct Coroutine* coroutine;     // 关联的协程
    void* result;             // 执行结果
    void* stack;              // 执行栈（如果不使用协程）
    size_t stack_size;        // 栈大小

    // 执行函数（spawn使用）
    void (*entry_point)(void*);  // 任务入口函数
    void* arg;                  // 函数参数

    // 队列管理（用于调度器）
    struct Task* next;        // 下一个任务指针

    // 同步机制
    pthread_mutex_t mutex;    // 任务状态保护
    pthread_cond_t cond;      // 条件变量，用于等待

    // 回调函数
    void (*on_complete)(struct Task*);  // 完成回调
    void* user_data;         // 用户数据
} Task;

// 任务管理函数声明
Task* task_create(void (*entry_point)(void*), void* arg);
void task_destroy(Task* task);
void task_execute(Task* task);
bool task_is_complete(Task* task);
void task_wait(Task* task);
void task_set_future(Task* task, struct Future* future);
void task_wake(Task* task);
uint64_t task_generate_id(void);

#endif // TASK_H
