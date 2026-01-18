/**
 * @file context_platform.h
 * @brief 跨平台上下文切换统一接口
 *
 * 提供统一的上下文切换接口，自动选择合适的平台实现
 */

#ifndef CONTEXT_PLATFORM_H
#define CONTEXT_PLATFORM_H

#include <stdint.h>
#include <stddef.h>

// 前向声明context_t（定义在domain层）
typedef struct Context context_t;

// ============================================================================
// 平台检测
// ============================================================================

/**
 * @brief 支持的CPU架构
 */
typedef enum {
    ARCH_UNKNOWN = 0,
    ARCH_X86_64  = 1,
    ARCH_AARCH64 = 2,
} cpu_arch_t;

/**
 * @brief 检测当前CPU架构
 *
 * @return CPU架构枚举值
 */
cpu_arch_t detect_cpu_arch(void);

/**
 * @brief 获取架构名称字符串
 *
 * @param arch CPU架构
 * @return 架构名称字符串
 */
const char* get_arch_name(cpu_arch_t arch);

// ============================================================================
// 统一的上下文结构和接口
// ============================================================================

/**
 * @brief 统一的协程上下文结构
 *
 * 这个结构的大小必须足够容纳所有平台特定的上下文结构
 */
typedef struct coro_context {
    // 平台特定的上下文数据
    union {
#ifdef __x86_64__
        struct {
            uint64_t rbx, rbp, r12, r13, r14, r15;
            uint64_t rsp, rip;
            uint64_t stack_base, stack_size;
            int state;
        } x86_64;
#endif
#ifdef __aarch64__
        struct {
            uint64_t x19, x20, x21, x22, x23, x24, x25, x26, x27, x28;
            uint64_t sp, lr;
            uint64_t stack_base, stack_size;
            int state;
        } aarch64;
#endif
        // 通用填充，确保结构体大小一致
        uint64_t data[16];
    } platform;

    // 通用字段
    cpu_arch_t arch;        // 使用的架构
    void* user_data;        // 用户数据指针
} coro_context_t;

/**
 * @brief 初始化协程上下文
 *
 * @param ctx 上下文指针
 * @param stack_base 栈底地址
 * @param stack_size 栈大小
 * @param entry_point 入口函数
 * @param arg 函数参数
 * @return 0成功，-1失败
 */
int coro_context_init(coro_context_t* ctx, void* stack_base, size_t stack_size,
                      void (*entry_point)(void*), void* arg);

/**
 * @brief 切换协程上下文
 *
 * @param from_ctx 当前上下文 (NULL表示不保存)
 * @param to_ctx 目标上下文
 */
void coro_context_switch(coro_context_t* from_ctx, coro_context_t* to_ctx);

/**
 * @brief 验证上下文
 *
 * @param ctx 上下文指针
 * @return 1有效，0无效
 */
int coro_context_validate(const coro_context_t* ctx);

/**
 * @brief 获取上下文架构
 *
 * @param ctx 上下文指针
 * @return CPU架构
 */
cpu_arch_t coro_context_get_arch(const coro_context_t* ctx);

// ============================================================================
// 平台特定的实现声明
// ============================================================================

// 平台相关的上下文初始化和切换函数
// 这些函数由各个平台提供具体的实现
// 注意：这些函数操作的是coro_context_t中的platform union成员
void platform_context_init(void* platform_ctx, void* stack_base, size_t stack_size,
                          void (*entry_point)(void*), void* arg);
void platform_context_switch(void* from_platform_ctx, void* to_platform_ctx);

#ifdef __x86_64__
#include "x86_64/context.h"
#endif

#ifdef __aarch64__
#include "aarch64/context.h"
#endif

#endif // CONTEXT_PLATFORM_H
