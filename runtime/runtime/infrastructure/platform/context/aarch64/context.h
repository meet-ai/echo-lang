/**
 * @file context.h
 * @brief aarch64 平台上下文结构定义
 *
 * 定义协程上下文的结构，包含CPU寄存器状态和栈信息
 */

#ifndef CONTEXT_H
#define CONTEXT_H

#include <stdint.h>
#include <stddef.h>

// ============================================================================
// 数据结构定义
// ============================================================================

/**
 * @brief 协程上下文结构
 *
 * 保存协程切换时需要保存的所有CPU状态
 */
typedef struct context {
    // 通用寄存器 (按照ARM64 ABI，x19-x28为callee-saved)
    uint64_t x19;    // callee-saved
    uint64_t x20;    // callee-saved
    uint64_t x21;    // callee-saved
    uint64_t x22;    // callee-saved
    uint64_t x23;    // callee-saved
    uint64_t x24;    // callee-saved
    uint64_t x25;    // callee-saved
    uint64_t x26;    // callee-saved
    uint64_t x27;    // callee-saved
    uint64_t x28;    // callee-saved

    // 栈指针和链接寄存器
    uint64_t sp;     // stack pointer
    uint64_t lr;     // link register (return address)

    // 浮点状态 (可选，用于支持浮点操作)
    // ARM64的浮点状态比较复杂，这里暂时省略

    // 栈信息 (用于调试和栈增长检查)
    uint64_t stack_base;   // 栈底地址
    uint64_t stack_size;   // 栈大小

    // 上下文状态
    int state;             // 0: 未初始化, 1: 已初始化, 2: 运行中, 3: 已完成
} platform_context_t;

// ============================================================================
// 常量定义
// ============================================================================

/** 上下文状态常量 */
#define CONTEXT_STATE_UNINITIALIZED 0
#define CONTEXT_STATE_INITIALIZED   1
#define CONTEXT_STATE_RUNNING       2
#define CONTEXT_STATE_COMPLETED     3

/** 默认栈大小 (64KB) */
#define DEFAULT_STACK_SIZE (64 * 1024)

/** 栈对齐要求 (16字节对齐，符合ARM64 ABI) */
#define STACK_ALIGNMENT 16

// ============================================================================
// 函数声明
// ============================================================================

/**
 * @brief 初始化上下文
 *
 * @param ctx 上下文指针
 * @param stack_base 栈底地址
 * @param stack_size 栈大小
 * @param entry_point 入口函数
 * @param arg 函数参数
 */
void context_init(platform_context_t* ctx, void* stack_base, size_t stack_size,
                  void (*entry_point)(void*), void* arg);

/**
 * @brief 切换上下文
 *
 * 保存当前上下文到from_ctx，恢复to_ctx的上下文
 *
 * @param from_ctx 当前上下文 (NULL表示不保存)
 * @param to_ctx 目标上下文
 */
void context_switch(platform_context_t* from_ctx, platform_context_t* to_ctx);

/**
 * @brief 获取当前栈指针
 *
 * @return 当前栈指针值
 */
uint64_t context_get_sp(void);

/**
 * @brief 获取当前指令指针
 *
 * @return 当前指令指针值
 */
uint64_t context_get_ip(void);

/**
 * @brief 验证上下文结构
 *
 * @param ctx 上下文指针
 * @return 1表示有效，0表示无效
 */
int context_validate(const platform_context_t* ctx);

#endif // CONTEXT_H
