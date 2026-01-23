/**
 * @file platform_detect.c
 * @brief 运行时平台检测实现
 *
 * 职责：检测当前运行的操作系统平台，选择合适的I/O多路复用实现
 */

#include "echo_event.h"
#include <stdio.h>
#include <string.h>

#if defined(__linux__)
#include <sys/utsname.h>
#endif

/**
 * @brief 检测当前运行平台
 */
platform_type_t detect_platform(void) {
#if defined(__linux__)
    return PLATFORM_LINUX;
#elif defined(__APPLE__)
    return PLATFORM_MACOS;
#elif defined(_WIN32)
    return PLATFORM_WINDOWS;
#else
    return PLATFORM_UNKNOWN;
#endif
}

/**
 * @brief 获取Linux平台操作集
 */
extern const struct platform_event_operations* get_linux_event_ops(void);

/**
 * @brief 获取macOS平台操作集
 */
extern const struct platform_event_operations* get_macos_event_ops(void);

/**
 * @brief 获取Windows平台操作集
 */
extern const struct platform_event_operations* get_windows_event_ops(void);

/**
 * @brief 获取select后备操作集
 */
extern const struct platform_event_operations* get_select_fallback_ops(void);

/**
 * @brief 获取当前平台的操作集
 */
const struct platform_event_operations* get_platform_event_ops(void) {
    platform_type_t platform = detect_platform();

    printf("DEBUG: Detected platform: ");
    switch (platform) {
        case PLATFORM_LINUX:
            printf("Linux\n");
            return get_linux_event_ops();
        case PLATFORM_MACOS:
            printf("macOS\n");
            return get_macos_event_ops();
        case PLATFORM_WINDOWS:
            printf("Windows\n");
            return get_windows_event_ops();
        default:
            printf("Unknown, using select fallback\n");
            return get_select_fallback_ops();
    }
}
