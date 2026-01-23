/**
 * @file event_loop.c
 * @brief 事件循环领域层实现
 *
 * 领域层职责：
 * - 封装业务逻辑，不依赖具体平台实现
 * - 通过虚函数表实现依赖倒置
 * - 提供统一的领域接口给应用层使用
 */

#include "event_loop.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>

// ============================================================================
// 平台检测和工厂方法
// ============================================================================

/**
 * @brief 平台类型枚举
 */
typedef enum {
    PLATFORM_LINUX,
    PLATFORM_MACOS,
    PLATFORM_WINDOWS,
    PLATFORM_UNKNOWN
} platform_type_t;

/**
 * @brief 检测运行平台
 * @return 平台类型
 */
static platform_type_t detect_platform(void) {
#if defined(__linux__)
    return PLATFORM_LINUX;
#elif defined(__APPLE__) && defined(__MACH__)
    return PLATFORM_MACOS;
#elif defined(_WIN32) || defined(_WIN64)
    return PLATFORM_WINDOWS;
#else
    return PLATFORM_UNKNOWN;
#endif
}

/**
 * @brief 创建平台特定的实现
 * 工厂方法模式：根据平台创建相应的基础设施实现
 */
static void* create_platform_impl(platform_type_t platform) {
    switch (platform) {
        case PLATFORM_LINUX:
            // TODO: return epoll_create();
            fprintf(stderr, "Linux epoll implementation not available yet\n");
            return NULL;

        case PLATFORM_MACOS:
            // 导入macOS基础设施实现
            extern void* kqueue_create(void);
            return kqueue_create();

        case PLATFORM_WINDOWS:
            // TODO: return iocp_create();
            fprintf(stderr, "Windows IOCP implementation not available yet\n");
            return NULL;

        default:
            fprintf(stderr, "Unsupported platform\n");
            return NULL;
    }
}

/**
 * @brief 获取平台特定的虚函数表
 */
static const event_loop_vtable_t* get_platform_vtable(platform_type_t platform) {
    switch (platform) {
        case PLATFORM_LINUX:
            // TODO: return &epoll_vtable;
            return NULL;

        case PLATFORM_MACOS:
            // 导入macOS虚函数表
            extern const event_loop_vtable_t kqueue_vtable;
            return &kqueue_vtable;

        case PLATFORM_WINDOWS:
            // TODO: return &iocp_vtable;
            return NULL;

        default:
            return NULL;
    }
}

// ============================================================================
// 领域层：业务逻辑实现
// ============================================================================

/**
 * @brief 创建事件循环实例
 * 领域服务：封装创建逻辑，屏蔽平台差异
 */
event_loop_t* event_loop_create(void) {
    platform_type_t platform = detect_platform();

    // 创建平台特定的实现
    void* impl = create_platform_impl(platform);
    if (!impl) {
        fprintf(stderr, "Failed to create platform-specific event loop implementation\n");
        return NULL;
    }

    // 获取虚函数表
    const event_loop_vtable_t* vtable = get_platform_vtable(platform);
    if (!vtable) {
        fprintf(stderr, "Failed to get platform-specific vtable\n");
        // 清理已创建的实现
        // TODO: 调用平台的destroy函数
        return NULL;
    }

    // 创建领域对象
    event_loop_t* loop = (event_loop_t*)malloc(sizeof(event_loop_t));
    if (!loop) {
        fprintf(stderr, "Failed to allocate event_loop_t\n");
        // TODO: 清理实现
        return NULL;
    }

    loop->impl = impl;
    loop->vtable = vtable;

    printf("DEBUG: Created event loop for platform %d\n", platform);
    return loop;
}

/**
 * @brief 销毁事件循环实例
 */
void event_loop_destroy(event_loop_t* loop) {
    if (!loop) return;

    // 调用平台的销毁函数
    if (loop->vtable && loop->vtable->destroy) {
        loop->vtable->destroy(loop->impl);
    }

    free(loop);
    printf("DEBUG: Destroyed event loop\n");
}

/**
 * @brief 添加事件监听
 * 聚合根行为：封装完整的添加事件业务逻辑
 */
int event_loop_add_event(event_loop_t* loop, int fd, event_mask_t mask,
                        event_callback_t callback, void* user_data) {
    if (!loop || !loop->vtable || !loop->vtable->add_event) {
        return -1;
    }

    return loop->vtable->add_event(loop->impl, fd, mask, callback, user_data);
}

/**
 * @brief 移除事件监听
 */
int event_loop_remove_event(event_loop_t* loop, int fd) {
    if (!loop || !loop->vtable || !loop->vtable->remove_event) {
        return -1;
    }

    return loop->vtable->remove_event(loop->impl, fd);
}

/**
 * @brief 修改事件监听
 */
int event_loop_modify_event(event_loop_t* loop, int fd, event_mask_t mask) {
    if (!loop || !loop->vtable || !loop->vtable->modify_event) {
        return -1;
    }

    return loop->vtable->modify_event(loop->impl, fd, mask);
}

/**
 * @brief 执行事件轮询
 * 核心业务逻辑：封装轮询逻辑，统一错误处理
 */
int event_loop_poll(event_loop_t* loop, timeout_t* timeout,
                   event_notification_t* notifications, int max_events) {
    if (!loop || !loop->vtable || !loop->vtable->poll_events) {
        return -1;
    }

    if (!notifications || max_events <= 0) {
        errno = EINVAL;
        return -1;
    }

    return loop->vtable->poll_events(loop->impl, *timeout, notifications, max_events);
}

/**
 * @brief 唤醒事件循环
 * 跨线程唤醒机制：封装唤醒逻辑
 */
int event_loop_wakeup(event_loop_t* loop) {
    if (!loop || !loop->vtable || !loop->vtable->wakeup) {
        return -1;
    }

    return loop->vtable->wakeup(loop->impl);
}
