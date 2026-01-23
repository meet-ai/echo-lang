/**
 * @file select_fallback.c
 * @brief select()后备事件循环实现
 *
 * 基础设施层职责：
 * - 提供跨平台兼容的select()实现
 * - 作为其他I/O多路复用机制的后备方案
 * - 确保基本功能在所有平台上可用
 */

#include "../event_loop.h"
#include <sys/select.h>
#include <sys/time.h>
#include <unistd.h>
#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

// ============================================================================
// 基础设施层：平台特定数据结构
// ============================================================================

/**
 * @brief select实现结构体
 * 基础设施模式：封装select相关的所有状态
 */
typedef struct select_impl {
    int wakeup_pipe[2];                 // 用于跨线程唤醒的管道
    fd_set read_fds;                    // 读取文件描述符集合
    fd_set write_fds;                   // 写入文件描述符集合
    fd_set error_fds;                   // 错误文件描述符集合
    int max_fd;                         // 最大文件描述符
    event_callback_t* callbacks;        // 回调函数数组
    int max_callbacks;                  // 回调数组大小
    event_mask_t* masks;                // 事件掩码数组
} select_impl_t;

// ============================================================================
// 基础设施层：内部工具函数
// ============================================================================

/**
 * @brief 创建唤醒管道
 */
static int create_wakeup_pipe(select_impl_t* impl) {
    if (pipe(impl->wakeup_pipe) != 0) {
        return -1;
    }

    // 设置读取端为非阻塞
    int flags = fcntl(impl->wakeup_pipe[0], F_GETFL, 0);
    if (flags == -1 ||
        fcntl(impl->wakeup_pipe[0], F_SETFL, flags | O_NONBLOCK) == -1) {
        close(impl->wakeup_pipe[0]);
        close(impl->wakeup_pipe[1]);
        return -1;
    }

    // 初始化文件描述符集合
    FD_ZERO(&impl->read_fds);
    FD_ZERO(&impl->write_fds);
    FD_ZERO(&impl->error_fds);

    // 添加管道到读取集合
    FD_SET(impl->wakeup_pipe[0], &impl->read_fds);
    impl->max_fd = impl->wakeup_pipe[0];

    return 0;
}

/**
 * @brief 销毁唤醒管道
 */
static void destroy_wakeup_pipe(select_impl_t* impl) {
    if (impl->wakeup_pipe[0] >= 0) {
        close(impl->wakeup_pipe[0]);
        impl->wakeup_pipe[0] = -1;
    }
    if (impl->wakeup_pipe[1] >= 0) {
        close(impl->wakeup_pipe[1]);
        impl->wakeup_pipe[1] = -1;
    }
}

/**
 * @brief 更新最大文件描述符
 */
static void update_max_fd(select_impl_t* impl) {
    impl->max_fd = impl->wakeup_pipe[0];

    for (int i = 0; i < impl->max_callbacks; i++) {
        if (impl->callbacks[i]) {
            if (i > impl->max_fd) {
                impl->max_fd = i;
            }
        }
    }
}

/**
 * @brief 将select结果转换为领域事件通知
 */
static void select_result_to_notification(
    int fd,
    select_impl_t* impl,
    event_notification_t* notification
) {
    notification->fd = fd;
    notification->type = EVENT_TYPE_READ;  // 默认类型

    // 检查是什么类型的事件
    if (FD_ISSET(fd, &impl->read_fds)) {
        notification->type = EVENT_TYPE_READ;
    } else if (FD_ISSET(fd, &impl->write_fds)) {
        notification->type = EVENT_TYPE_WRITE;
    } else if (FD_ISSET(fd, &impl->error_fds)) {
        notification->type = EVENT_TYPE_ERROR;
    }

    // 获取用户数据
    if (fd == impl->wakeup_pipe[0]) {
        notification->user_data = NULL;
    } else {
        notification->user_data = (void*)&impl->callbacks[fd];
    }
}

// ============================================================================
// 基础设施层：平台操作实现
// ============================================================================

/**
 * @brief 创建select实例
 */
static void* select_create(void) {
    select_impl_t* impl = (select_impl_t*)malloc(sizeof(select_impl_t));
    if (!impl) return NULL;

    memset(impl, 0, sizeof(select_impl_t));

    // 初始化管道描述符
    impl->wakeup_pipe[0] = -1;
    impl->wakeup_pipe[1] = -1;

    // 初始化回调数组
    impl->max_callbacks = FD_SETSIZE;  // select最大限制
    impl->callbacks = (event_callback_t*)calloc(impl->max_callbacks, sizeof(event_callback_t));
    if (!impl->callbacks) {
        fprintf(stderr, "Failed to allocate callback array\n");
        free(impl);
        return NULL;
    }

    impl->masks = (event_mask_t*)calloc(impl->max_callbacks, sizeof(event_mask_t));
    if (!impl->masks) {
        fprintf(stderr, "Failed to allocate mask array\n");
        free(impl->callbacks);
        free(impl);
        return NULL;
    }

    // 创建唤醒管道
    if (create_wakeup_pipe(impl) != 0) {
        fprintf(stderr, "Failed to create wakeup pipe\n");
        free(impl->masks);
        free(impl->callbacks);
        free(impl);
        return NULL;
    }

    printf("DEBUG: Created select fallback implementation\n");
    return impl;
}

/**
 * @brief 添加文件描述符到select监听
 */
static int select_add_fd(void* opaque, int fd, uint32_t events, void* user_data) {
    select_impl_t* impl = (select_impl_t*)opaque;

    if (fd < 0 || fd >= impl->max_callbacks) {
        fprintf(stderr, "File descriptor %d out of range\n", fd);
        return -1;
    }

    // 存储回调和掩码信息
    impl->callbacks[fd] = (event_callback_t)user_data;
    impl->masks[fd].mask = events;

    // 添加到相应的集合
    if (events & EVENT_READ) {
        FD_SET(fd, &impl->read_fds);
    }
    if (events & EVENT_WRITE) {
        FD_SET(fd, &impl->write_fds);
    }
    if (events & EVENT_ERROR) {
        FD_SET(fd, &impl->error_fds);
    }

    update_max_fd(impl);

    printf("DEBUG: Added fd %d to select with events 0x%x\n", fd, events);
    return 0;
}

/**
 * @brief 从select监听中移除文件描述符
 */
static int select_remove_fd(void* opaque, int fd) {
    select_impl_t* impl = (select_impl_t*)opaque;

    if (fd < 0 || fd >= impl->max_callbacks) {
        fprintf(stderr, "File descriptor %d out of range\n", fd);
        return -1;
    }

    // 从所有集合中移除
    FD_CLR(fd, &impl->read_fds);
    FD_CLR(fd, &impl->write_fds);
    FD_CLR(fd, &impl->error_fds);

    // 清除回调和掩码信息
    impl->callbacks[fd] = NULL;
    impl->masks[fd].mask = 0;

    update_max_fd(impl);

    printf("DEBUG: Removed fd %d from select\n", fd);
    return 0;
}

/**
 * @brief 修改文件描述符的监听事件
 */
static int select_modify_fd(void* opaque, int fd, uint32_t events) {
    select_impl_t* impl = (select_impl_t*)opaque;

    if (fd < 0 || fd >= impl->max_callbacks) {
        fprintf(stderr, "File descriptor %d out of range\n", fd);
        return -1;
    }

    // 先移除旧的事件
    FD_CLR(fd, &impl->read_fds);
    FD_CLR(fd, &impl->write_fds);
    FD_CLR(fd, &impl->error_fds);

    // 更新掩码
    impl->masks[fd].mask = events;

    // 添加新的事件
    if (events & EVENT_READ) {
        FD_SET(fd, &impl->read_fds);
    }
    if (events & EVENT_WRITE) {
        FD_SET(fd, &impl->write_fds);
    }
    if (events & EVENT_ERROR) {
        FD_SET(fd, &impl->error_fds);
    }

    printf("DEBUG: Modified fd %d in select with events 0x%x\n", fd, events);
    return 0;
}

/**
 * @brief 轮询事件
 */
static int select_poll(void* opaque, event_notification_t* notifications, int max_events, reactor_timeout_t timeout) {
    select_impl_t* impl = (select_impl_t*)opaque;

    // 复制文件描述符集合（select会修改它们）
    fd_set read_fds = impl->read_fds;
    fd_set write_fds = impl->write_fds;
    fd_set error_fds = impl->error_fds;

    // 设置超时
    struct timeval tv;
    struct timeval* tv_ptr = NULL;

    if (timeout.has_timeout) {
        tv.tv_sec = timeout.milliseconds / 1000;
        tv.tv_usec = (timeout.milliseconds % 1000) * 1000;
        tv_ptr = &tv;
    }

    int result = select(impl->max_fd + 1, &read_fds, &write_fds, &error_fds, tv_ptr);
    if (result == -1) {
        if (errno == EINTR) {
            return 0;  // 被信号中断，正常情况
        }
        fprintf(stderr, "select failed: %s\n", strerror(errno));
        return -1;
    }

    if (result == 0) {
        return 0;  // 超时
    }

    // 处理就绪的文件描述符
    int notification_count = 0;
    for (int fd = 0; fd <= impl->max_fd && notification_count < max_events; fd++) {
        bool is_ready = false;

        // 检查是否是唤醒管道
        if (fd == impl->wakeup_pipe[0] && FD_ISSET(fd, &read_fds)) {
            // 消耗管道数据
            char buf[1];
            while (read(impl->wakeup_pipe[0], buf, sizeof(buf)) > 0) {
                // 消耗所有数据
            }
            continue;  // 跳过唤醒事件
        }

        // 检查其他文件描述符
        if ((FD_ISSET(fd, &read_fds) && FD_ISSET(fd, &impl->read_fds)) ||
            (FD_ISSET(fd, &write_fds) && FD_ISSET(fd, &impl->write_fds)) ||
            (FD_ISSET(fd, &error_fds) && FD_ISSET(fd, &impl->error_fds))) {
            is_ready = true;
        }

        if (is_ready) {
            select_result_to_notification(fd, impl, &notifications[notification_count]);
            notification_count++;
        }
    }

    printf("DEBUG: select polled %d events\n", notification_count);
    return notification_count;
}

/**
 * @brief 唤醒select等待
 */
static void select_wakeup(void* opaque) {
    select_impl_t* impl = (select_impl_t*)opaque;

    char buf[1] = {1};
    if (write(impl->wakeup_pipe[1], buf, sizeof(buf)) == -1) {
        fprintf(stderr, "Failed to write to wakeup pipe: %s\n", strerror(errno));
    }
}

/**
 * @brief 销毁select实例
 */
static void select_destroy(void* opaque) {
    select_impl_t* impl = (select_impl_t*)opaque;

    if (!impl) return;

    // 销毁唤醒管道
    destroy_wakeup_pipe(impl);

    // 释放数组
    if (impl->callbacks) {
        free(impl->callbacks);
    }
    if (impl->masks) {
        free(impl->masks);
    }

    free(impl);
    printf("DEBUG: Destroyed select fallback implementation\n");
}

// ============================================================================
// 基础设施层：平台操作集导出
// ============================================================================

/**
 * @brief select操作集
 */
static const struct platform_event_operations select_ops = {
    .create = select_create,
    .add_fd = select_add_fd,
    .remove_fd = select_remove_fd,
    .modify_fd = select_modify_fd,
    .poll = select_poll,
    .wakeup = select_wakeup,
    .destroy = select_destroy,
};

/**
 * @brief 获取select后备操作集
 */
const struct platform_event_operations* get_select_fallback_ops(void) {
    return &select_ops;
}
