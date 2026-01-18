// context_aarch64.h - ARM64架构上下文结构体定义

#ifndef CONTEXT_AARCH64_H
#define CONTEXT_AARCH64_H

#include <stdint.h>

// ARM64上下文结构体
typedef struct coro_context {
    void* x19;   // 被调用者保存寄存器
    void* x20;   // 被调用者保存寄存器
    void* x21;   // 被调用者保存寄存器
    void* x22;   // 被调用者保存寄存器
    void* x23;   // 被调用者保存寄存器
    void* x24;   // 被调用者保存寄存器
    void* x25;   // 被调用者保存寄存器
    void* x26;   // 被调用者保存寄存器
    void* x27;   // 被调用者保存寄存器
    void* x28;   // 被调用者保存寄存器
    void* x29;   // 帧指针 (FP)
    void* x30;   // 链接寄存器 (LR) - 相当于x86的RIP
    void* sp;    // 栈指针
} coro_context_t;

#endif // CONTEXT_AARCH64_H
