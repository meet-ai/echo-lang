/**
 * @file platform_detector.c
 * @brief 平台检测和运行时操作集选择
 *
 * 实现策略模式：根据运行时平台动态选择最优的I/O多路复用实现
 */

#include "../../../include/echo/reactor.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ============================================================================
// 前向声明：各平台操作集
// ============================================================================

// 声明各平台的操作集获取函数
#ifdef __APPLE__
extern const struct platform_event_operations* get_kqueue_ops(void);
#endif

#ifdef __linux__
extern const struct platform_event_operations* get_epoll_ops(void);
extern const struct platform_event_operations* get_io_uring_ops(void);
#endif

#ifdef _WIN32
extern const struct platform_event_operations* get_iocp_ops(void);
#endif

// 通用后备方案
extern const struct platform_event_operations* get_select_ops(void);

// ============================================================================
// 平台检测和选择逻辑
// ============================================================================

/**
 * @brief 检测Linux内核版本是否支持io_uring
 */
static bool has_io_uring_support(void) {
#ifdef __linux__
    // 检查内核版本 >= 5.1
    FILE* fp = fopen("/proc/version", "r");
    if (!fp) {
        return false;
    }

    char buffer[256];
    if (!fgets(buffer, sizeof(buffer), fp)) {
        fclose(fp);
        return false;
    }
    fclose(fp);

    // 检查是否包含版本信息 (简化检测)
    if (strstr(buffer, "Linux version 5.") || strstr(buffer, "Linux version 6.")) {
        // 进一步检查io_uring系统调用是否可用
        // 这里简化实现，实际应该尝试创建io_uring实例
        return true;
    }

    return false;
#else
    return false;
#endif
}

/**
 * @brief 获取当前平台的最优操作集
 *
 * 优先级策略：
 * 1. 平台最优实现 (epoll/kqueue/IOCP)
 * 2. 高级特性 (io_uring)
 * 3. 通用后备 (select)
 */
const struct platform_event_operations* get_platform_event_ops(void) {
#if defined(__APPLE__)
    // macOS: 使用kqueue
    printf("DEBUG: Detected macOS, using kqueue\n");
    return get_kqueue_ops();

#elif defined(__linux__)
    // Linux: 优先使用io_uring，然后epoll
    if (has_io_uring_support()) {
        printf("DEBUG: Detected Linux with io_uring support, using io_uring\n");
        return get_io_uring_ops();
    }

    printf("DEBUG: Detected Linux (no io_uring), using epoll\n");
    return get_epoll_ops();

#elif defined(_WIN32)
    // Windows: 使用IOCP
    printf("DEBUG: Detected Windows, using IOCP\n");
    return get_iocp_ops();

#endif

    // 通用后备方案
    printf("DEBUG: Unknown platform or no specific implementation, using select\n");
    return get_select_ops();
}

/**
 * @brief 获取平台信息字符串
 */
const char* get_platform_info(void) {
#if defined(__APPLE__)
    return "macOS (kqueue)";
#elif defined(__linux__)
    if (has_io_uring_support()) {
        return "Linux (io_uring)";
    } else {
        return "Linux (epoll)";
    }
#elif defined(_WIN32)
    return "Windows (IOCP)";
#else
    return "Unknown (select)";
#endif
}

/**
 * @brief 打印平台检测结果
 */
void print_platform_detection(void) {
    printf("Echo Runtime Platform Detection:\n");
    printf("================================\n");
    printf("Platform: %s\n", get_platform_info());
    printf("Operation set: %p\n", (void*)get_platform_event_ops());
    printf("\n");
}