/**
 * @file kqueue.c
 * @brief macOS kqueue事件循环基础设施实现
 *
 * 基础设施层职责：
 * - 实现平台特定的技术细节
 * - 封装操作系统API调用
 * - 提供给领域层使用的适配器接口
 */

#include "../../echo_event.h"
#include <sys/event.h>
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
 * @brief kqueue实现结构体
 */
typedef struct kqueue_impl {
    int kq_fd;                           // kqueue文件描述符
    int wakeup_pipe[2];                  // 用于跨线程唤醒的管道
} kqueue_impl_t;

// ============================================================================
// 基础设施层：内部工具函数
// ============================================================================

/**
 * @brief 创建唤醒管道
 */
static int create_wakeup_pipe(kqueue_impl_t* impl) {
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

    return 0;
}

/**
 * @brief 将领域层事件掩码转换为kqueue事件标志
 */
static uint16_t events_to_kqueue_filters(uint32_t events) {
    uint16_t filters = 0;

    if (events & REACTOR_EVENT_READ) {
        filters |= EVFILT_READ;
    }
    if (events & REACTOR_EVENT_WRITE) {
        filters |= EVFILT_WRITE;
    }

    return filters;
}

// ============================================================================
// 基础设施层：平台操作实现
// ============================================================================

/**
 * @brief 创建kqueue实现
 */
static void* kqueue_create(void) {
    kqueue_impl_t* impl = (kqueue_impl_t*)malloc(sizeof(kqueue_impl_t));
    if (!impl) {
        return NULL;
    }

    memset(impl, 0, sizeof(kqueue_impl_t));

    // 创建kqueue实例
    impl->kq_fd = kqueue();
    if (impl->kq_fd == -1) {
        free(impl);
        return NULL;
    }

    // 创建唤醒管道
    if (create_wakeup_pipe(impl) != 0) {
        close(impl->kq_fd);
        free(impl);
        return NULL;
    }

    // 将管道读端添加到kqueue
    struct kevent ev;
    EV_SET(&ev, impl->wakeup_pipe[0], EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, NULL);
    if (kevent(impl->kq_fd, &ev, 1, NULL, 0, NULL) == -1) {
        close(impl->wakeup_pipe[0]);
        close(impl->wakeup_pipe[1]);
        close(impl->kq_fd);
        free(impl);
        return NULL;
    }

    printf("DEBUG: Created kqueue implementation with fd %d\n", impl->kq_fd);
    return impl;
}

/**
 * @brief 添加文件描述符事件
 */
static int kqueue_add_fd(void* state, int fd, uint32_t events, void* user_data) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    uint16_t filters = events_to_kqueue_filters(events);
    if (filters == 0) {
        return -1;  // 不支持的事件类型
    }

    // 添加事件到kqueue
    struct kevent ev[2];
    int n = 0;

    if (filters & EVFILT_READ) {
        EV_SET(&ev[n++], fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, user_data);
    }
    if (filters & EVFILT_WRITE) {
        EV_SET(&ev[n++], fd, EVFILT_WRITE, EV_ADD | EV_ENABLE, 0, 0, user_data);
    }

    if (kevent(impl->kq_fd, ev, n, NULL, 0, NULL) == -1) {
        return -1;
    }

    printf("DEBUG: Added fd %d to kqueue (events: 0x%x)\n", fd, events);
    return 0;
}

/**
 * @brief 移除文件描述符事件
 */
static int kqueue_remove_fd(void* state, int fd) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 从kqueue移除事件
    struct kevent ev[2];
    int n = 0;

    EV_SET(&ev[n++], fd, EVFILT_READ, EV_DELETE, 0, 0, NULL);
    EV_SET(&ev[n++], fd, EVFILT_WRITE, EV_DELETE, 0, 0, NULL);

    kevent(impl->kq_fd, ev, n, NULL, 0, NULL);  // 忽略错误，事件可能不存在

    printf("DEBUG: Removed fd %d from kqueue\n", fd);
    return 0;
}

/**
 * @brief 修改文件描述符事件
 */
static int kqueue_modify_fd(void* state, int fd, uint32_t events) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 先移除旧事件，再添加新事件
    kqueue_remove_fd(state, fd);
    return kqueue_add_fd(state, fd, events, NULL);
}

/**
 * @brief 轮询事件
 */
static int kqueue_poll(void* state, reactor_event_t* events, int max_events, reactor_timeout_t timeout) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl) {
        return -1;
    }

    struct kevent kev_events[64];  // 临时缓冲区
    int max_kev_events = sizeof(kev_events) / sizeof(kev_events[0]);
    if (max_events < max_kev_events) {
        max_kev_events = max_events;
    }

    // 计算超时时间
    struct timespec ts;
    struct timespec* ts_ptr = NULL;

    // 简化实现：假设timeout.milliseconds为相对超时时间
    if (timeout.milliseconds > 0) {
        ts.tv_sec = timeout.milliseconds / 1000;
        ts.tv_nsec = (timeout.milliseconds % 1000) * 1000000;
        ts_ptr = &ts;
    }

    int nfds = kevent(impl->kq_fd, NULL, 0, kev_events, max_kev_events, ts_ptr);
    if (nfds < 0) {
        if (errno == EINTR) {
            return 0;  // 被信号中断
        }
        return -1;
    }

    if (nfds == 0) {
        return 0;  // 超时
    }

    int processed = 0;
    for (int i = 0; i < nfds && processed < max_events; i++) {
        struct kevent* kev = &kev_events[i];

        // 检查是否是唤醒管道事件
        if ((uintptr_t)kev->udata == 0) {
            // 唤醒管道事件，读取数据清空管道
            char buf[1];
            while (read(impl->wakeup_pipe[0], buf, sizeof(buf)) > 0) {
                // 清空管道
            }
            continue;
        }

        // 正常的事件
        events[processed].fd = (int)kev->ident;
        events[processed].type = 0;

        if (kev->filter == EVFILT_READ) {
            events[processed].type |= REACTOR_EVENT_READ;
        } else if (kev->filter == EVFILT_WRITE) {
            events[processed].type |= REACTOR_EVENT_WRITE;
        }

        if (kev->flags & EV_ERROR) {
            events[processed].type |= REACTOR_EVENT_ERROR;
        }

        events[processed].user_data = kev->udata;
        processed++;
    }

    return processed;
}

/**
 * @brief 唤醒事件循环
 */
static void kqueue_wakeup(void* state) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl) {
        return;
    }

    char buf[1] = {1};
    write(impl->wakeup_pipe[1], buf, sizeof(buf));
}

/**
 * @brief 销毁kqueue实现
 */
static void kqueue_destroy(void* state) {
    kqueue_impl_t* impl = (kqueue_impl_t*)state;
    if (!impl) {
        return;
    }

    // 关闭管道
    close(impl->wakeup_pipe[0]);
    close(impl->wakeup_pipe[1]);

    // 关闭kqueue
    close(impl->kq_fd);

    free(impl);
    printf("DEBUG: Destroyed kqueue implementation\n");
}

// ============================================================================
// 基础设施层：操作集导出
// ============================================================================

/**
 * @brief kqueue操作集
 */
static const struct platform_event_operations kqueue_ops = {
    .create = kqueue_create,
    .add_fd = kqueue_add_fd,
    .remove_fd = kqueue_remove_fd,
    .modify_fd = kqueue_modify_fd,
    .poll = kqueue_poll,
    .wakeup = kqueue_wakeup,
    .destroy = kqueue_destroy,
};

/**
 * @brief 获取kqueue操作集
 */
const struct platform_event_operations* get_kqueue_ops(void) {
    return &kqueue_ops;
}