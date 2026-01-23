/**
 * @file context_switch.S
 * @brief aarch64 汇编实现的上下文切换
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
 * 参数 (按照ARM64 ABI):
 *   x0: from_ctx (context_t*)
 *   x1: to_ctx (context_t*)
 *
 * 注意：函数使用ARM64 ABI调用约定
 */
context_switch:
    // 保存当前上下文 (如果from_ctx不为NULL)
    cbz     x0, 1f              // 如果from_ctx为NULL，跳过保存

    // 保存callee-saved寄存器
    str     x19, [x0, #0]       // x19
    str     x20, [x0, #8]       // x20
    str     x21, [x0, #16]      // x21
    str     x22, [x0, #24]      // x22
    str     x23, [x0, #32]      // x23
    str     x24, [x0, #40]      // x24
    str     x25, [x0, #48]      // x25
    str     x26, [x0, #56]      // x26
    str     x27, [x0, #64]      // x27
    str     x28, [x0, #72]      // x28

    // 保存栈指针和链接寄存器
    mov     x2, sp
    str     x2, [x0, #80]       // sp
    str     lr, [x0, #88]       // lr

1:
    // 恢复目标上下文
    // 恢复callee-saved寄存器
    ldr     x19, [x1, #0]       // x19
    ldr     x20, [x1, #8]       // x20
    ldr     x21, [x1, #16]      // x21
    ldr     x22, [x1, #24]      // x22
    ldr     x23, [x1, #32]      // x23
    ldr     x24, [x1, #40]      // x24
    ldr     x25, [x1, #48]      // x25
    ldr     x26, [x1, #56]      // x26
    ldr     x27, [x1, #64]      // x27
    ldr     x28, [x1, #72]      // x28

    // 恢复栈指针
    ldr     x2, [x1, #80]       // sp
    mov     sp, x2

    // 恢复链接寄存器
    ldr     lr, [x1, #88]       // lr

    // 返回 (实际上跳转到lr指向的地址)
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
    mov     x0, sp
    ret

/**
 * uint64_t context_get_ip(void)
 *
 * 获取当前指令指针 (近似值，使用lr)
 */
.global context_get_ip
context_get_ip:
    mov     x0, lr
    ret
