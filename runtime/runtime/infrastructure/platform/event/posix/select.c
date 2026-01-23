/**
 * @file select.c
 * @brief select()系统调用后备实现
 *
 * 通用兼容性方案：使用POSIX select()作为最后的后备
 * 适用于所有POSIX兼容系统，但性能较低
 */

#include "../../../../include/echo/reactor.h"
#include "../../../domain/reactor/event_loop.h"
#include "../echo_event.h"
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
 */
typedef struct select_impl {
    fd_set read_fds;                    // 读事件文件描述符集合
    fd_set write_fds;                   // 写事件文件描述符集合
    int max_fd;                         // 最大文件描述符
    int wakeup_pipe[2];                 // 用于跨线程唤醒的管道
} select_impl_t;

/**
 * @brief FD注册信息
 */
typedef struct fd_info {
    uint32_t events;
    void* user_data;
    bool active;
} fd_info_t;

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

    // 设置管道为非阻塞
    int flags = fcntl(impl->wakeup_pipe[0], F_GETFL, 0);
    if (flags == -1) {
        close(impl->wakeup_pipe[0]);
        close(impl->wakeup_pipe[1]);
        return -1;
    }

    if (fcntl(impl->wakeup_pipe[0], F_SETFL, flags | O_NONBLOCK) == -1) {
        close(impl->wakeup_pipe[0]);
        close(impl->wakeup_pipe[1]);
        return -1;
    }

    // 更新最大fd
    if (impl->wakeup_pipe[0] > impl->max_fd) {
        impl->max_fd = impl->wakeup_pipe[0];
    }
    if (impl->wakeup_pipe[1] > impl->max_fd) {
        impl->max_fd = impl->wakeup_pipe[1];
    }

    return 0;
}



// ============================================================================
// 基础设施层：平台操作实现
// ============================================================================

/**
 * @brief 创建select实现
 */
static void* select_create(void) {
    select_impl_t* impl = (select_impl_t*)malloc(sizeof(select_impl_t));
    if (!impl) {
        return NULL;
    }

    memset(impl, 0, sizeof(select_impl_t));
    impl->max_fd = 0;

    // 初始化fd集合
    FD_ZERO(&impl->read_fds);
    FD_ZERO(&impl->write_fds);

    // 创建唤醒管道
    if (create_wakeup_pipe(impl) != 0) {
        free(impl);
        return NULL;
    }

    // 将管道读端加入读集合
    FD_SET(impl->wakeup_pipe[0], &impl->read_fds);

    printf("DEBUG: Created select implementation\n");
    return impl;
}

/**
 * @brief 添加文件描述符事件
 */
static int select_add_fd(void* state, int fd, uint32_t events, void* user_data) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 添加到对应的fd集合
    if (events & REACTOR_EVENT_READ) {
        FD_SET(fd, &impl->read_fds);
    }
    if (events & REACTOR_EVENT_WRITE) {
        FD_SET(fd, &impl->write_fds);
    }

    // 更新最大fd
    if (fd > impl->max_fd) {
        impl->max_fd = fd;
    }

    printf("DEBUG: Added fd %d to select (events: 0x%x)\n", fd, events);
    return 0;
}

/**
 * @brief 移除文件描述符事件
 */
static int select_remove_fd(void* state, int fd) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    FD_CLR(fd, &impl->read_fds);
    FD_CLR(fd, &impl->write_fds);

    printf("DEBUG: Removed fd %d from select\n", fd);
    return 0;
}

/**
 * @brief 修改文件描述符事件
 */
static int select_modify_fd(void* state, int fd, uint32_t events) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 先清除
    FD_CLR(fd, &impl->read_fds);
    FD_CLR(fd, &impl->write_fds);

    // 重新设置
    if (events & REACTOR_EVENT_READ) {
        FD_SET(fd, &impl->read_fds);
    }
    if (events & REACTOR_EVENT_WRITE) {
        FD_SET(fd, &impl->write_fds);
    }

    printf("DEBUG: Modified fd %d in select (new events: 0x%x)\n", fd, events);
    return 0;
}

/**
 * @brief 轮询事件
 */
static int select_poll(void* state, reactor_event_t* events, int max_events, reactor_timeout_t timeout) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl) {
        return -1;
    }

    // 复制fd集合（select会修改它们）
    fd_set read_fds = impl->read_fds;
    fd_set write_fds = impl->write_fds;

    // 计算超时时间
    struct timeval tv;
    struct timeval* tv_ptr = NULL;

    // 简化实现：假设timeout.milliseconds为相对超时时间
    if (timeout.milliseconds > 0) {
        tv.tv_sec = timeout.milliseconds / 1000;
        tv.tv_usec = (timeout.milliseconds % 1000) * 1000;
        tv_ptr = &tv;
    }

    int result = select(impl->max_fd + 1, &read_fds, &write_fds, NULL, tv_ptr);
    if (result < 0) {
        if (errno == EINTR) {
            return 0; // 被信号中断
        }
        return -1;
    }

    if (result == 0) {
        return 0; // 超时
    }

    int processed = 0;

    // 检查唤醒管道
    if (FD_ISSET(impl->wakeup_pipe[0], &read_fds)) {
        char buf[1];
        while (read(impl->wakeup_pipe[0], buf, sizeof(buf)) > 0) {
            // 清空管道
        }
        result--; // 减去唤醒管道事件
    }

    // 检查其他fd
    for (int fd = 0; fd <= impl->max_fd && processed < max_events && result > 0; fd++) {
        uint32_t event_flags = 0;

        if (FD_ISSET(fd, &read_fds)) {
            event_flags |= REACTOR_EVENT_READ;
        }
        if (FD_ISSET(fd, &write_fds)) {
            event_flags |= REACTOR_EVENT_WRITE;
        }

        if (event_flags != 0) {
            events[processed].fd = fd;
            events[processed].type = (reactor_event_type_t)event_flags;
            events[processed].user_data = NULL; // 简化实现
            processed++;
            result--;
        }
    }

    return processed;
}

/**
 * @brief 唤醒事件循环
 */
static void select_wakeup(void* state) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl) {
        return;
    }

    char buf[1] = {1};
    write(impl->wakeup_pipe[1], buf, sizeof(buf));
}

/**
 * @brief 销毁select实现
 */
static void select_destroy(void* state) {
    select_impl_t* impl = (select_impl_t*)state;
    if (!impl) {
        return;
    }

    close(impl->wakeup_pipe[0]);
    close(impl->wakeup_pipe[1]);
    free(impl);

    printf("DEBUG: Destroyed select implementation\n");
}

// ============================================================================
// 基础设施层：操作集导出
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
 * @brief 获取select操作集
 */
const struct platform_event_operations* get_select_ops(void) {
    return &select_ops;
}
