/**
 * @file makecontext.c
 * @brief x86_64 上下文初始化实现
 *
 * 实现协程上下文的创建和初始化
 */

#include "context.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <assert.h>

// ============================================================================
// 协程入口包装函数
// ============================================================================

/**
 * @brief 协程入口包装函数
 *
 * 这个函数会被设置为协程的入口点，它负责：
 * 1. 调用用户指定的入口函数
 * 2. 处理协程完成后的清理工作
 *
 * @param entry_point 用户入口函数
 * @param arg 用户参数
 */
static void coroutine_entry_wrapper(void (*entry_point)(void*), void* arg) {
    // 调用用户入口函数
    if (entry_point) {
        entry_point(arg);
    }

    // 协程执行完毕
    // 注意：在协程完成时，我们需要通知调度器
    // 这里应该调用一个全局的协程完成回调函数
    // 由于我们使用的是汇编上下文切换，这里通过特殊的返回机制处理

    // 永远不应该到达这里，因为协程应该通过context_switch返回
    fprintf(stderr, "ERROR: Coroutine reached end of entry_wrapper unexpectedly\n");
    abort();
}

// ============================================================================
// 上下文初始化实现
// ============================================================================

void platform_context_init(platform_context_t* ctx, void* stack_base, size_t stack_size,
                           void (*entry_point)(void*), void* arg) {
    if (!ctx || !stack_base || stack_size == 0) {
        fprintf(stderr, "ERROR: Invalid parameters for context_init\n");
        return;
    }

    // 清空上下文
    memset(ctx, 0, sizeof(platform_context_t));

    // 设置栈信息
    ctx->stack_base = (uint64_t)stack_base;
    ctx->stack_size = stack_size;

    // 计算栈顶地址 (栈向下增长)
    uint64_t stack_top = (uint64_t)stack_base + stack_size;

    // 确保栈对齐
    stack_top = stack_top & ~(STACK_ALIGNMENT - 1);

    // 在栈上设置协程的初始栈帧
    // 我们需要模拟函数调用栈帧的结构

    // 栈布局 (从高地址到低地址):
    // | ...                    |
    // | 参数区域               |
    // | 返回地址 (coroutine_entry_wrapper) |
    // | 保存的rbp              |
    // | ...                    |

    uint64_t* stack_ptr = (uint64_t*)stack_top;

    // 预留空间用于函数参数 (entry_point 和 arg)
    stack_ptr -= 2;
    stack_ptr[0] = (uint64_t)arg;           // 第二个参数: arg
    stack_ptr[1] = (uint64_t)entry_point;   // 第一个参数: entry_point

    // 设置返回地址为coroutine_entry_wrapper
    stack_ptr -= 1;
    *stack_ptr = (uint64_t)coroutine_entry_wrapper;

    // 预留空间用于保存的rbp (模拟函数调用)
    stack_ptr -= 1;
    *stack_ptr = 0;  // 初始rbp设为0

    // 设置栈指针
    ctx->rsp = (uint64_t)stack_ptr;

    // 设置指令指针 (指向coroutine_entry_wrapper)
    ctx->rip = (uint64_t)coroutine_entry_wrapper;

    // 设置状态
    ctx->state = CONTEXT_STATE_INITIALIZED;

    printf("DEBUG: Initialized context with stack_base=%p, stack_size=%zu, rsp=%p, rip=%p\n",
           stack_base, stack_size, (void*)ctx->rsp, (void*)ctx->rip);
}

int context_validate(const platform_context_t* ctx) {
    if (!ctx) return 0;

    // 检查基本字段
    if (ctx->stack_base == 0 || ctx->stack_size == 0) return 0;
    if (ctx->rsp == 0 || ctx->rip == 0) return 0;

    // 检查栈指针在有效范围内
    uint64_t stack_top = ctx->stack_base + ctx->stack_size;
    if (ctx->rsp < ctx->stack_base || ctx->rsp >= stack_top) return 0;

    return 1;
}
