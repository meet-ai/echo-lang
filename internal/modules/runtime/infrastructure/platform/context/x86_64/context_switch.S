/**
 * @file context_switch.S
 * @brief x86_64 汇编实现的上下文切换
 *
 * 实现协程间的上下文切换，保存和恢复CPU状态
 */

.text
.global context_switch

// ============================================================================
// 上下文切换函数
// ============================================================================

/**
 * void context_switch(context_t* from_ctx, context_t* to_ctx)
 *
 * 切换上下文：
 * 1. 如果from_ctx不为NULL，保存当前CPU状态到from_ctx
 * 2. 从to_ctx恢复CPU状态
 *
 * 参数：
 *   %rdi: from_ctx (context_t*)
 *   %rsi: to_ctx (context_t*)
 *
 * 注意：函数使用System V AMD64 ABI调用约定
 */
context_switch:
    // 保存当前上下文 (如果from_ctx不为NULL)
    testq   %rdi, %rdi          // 检查from_ctx是否为NULL
    jz      1f                  // 如果为NULL，跳过保存

    // 保存callee-saved寄存器
    movq    %rbx, 0(%rdi)       // rbx
    movq    %rbp, 8(%rdi)       // rbp
    movq    %r12, 16(%rdi)      // r12
    movq    %r13, 24(%rdi)      // r13
    movq    %r14, 32(%rdi)      // r14
    movq    %r15, 40(%rdi)      // r15

    // 保存栈指针和返回地址
    movq    %rsp, 48(%rdi)      // rsp
    movq    (%rsp), %rax         // 获取栈顶的返回地址
    movq    %rax, 56(%rdi)      // rip (返回地址)

1:
    // 恢复目标上下文
    // 恢复callee-saved寄存器
    movq    0(%rsi), %rbx       // rbx
    movq    8(%rsi), %rbp       // rbp
    movq    16(%rsi), %r12      // r12
    movq    24(%rsi), %r13      // r13
    movq    32(%rsi), %r14      // r14
    movq    40(%rsi), %r15      // r15

    // 恢复栈指针
    movq    48(%rsi), %rsp      // rsp

    // 恢复指令指针 (通过ret指令)
    movq    56(%rsi), %rax      // rip
    movq    %rax, (%rsp)        // 将返回地址放到栈顶

    // 返回 (实际上跳转到rip指向的地址)
    ret

// ============================================================================
// 辅助函数
// ============================================================================

/**
 * uint64_t context_get_sp(void)
 *
 * 获取当前栈指针
 */
.global context_get_sp
context_get_sp:
    movq    %rsp, %rax
    ret

/**
 * uint64_t context_get_ip(void)
 *
 * 获取当前指令指针 (近似值)
 */
.global context_get_ip
context_get_ip:
    movq    (%rsp), %rax     // 返回地址
    ret
