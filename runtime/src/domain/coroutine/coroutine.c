/**
 * @file coroutine.c
 * @brief Coroutine 聚合根实现
 *
 * 协程负责管理自己的生命周期、上下文切换和状态转换。
 */

#include "coroutine.h"
#include "context.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <setjmp.h>

// 全局变量：协程返回环境的jmp_buf
jmp_buf g_coroutine_return_env;

// 全局协程ID计数器
static coroutine_id_t next_coroutine_id = 1;

// ============================================================================
// Coroutine 聚合根实现
// ============================================================================

/**
 * @brief 创建协程
 */
coroutine_t* coroutine_create(void (*entry_point)(void*), void* arg, size_t stack_size, context_t* scheduler_context) {
    coroutine_t* coroutine = (coroutine_t*)malloc(sizeof(coroutine_t));
    if (!coroutine) {
        fprintf(stderr, "Failed to allocate coroutine\n");
        return NULL;
    }

    memset(coroutine, 0, sizeof(coroutine_t));

    // 初始化协程属性
    coroutine->id = next_coroutine_id++;
    coroutine->state = COROUTINE_STATE_READY;
    coroutine->stack_size = stack_size;
    coroutine->entry_point = entry_point;
    coroutine->arg = arg;
    coroutine->scheduler_context = scheduler_context;  // 保存调度器上下文

    // 创建上下文
    coroutine->context = context_create(stack_size);
    if (!coroutine->context) {
        fprintf(stderr, "Failed to create context for coroutine\n");
        free(coroutine);
        return NULL;
    }

    // 初始化上下文
    context_init(coroutine->context, entry_point, arg);

    printf("DEBUG: Created coroutine %llu with stack size %zu\n", coroutine->id, stack_size);
    return coroutine;
}

/**
 * @brief 销毁协程
 */
void coroutine_destroy(coroutine_t* coroutine) {
    if (!coroutine) return;

    // 销毁上下文
    if (coroutine->context) {
        context_destroy(coroutine->context);
        coroutine->context = NULL;
    }

    printf("DEBUG: Destroyed coroutine %llu\n", coroutine->id);
    free(coroutine);
}

/**
 * @brief 恢复协程执行
 */
void coroutine_resume(coroutine_t* coroutine) {
    if (!coroutine || coroutine->state != COROUTINE_STATE_READY) {
        return;
    }

    printf("DEBUG: Resuming coroutine %llu\n", coroutine->id);

    // 设置当前协程（用于yield操作和完成通知）
    // 函数在yield.c中定义，这里通过extern声明使用
    extern void set_current_coroutine(coroutine_t* coroutine);
    set_current_coroutine(coroutine);

    // 设置全局当前协程变量
    extern coroutine_t* g_current_coroutine;
    g_current_coroutine = coroutine;

    coroutine->state = COROUTINE_STATE_RUNNING;

    // 使用setjmp来捕获返回点
    extern jmp_buf g_coroutine_return_env;
    if (setjmp(g_coroutine_return_env) == 0) {
        // 执行上下文切换：从当前上下文切换到协程上下文
        // 注意：这里需要调度器的上下文作为参数，但暂时使用NULL表示主线程上下文
        // 在实际GMP实现中，这里应该传入调度器上下文
        context_switch(NULL, coroutine->context);
    } else {
        // 从协程返回（协程执行完毕或yield）
        printf("DEBUG: Coroutine %llu returned to caller\n", coroutine->id);
    }

    // 注意：执行到这里时，协程已经从context_switch返回，
    // 说明协程已经执行完毕或让出了控制权

    printf("DEBUG: Coroutine %llu resumed back to caller\n", coroutine->id);
}

/**
 * @brief 挂起协程
 */
void coroutine_suspend(coroutine_t* coroutine) {
    if (!coroutine || coroutine->state != COROUTINE_STATE_RUNNING) {
        return;
    }

    printf("DEBUG: Suspending coroutine %llu\n", coroutine->id);
    coroutine->state = COROUTINE_STATE_SUSPENDED;
}

/**
 * @brief 让出协程执行权
 */
void coroutine_yield(coroutine_t* coroutine) {
    if (!coroutine || coroutine->state != COROUTINE_STATE_RUNNING) {
        return;
    }

    printf("DEBUG: Coroutine %llu yielding\n", coroutine->id);

    // 暂时将状态改为挂起，等待被resume
    coroutine->state = COROUTINE_STATE_SUSPENDED;

    // 实际实现中，这里应该保存当前上下文并切换到调度器
    // context_switch(coroutine->context, scheduler_context);
}

/**
 * @brief 检查协程是否完成
 */
bool coroutine_is_completed(const coroutine_t* coroutine) {
    return coroutine && coroutine->state == COROUTINE_STATE_COMPLETED;
}

/**
 * @brief 获取协程ID
 */
coroutine_id_t coroutine_get_id(const coroutine_t* coroutine) {
    return coroutine ? coroutine->id : 0;
}

/**
 * @brief 获取协程状态
 */
coroutine_state_t coroutine_get_state(const coroutine_t* coroutine) {
    return coroutine ? coroutine->state : COROUTINE_STATE_READY;
}

/**
 * @brief 获取协程上下文
 */
struct Context* coroutine_get_context(const coroutine_t* coroutine) {
    return coroutine ? coroutine->context : NULL;
}

