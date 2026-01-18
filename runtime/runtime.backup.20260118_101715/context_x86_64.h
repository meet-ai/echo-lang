// context_x86_64.h - x86_64架构上下文结构体定义

#ifndef CONTEXT_X86_64_H
#define CONTEXT_X86_64_H

#include <stdint.h>
#include <setjmp.h>

// x86_64上下文结构体
typedef struct coro_context {
    void* rsp;           // 栈指针
    void* rbp;           // 基址指针
    void* rbx;           // 被调用者保存寄存器
    void* r12;           // 被调用者保存寄存器
    void* r13;           // 被调用者保存寄存器
    void* r14;           // 被调用者保存寄存器
    void* r15;           // 被调用者保存寄存器
    void* rip;           // 指令指针
    jmp_buf* jmp_env;    // setjmp/longjmp环境（临时实现）
} coro_context_t;

#endif // CONTEXT_X86_64_H
