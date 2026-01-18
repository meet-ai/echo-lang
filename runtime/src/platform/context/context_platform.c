/**
 * @file context_platform.c
 * @brief 跨平台上下文切换统一实现
 *
 * 自动检测平台并调用相应的上下文切换实现
 */

#include "context_platform.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ============================================================================
// 平台检测实现
// ============================================================================

cpu_arch_t detect_cpu_arch(void) {
#if defined(__x86_64__) || defined(_M_X64)
    return ARCH_X86_64;
#elif defined(__aarch64__) || defined(_M_ARM64)
    return ARCH_AARCH64;
#else
    return ARCH_UNKNOWN;
#endif
}

const char* get_arch_name(cpu_arch_t arch) {
    switch (arch) {
        case ARCH_X86_64:  return "x86_64";
        case ARCH_AARCH64: return "aarch64";
        default:           return "unknown";
    }
}

// ============================================================================
// 统一的上下文接口实现
// ============================================================================

int coro_context_init(coro_context_t* ctx, void* stack_base, size_t stack_size,
                      void (*entry_point)(void*), void* arg) {
    if (!ctx || !stack_base || stack_size == 0) {
        return -1;
    }

    // 检测当前架构
    cpu_arch_t arch = detect_cpu_arch();
    ctx->arch = arch;

    // 根据架构调用相应的初始化函数
    switch (arch) {
#ifdef __x86_64__
        case ARCH_X86_64: {
            // 使用x86_64特定的上下文结构
            platform_context_init(&ctx->platform.x86_64, stack_base, stack_size, entry_point, arg);
            break;
        }
#endif
#ifdef __aarch64__
        case ARCH_AARCH64: {
            // 使用aarch64特定的上下文结构
            platform_context_init(&ctx->platform.aarch64, stack_base, stack_size, entry_point, arg);
            break;
        }
#endif
        default:
            fprintf(stderr, "ERROR: Unsupported CPU architecture: %s\n", get_arch_name(arch));
            return -1;
    }

    return 0;
}

void coro_context_switch(coro_context_t* from_ctx, coro_context_t* to_ctx) {
    if (!to_ctx) {
        fprintf(stderr, "ERROR: Target context is NULL\n");
        return;
    }

    // 检查架构一致性
    if (from_ctx && from_ctx->arch != to_ctx->arch) {
        fprintf(stderr, "ERROR: Context architecture mismatch: %s vs %s\n",
                get_arch_name(from_ctx->arch), get_arch_name(to_ctx->arch));
        return;
    }

    cpu_arch_t arch = to_ctx->arch;

    // 根据架构调用相应的切换函数
    switch (arch) {
#ifdef __x86_64__
        case ARCH_X86_64: {
            void* from_x86 = from_ctx ? &from_ctx->platform.x86_64 : NULL;
            void* to_x86 = &to_ctx->platform.x86_64;
            platform_context_switch(from_x86, to_x86);
            break;
        }
#endif
#ifdef __aarch64__
        case ARCH_AARCH64: {
            void* from_arm = from_ctx ? &from_ctx->platform.aarch64 : NULL;
            void* to_arm = &to_ctx->platform.aarch64;
            platform_context_switch(from_arm, to_arm);
            break;
        }
#endif
        default:
            fprintf(stderr, "ERROR: Unsupported CPU architecture for context switch: %s\n",
                    get_arch_name(arch));
            break;
    }
}

int coro_context_validate(const coro_context_t* ctx) {
    if (!ctx) return 0;

    cpu_arch_t arch = ctx->arch;

    // 根据架构调用相应的验证函数
    switch (arch) {
#ifdef __x86_64__
        case ARCH_X86_64: {
            return context_validate(&ctx->platform.x86_64);
        }
#endif
#ifdef __aarch64__
        case ARCH_AARCH64: {
            return context_validate(&ctx->platform.aarch64);
        }
#endif
        default:
            return 0;
    }
}

cpu_arch_t coro_context_get_arch(const coro_context_t* ctx) {
    return ctx ? ctx->arch : ARCH_UNKNOWN;
}
