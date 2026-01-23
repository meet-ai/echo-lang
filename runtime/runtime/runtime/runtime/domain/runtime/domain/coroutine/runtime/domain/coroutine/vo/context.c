/**
 * @file context.c
 * @brief Context 实体实现
 *
 * 使用 setjmp/longjmp 实现基本的上下文切换
 */

#include "context.h"
#include "coroutine.h"  // 包含coroutine_t定义
#include "../../platform/context/context_platform.h"  // 平台上下文接口
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <setjmp.h>

// 全局变量：调度器上下文（用于协程完成时切换回）
context_t* g_scheduler_context = NULL;
// 全局变量：当前正在执行的协程
coroutine_t* g_current_coroutine = NULL;

// ============================================================================
// 内部数据结构
// ============================================================================

/**
 * @brief 上下文内部结构体
 */
typedef struct ContextInternal {
    context_t base;      // 基础上下文
    jmp_buf env;         // setjmp/longjmp 环境 (后备方案)
    coro_context_t* platform_context;  // 平台特定的上下文
    void (*entry_point)(void*);  // 入口函数
    void* arg;           // 函数参数
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
    extern coroutine_t* g_current_coroutine;
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
    g_current_coroutine->state = COROUTINE_STATE_COMPLETED;
    printf("DEBUG: Set coroutine %llu state to COMPLETED\n", g_current_coroutine->id);

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

    // 分配平台上下文
    ctx->platform_context = (coro_context_t*)malloc(sizeof(coro_context_t));
    if (!ctx->platform_context) {
        fprintf(stderr, "Failed to allocate platform context\n");
        free(ctx);
        return NULL;
    }
    memset(ctx->platform_context, 0, sizeof(coro_context_t));

    // 分配栈空间（简化实现，实际可能不需要手动管理）
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

    if (ctx->platform_context) {
        free(ctx->platform_context);
        ctx->platform_context = NULL;
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

    // 初始化平台上下文
    if (ctx->platform_context) {
        // 分配栈空间（简化实现）
        void* stack_base = malloc(ctx->base.stack_size);
        if (!stack_base) {
            fprintf(stderr, "Failed to allocate stack for context %llu\n", ctx->base.id);
            return;
        }

        // 设置栈信息
        ctx->base.stack = stack_base;

        // 使用平台无关的接口初始化上下文
        coro_context_init(ctx->platform_context, stack_base, ctx->base.stack_size,
                         entry_point, arg);
    }

    // 设置初始jmp_buf环境（作为后备方案）
    // 简化实现：实际应该设置正确的栈指针等
    if (setjmp(ctx->env) == 0) {
        // 初始设置
        printf("DEBUG: Initialized context %llu with entry point\n", ctx->base.id);
    } else {
        // 从longjmp返回，执行协程
        coroutine_entry_wrapper();
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
 * @brief 平台相关的上下文切换实现
 * 这个函数由各个平台提供具体的汇编实现
 */
void platform_context_switch(void* from_platform_ctx, void* to_platform_ctx) {
    // 直接调用平台的上下文切换
    coro_context_switch((coro_context_t*)from_platform_ctx, (coro_context_t*)to_platform_ctx);
}

/**
 * @brief 切换上下文
 * 使用平台相关的汇编实现进行高效的上下文切换
 */
void context_switch(context_t* from, context_t* to) {
    if (!to) return;

    context_internal_t* to_ctx = (context_internal_t*)to;

    printf("DEBUG: Switching to context %llu (from=%p)\n", to_ctx->base.id, from);

    // 获取平台上下文指针
    coro_context_t* from_coro = from ? (coro_context_t*)(((context_internal_t*)from)->platform_context) : NULL;
    coro_context_t* to_coro = (coro_context_t*)(((context_internal_t*)to)->platform_context);

    // 使用平台无关的接口
    coro_context_switch(from_coro, to_coro);
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

