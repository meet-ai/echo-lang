/**
 * @file epoll.c
 * @brief Linux epoll事件循环基础设施实现
 *
 * 基础设施层职责：
 * - 实现平台特定的技术细节
 * - 封装操作系统API调用
 * - 提供给领域层使用的适配器接口
 */

#include "../../../../../include/echo/reactor.h"
#include <sys/epoll.h>
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
 * @brief epoll实现结构体
 * 基础设施模式：封装epoll相关的所有状态
 */
typedef struct epoll_impl {
    int epfd;                           // epoll文件描述符
    int wakeup_pipe[2];                 // 用于跨线程唤醒的管道
    event_callback_t* callbacks;        // 回调函数数组
    int max_callbacks;                  // 回调数组大小
} epoll_impl_t;

/**
 * @brief 注册信息结构体
 * 用于跟踪每个fd的注册状态
 */
typedef struct registration_info {
    event_mask_t mask;
    event_callback_t callback;
    void* user_data;
    bool active;
} registration_info_t;

// ============================================================================
// 基础设施层：内部工具函数
// ============================================================================

/**
 * @brief 创建唤醒管道
 */
static int create_wakeup_pipe(epoll_impl_t* impl) {
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

    // 将管道读端添加到epoll
    struct epoll_event ev;
    memset(&ev, 0, sizeof(ev));
    ev.events = EPOLLIN;
    ev.data.ptr = NULL;  // 特殊标记，表示这是唤醒管道

    if (epoll_ctl(impl->epfd, EPOLL_CTL_ADD, impl->wakeup_pipe[0], &ev) == -1) {
        close(impl->wakeup_pipe[0]);
        close(impl->wakeup_pipe[1]);
        return -1;
    }

    return 0;
}

/**
 * @brief 将领域层事件掩码转换为epoll事件标志
 */
static uint32_t mask_to_epoll_events(event_mask_t mask) {
    uint32_t events = 0;

    if (mask.mask & EVENT_READ) {
        events |= EPOLLIN;
    }
    if (mask.mask & EVENT_WRITE) {
        events |= EPOLLOUT;
    }

    return events;
}

/**
 * @brief 将epoll事件转换为领域层事件
 */
static event_mask_t epoll_events_to_mask(uint32_t epoll_events) {
    event_mask_t mask = {0};

    if (epoll_events & EPOLLIN) {
        mask.mask |= EVENT_READ;
    }
    if (epoll_events & EPOLLOUT) {
        mask.mask |= EVENT_WRITE;
    }
    if (epoll_events & EPOLLERR) {
        mask.mask |= EVENT_ERROR;
    }

    return mask;
}

/**
 * @brief 扩展回调数组
 */
static int expand_callbacks(epoll_impl_t* impl, int new_size) {
    event_callback_t* new_callbacks = (event_callback_t*)realloc(
        impl->callbacks, new_size * sizeof(event_callback_t));

    if (!new_callbacks) {
        return -1;
    }

    // 初始化新分配的内存
    memset(&new_callbacks[impl->max_callbacks], 0,
           (new_size - impl->max_callbacks) * sizeof(event_callback_t));

    impl->callbacks = new_callbacks;
    impl->max_callbacks = new_size;

    return 0;
}

/**
 * @brief 获取或创建回调索引
 */
static int get_callback_index(epoll_impl_t* impl, event_callback_t callback, void* user_data) {
    // 查找现有回调
    for (int i = 0; i < impl->max_callbacks; i++) {
        if (impl->callbacks[i] == callback) {
            // 简化检查，实际应该比较user_data等
            return i;
        }
    }

    // 需要扩展数组
    if (expand_callbacks(impl, impl->max_callbacks * 2) != 0) {
        return -1;
    }

    // 找到空闲位置
    for (int i = 0; i < impl->max_callbacks; i++) {
        if (impl->callbacks[i] == NULL) {
            impl->callbacks[i] = callback;
            return i;
        }
    }

    return -1;
}

// ============================================================================
// 基础设施层：平台操作实现
// ============================================================================

/**
 * @brief 创建epoll实现
 */
static void* epoll_create(void) {
    epoll_impl_t* impl = (epoll_impl_t*)malloc(sizeof(epoll_impl_t));
    if (!impl) {
        return NULL;
    }

    memset(impl, 0, sizeof(epoll_impl_t));

    // 创建epoll实例
    impl->epfd = epoll_create1(EPOLL_CLOEXEC);
    if (impl->epfd == -1) {
        free(impl);
        return NULL;
    }

    // 初始化回调数组
    impl->max_callbacks = 1024;  // 最大支持1024个fd
    impl->callbacks = (event_callback_t*)calloc(impl->max_callbacks, sizeof(event_callback_t));
    if (!impl->callbacks) {
        close(impl->epfd);
        free(impl);
        return NULL;
    }

    // 创建唤醒管道
    if (create_wakeup_pipe(impl) != 0) {
        free(impl->callbacks);
        close(impl->epfd);
        free(impl);
        return NULL;
    }

    printf("DEBUG: Created epoll implementation with fd %d\n", impl->epfd);
    return impl;
}

/**
 * @brief 添加文件描述符事件
 */
static int epoll_add_fd(void* state, int fd, uint32_t events, event_callback_t callback, void* user_data) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 获取回调索引
    int callback_index = get_callback_index(impl, callback, user_data);
    if (callback_index < 0) {
        return -1;
    }

    // 设置epoll事件
    struct epoll_event ev;
    memset(&ev, 0, sizeof(ev));
    ev.events = mask_to_epoll_events(events);
    ev.data.u32 = callback_index;  // 使用回调索引作为标识

    int op = EPOLL_CTL_ADD;
    if (epoll_ctl(impl->epfd, EPOLL_CTL_ADD, fd, &ev) == -1) {
        if (errno == EEXIST) {
            // 已经存在，尝试修改
            op = EPOLL_CTL_MOD;
            if (epoll_ctl(impl->epfd, EPOLL_CTL_MOD, fd, &ev) == -1) {
                return -1;
            }
        } else {
            return -1;
        }
    }

    printf("DEBUG: Added fd %d to epoll (events: 0x%x)\n", fd, ev.events);
    return 0;
}

/**
 * @brief 移除文件描述符事件
 */
static int epoll_remove_fd(void* state, int fd) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    if (epoll_ctl(impl->epfd, EPOLL_CTL_DEL, fd, NULL) == -1) {
        return -1;
    }

    printf("DEBUG: Removed fd %d from epoll\n", fd);
    return 0;
}

/**
 * @brief 修改文件描述符事件
 */
static int epoll_modify_fd(void* state, int fd, uint32_t events) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    struct epoll_event ev;
    memset(&ev, 0, sizeof(ev));
    ev.events = mask_to_epoll_events(events);
    ev.data.fd = fd;

    if (epoll_ctl(impl->epfd, EPOLL_CTL_MOD, fd, &ev) == -1) {
        return -1;
    }

    printf("DEBUG: Modified fd %d in epoll (new events: 0x%x)\n", fd, ev.events);
    return 0;
}

/**
 * @brief 轮询事件
 */
static int epoll_poll(void* state, reactor_event_t* events, int max_events, reactor_timeout_t timeout) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl) {
        return -1;
    }

    struct epoll_event epoll_events[64];  // 临时缓冲区
    int max_epoll_events = sizeof(epoll_events) / sizeof(epoll_events[0]);
    if (max_events < max_epoll_events) {
        max_epoll_events = max_events;
    }

    // 计算超时时间
    int timeout_ms = -1;
    // 简化实现：假设timeout.milliseconds为相对超时时间
    if (timeout.milliseconds > 0) {
        timeout_ms = timeout.milliseconds;
    }

    int nfds = epoll_wait(impl->epfd, epoll_events, max_epoll_events, timeout_ms);
    if (nfds < 0) {
        if (errno == EINTR) {
            return 0;  // 被信号中断
        }
        return -1;
    }

    int processed = 0;
    for (int i = 0; i < nfds && processed < max_events; i++) {
        struct epoll_event* ev = &epoll_events[i];

        // 检查是否是唤醒管道事件
        if (ev->data.ptr == NULL) {
            // 唤醒管道事件，读取数据清空管道
            char buf[1];
            while (read(impl->wakeup_pipe[0], buf, sizeof(buf)) > 0) {
                // 清空管道
            }
            continue;
        }

        // 正常的事件
        int callback_index = ev->data.u32;
        if (callback_index >= 0 && callback_index < impl->max_callbacks) {
            event_callback_t callback = impl->callbacks[callback_index];
            if (callback) {
                events[processed].fd = -1;  // epoll不直接提供fd，需要从data中提取
                events[processed].type = (reactor_event_type_t)epoll_events_to_mask(ev->events).mask;
                events[processed].user_data = NULL;  // 简化实现
                processed++;
            }
        }
    }

    return processed;
}

/**
 * @brief 唤醒事件循环
 */
static void epoll_wakeup(void* state) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl) {
        return;
    }

    char buf[1] = {1};
    write(impl->wakeup_pipe[1], buf, sizeof(buf));
}

/**
 * @brief 销毁epoll实现
 */
static void epoll_destroy(void* state) {
    epoll_impl_t* impl = (epoll_impl_t*)state;
    if (!impl) {
        return;
    }

    // 关闭管道
    close(impl->wakeup_pipe[0]);
    close(impl->wakeup_pipe[1]);

    // 关闭epoll
    close(impl->epfd);

    // 释放回调数组
    free(impl->callbacks);

    free(impl);
    printf("DEBUG: Destroyed epoll implementation\n");
}

// ============================================================================
// 基础设施层：操作集导出
// ============================================================================

/**
 * @brief epoll操作集
 */
static const struct platform_event_operations epoll_ops = {
    .create = epoll_create,
    .add_fd = epoll_add_fd,
    .remove_fd = epoll_remove_fd,
    .modify_fd = epoll_modify_fd,
    .poll = epoll_poll,
    .wakeup = epoll_wakeup,
    .destroy = epoll_destroy,
};

/**
 * @brief 获取epoll操作集
 */
const struct platform_event_operations* get_epoll_ops(void) {
    return &epoll_ops;
}