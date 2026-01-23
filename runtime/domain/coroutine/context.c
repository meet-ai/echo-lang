/**
 * @file context.c
 * @brief Context 实体实现
 *
 * 使用 setjmp/longjmp 实现基本的上下文切换
 */

#include "context.h"
#include "coroutine.h"  // 包含Coroutine定义
#include "../scheduler/scheduler.h"  // 包含Scheduler定义
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <setjmp.h>

// 全局变量：调度器上下文（用于协程完成时切换回）
context_t* g_scheduler_context = NULL;
// 全局变量：当前正在执行的协程
Coroutine* g_current_coroutine = NULL;

// ============================================================================
// 内部数据结构
// ============================================================================

/**
 * @brief 上下文内部结构体
 * 使用 setjmp/longjmp 实现上下文切换（简化版，用于测试）
 */
typedef struct ContextInternal {
    context_t base;      // 基础上下文
    jmp_buf env;         // setjmp/longjmp 环境
    void (*entry_point)(void*);  // 入口函数
    void* arg;           // 函数参数
    bool initialized;    // 是否已初始化
} context_internal_t;

// 全局变量用于上下文切换
static context_internal_t* current_context = NULL;
static context_internal_t* target_context = NULL;

// ============================================================================
// 协程入口包装函数
// ============================================================================

/**
 * @brief 协程入口包装函数
 * 这个函数会被setjmp/longjmp调用
 */
static void coroutine_entry_wrapper(void) {
    // 使用全局协程变量，不依赖current_context
    extern Coroutine* g_current_coroutine;
    if (!g_current_coroutine) {
        fprintf(stderr, "ERROR: No current coroutine in entry wrapper\n");
        return;
    }

    printf("DEBUG: Executing coroutine %llu entry function\n", g_current_coroutine->id);

    // 调用用户入口函数
    if (g_current_coroutine->entry_point) {
        g_current_coroutine->entry_point(g_current_coroutine->arg);
    }

    // 协程执行完毕，设置状态并切换回调度器上下文
    printf("DEBUG: Coroutine %llu completed, switching back to scheduler\n", g_current_coroutine->id);

    // 设置协程状态为完成
    g_current_coroutine->state = COROUTINE_COMPLETED;
    printf("DEBUG: Set coroutine %llu state to COMPLETED\n", g_current_coroutine->id);

    // 通知调度器协程已完成
    extern Scheduler* get_global_scheduler(void);
    extern void scheduler_notify_coroutine_completed(Scheduler* scheduler, Coroutine* coroutine);
    Scheduler* scheduler = get_global_scheduler();
    if (scheduler) {
        scheduler_notify_coroutine_completed(scheduler, g_current_coroutine);
    }

    // 使用全局调度器上下文切换回
    extern context_t* g_scheduler_context;
    if (g_scheduler_context) {
        printf("DEBUG: Switching back to scheduler context\n");
        context_switch(g_current_coroutine->context, g_scheduler_context);
    } else {
        // 测试环境：没有调度器上下文，使用longjmp返回到coroutine_resume
        printf("DEBUG: No scheduler context, returning via longjmp\n");
        extern jmp_buf g_coroutine_return_env;
        longjmp(g_coroutine_return_env, 1);
    }
}

// ============================================================================
// Context 实体实现
// ============================================================================

/**
 * @brief 创建上下文
 */
context_t* context_create(size_t stack_size) {
    context_internal_t* ctx = (context_internal_t*)malloc(sizeof(context_internal_t));
    if (!ctx) {
        fprintf(stderr, "Failed to allocate context\n");
        return NULL;
    }

    memset(ctx, 0, sizeof(context_internal_t));

    // 初始化基础信息
    static uint64_t next_id = 1;
    ctx->base.id = next_id++;
    ctx->base.stack_size = stack_size;
    ctx->base.is_main_context = false;
    ctx->initialized = false;
    ctx->entry_point = NULL;
    ctx->arg = NULL;

    // 分配栈空间（简化实现，使用 setjmp/longjmp）
    if (stack_size > 0) {
        ctx->base.stack = (char*)malloc(stack_size);
        if (!ctx->base.stack) {
            fprintf(stderr, "Failed to allocate stack\n");
            free(ctx);
            return NULL;
        }
        memset(ctx->base.stack, 0, stack_size);
    }

    printf("DEBUG: Created context %llu with stack size %zu\n", ctx->base.id, stack_size);
    return (context_t*)ctx;
}

/**
 * @brief 销毁上下文
 */
void context_destroy(context_t* context) {
    if (!context) return;

    context_internal_t* ctx = (context_internal_t*)context;

    if (ctx->base.stack) {
        free(ctx->base.stack);
        ctx->base.stack = NULL;
    }

    printf("DEBUG: Destroyed context %llu\n", ctx->base.id);
    free(ctx);
}

/**
 * @brief 初始化上下文
 */
void context_init(context_t* context, void (*entry_point)(void*), void* arg) {
    if (!context) return;

    context_internal_t* ctx = (context_internal_t*)context;
    ctx->entry_point = entry_point;
    ctx->arg = arg;

    // 使用 setjmp/longjmp 实现上下文切换（简化版）
    // 注意：这是一个简化的实现，用于测试目的
    // 实际生产环境应该使用平台特定的汇编实现
    
    // 如果栈还没有分配，分配栈空间
    if (!ctx->base.stack && ctx->base.stack_size > 0) {
        ctx->base.stack = (char*)malloc(ctx->base.stack_size);
        if (!ctx->base.stack) {
            fprintf(stderr, "Failed to allocate stack for context %llu\n", ctx->base.id);
            return;
        }
        memset(ctx->base.stack, 0, ctx->base.stack_size);
    }

    // 设置初始jmp_buf环境
    if (setjmp(ctx->env) == 0) {
        // 初始设置
        ctx->initialized = true;
        printf("DEBUG: Initialized context %llu with entry point\n", ctx->base.id);
    } else {
        // 从longjmp返回，执行协程入口函数
        if (ctx->entry_point) {
            ctx->entry_point(ctx->arg);
        }
        // 协程执行完毕
        printf("DEBUG: Context %llu entry function completed\n", ctx->base.id);
    }
}

/**
 * @brief 保存当前上下文
 */
void context_save(context_t* context) {
    if (!context) return;

    context_internal_t* ctx = (context_internal_t*)context;

    // 使用setjmp保存当前上下文
    if (setjmp(ctx->env) == 0) {
        // 刚刚保存了上下文
        printf("DEBUG: Saved context %llu\n", ctx->base.id);
    } else {
        // 从longjmp返回，这里应该继续执行
        printf("DEBUG: Restored context %llu\n", ctx->base.id);
    }
}

/**
 * @brief 恢复上下文
 */
void context_restore(const context_t* context) {
    if (!context) return;

    const context_internal_t* ctx = (const context_internal_t*)context;

    // 使用longjmp恢复上下文
    longjmp(ctx->env, 1);
}

/**
 * @brief 切换上下文
 * 使用 setjmp/longjmp 实现上下文切换（简化版，用于测试）
 */
void context_switch(context_t* from, context_t* to) {
    if (!to) return;

    context_internal_t* to_ctx = (context_internal_t*)to;
    context_internal_t* from_ctx = from ? (context_internal_t*)from : NULL;

    printf("DEBUG: Switching to context %llu (from=%p)\n", to_ctx->base.id, from);

    // 如果目标上下文未初始化，先初始化它
    if (!to_ctx->initialized && to_ctx->entry_point) {
        // 保存当前上下文
        if (from_ctx) {
            if (setjmp(from_ctx->env) == 0) {
                // 跳转到目标上下文
                longjmp(to_ctx->env, 1);
            }
        } else {
            // 从调度器上下文跳转到协程上下文
            longjmp(to_ctx->env, 1);
        }
    } else {
        // 上下文已初始化，直接切换
        if (from_ctx) {
            if (setjmp(from_ctx->env) == 0) {
                longjmp(to_ctx->env, 1);
            }
        } else {
            longjmp(to_ctx->env, 1);
        }
    }
}

/**
 * @brief 获取上下文ID
 */
uint64_t context_get_id(const context_t* context) {
    return context ? context->id : 0;
}

/**
 * @brief 检查上下文是否有效
 */
bool context_is_valid(const context_t* context) {
    return context != NULL && context->id > 0;
}

