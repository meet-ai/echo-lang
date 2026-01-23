/**
 * @file yield.c
 * @brief Yield 操作实现
 *
 * Yield 允许协程主动让出执行权给调度器。
 */

#include "coroutine.h"
#include "context.h"
#include <stdio.h>
#include <pthread.h>

// 线程本地存储：当前正在执行的协程
static pthread_key_t current_coroutine_key;
static pthread_once_t key_once = PTHREAD_ONCE_INIT;

// 全局调度器上下文
static context_t* scheduler_context = NULL;

// 初始化线程本地存储key
static void init_current_coroutine_key(void) {
    pthread_key_create(&current_coroutine_key, NULL);
}

/**
 * @brief 获取当前线程正在执行的协程
 */
static Coroutine* get_current_coroutine(void) {
    pthread_once(&key_once, init_current_coroutine_key);
    return (Coroutine*)pthread_getspecific(current_coroutine_key);
}

/**
 * @brief 设置当前线程正在执行的协程
 */
void set_current_coroutine(Coroutine* coroutine) {
    pthread_once(&key_once, init_current_coroutine_key);
    pthread_setspecific(current_coroutine_key, coroutine);
}

/**
 * @brief 设置调度器上下文
 * 这个函数应该由Executor调用来设置调度器的上下文
 */
void yield_set_scheduler_context(context_t* context) {
    scheduler_context = context;
}

/**
 * @brief 执行yield操作
 */
void coroutine_yield_operation(Coroutine* coroutine) {
    if (!coroutine) return;

    printf("DEBUG: Executing yield for coroutine %llu\n", coroutine->id);

    if (!scheduler_context) {
        fprintf(stderr, "ERROR: No scheduler context set for yield\n");
        return;
    }

    // 保存当前协程状态
    coroutine_suspend(coroutine);

    // 执行上下文切换：从协程切换到调度器
    printf("DEBUG: Yielding from coroutine %llu to scheduler\n", coroutine->id);
    context_switch(coroutine->context, scheduler_context);
}

/**
 * @brief 从调度器恢复协程执行
 */
void coroutine_resume_from_yield(Coroutine* coroutine) {
    if (!coroutine || !scheduler_context) {
        fprintf(stderr, "ERROR: Cannot resume coroutine from yield\n");
        return;
    }

    printf("DEBUG: Resuming coroutine %llu from yield\n", coroutine->id);

    // 设置当前协程
    set_current_coroutine(coroutine);

    // 恢复协程状态
    coroutine->state = COROUTINE_RUNNING;

    // 执行上下文切换：从调度器切换到协程
    printf("DEBUG: Switching back to coroutine %llu\n", coroutine->id);
    context_switch(scheduler_context, coroutine->context);
}
