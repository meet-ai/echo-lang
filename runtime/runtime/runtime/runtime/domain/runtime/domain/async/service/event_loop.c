/**
 * @file event_loop.c
 * @brief EventLoop 实现
 *
 * 实现完整的异步事件循环，支持定时器、文件描述符、信号等事件。
 */

#include "event_loop.h"
#include "../../platform/event/echo_event.h"  // 平台抽象层
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <sys/time.h>
#include <signal.h>
#include <errno.h>
#include <fcntl.h>
// #include <sys/epoll.h>  // for EPOLLIN, EPOLLOUT (Linux only)

// 定义epoll事件常量（跨平台兼容）
#ifndef EPOLLIN
#define EPOLLIN 0x001
#endif

#ifndef EPOLLOUT
#define EPOLLOUT 0x004
#endif

// ============================================================================
// 内部辅助函数
// ============================================================================

/**
 * @brief 获取当前时间
 */
static struct timespec get_current_time(void) {
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts;
}

/**
 * @brief 计算时间差（毫秒）
 */
static long long timespec_diff_ms(struct timespec start, struct timespec end) {
    long long diff_sec = end.tv_sec - start.tv_sec;
    long long diff_nsec = end.tv_nsec - start.tv_nsec;
    return diff_sec * 1000 + diff_nsec / 1000000;
}

/**
 * @brief 检查定时器是否到期
 */
static bool timer_is_expired(timer_event_t* timer) {
    struct timespec now = get_current_time();
    if (now.tv_sec > timer->deadline.tv_sec ||
        (now.tv_sec == timer->deadline.tv_sec && now.tv_nsec >= timer->deadline.tv_nsec)) {
        return true;
    }
    return false;
}

/**
 * @brief 更新周期性定时器
 */
static void timer_update_periodic(timer_event_t* timer) {
    if (timer->periodic) {
        // 计算下一次触发时间
        timer->deadline.tv_sec += timer->interval.tv_sec;
        timer->deadline.tv_nsec += timer->interval.tv_nsec;

        if (timer->deadline.tv_nsec >= 1000000000) {
            timer->deadline.tv_sec += 1;
            timer->deadline.tv_nsec -= 1000000000;
        }
    }
}

/**
 * @brief 创建事件句柄
 */
static event_handle_t* create_event_handle(event_type_t type, event_callback_t callback, void* user_data) {
    static uint64_t next_id = 1;

    event_handle_t* handle = (event_handle_t*)malloc(sizeof(event_handle_t));
    if (!handle) return NULL;

    handle->id = next_id++;
    handle->type = type;
    handle->user_data = user_data;
    handle->callback = callback;
    handle->next = NULL;

    return handle;
}

/**
 * @brief 销毁事件句柄
 */
static void destroy_event_handle(event_handle_t* handle) {
    free(handle);
}

/**
 * @brief 添加事件到链表
 */
static void add_event_to_list(event_loop_t* loop, event_handle_t* event) {
    event->next = loop->events;
    loop->events = event;
    loop->total_events++;
}

/**
 * @brief 从链表移除事件
 */
static void remove_event_from_list(event_loop_t* loop, event_handle_t* event) {
    event_handle_t** curr = &loop->events;
    while (*curr) {
        if (*curr == event) {
            *curr = event->next;
            event->next = NULL;
            return;
        }
        curr = &(*curr)->next;
    }
}

/**
 * @brief 处理到期定时器
 */
static int process_expired_timers(event_loop_t* loop) {
    int processed = 0;
    event_handle_t* curr = loop->events;

    while (curr) {
        event_handle_t* next = curr->next;  // 先保存next指针

        if (curr->type == EVENT_TYPE_TIMER) {
            timer_event_t* timer = (timer_event_t*)curr;
            if (timer_is_expired(timer)) {
                // 执行回调
                if (timer->handle.callback) {
                    timer->handle.callback(timer->handle.user_data);
                }

                // 更新周期性定时器或移除
                if (timer->periodic) {
                    timer_update_periodic(timer);
                    processed++;
                    loop->processed_events++;
                } else {
                    remove_event_from_list(loop, curr);
                    destroy_event_handle(curr);
                    curr = next;  // 移动到下一个节点
                    continue;     // 跳过curr = curr->next
                }
            }
        }
        curr = next;  // 移动到下一个节点
    }

    return processed;
}

/**
 * @brief 处理文件描述符事件
 * 使用平台抽象层进行真正的FD事件处理
 */
static int process_fd_events(event_loop_t* loop) {
    if (!loop->platform_ops || !loop->platform_state) {
        return 0;
    }

    // 使用平台抽象层轮询事件
    reactor_event_t events[MAX_EVENTS];
    reactor_timeout_t timeout = { .milliseconds = 0 }; // 非阻塞

    int num_events = loop->platform_ops->poll(
        loop->platform_state, events, MAX_EVENTS, timeout
    );

    if (num_events < 0) {
        fprintf(stderr, "ERROR: Failed to poll events: %s\n", strerror(errno));
        return -1;
    }

    // 处理每个事件
    for (int i = 0; i < num_events; i++) {
        reactor_event_t* event = &events[i];
        event_handle_t* handle = find_event_handle(loop, event->fd);

        if (!handle) {
            fprintf(stderr, "WARNING: No handle found for fd %d\n", event->fd);
            continue;
        }

        // 构造通知结构
        event_notification_t notification = {
            .fd = event->fd,
            .events = event->type,
            .user_data = handle->user_data
        };

        // 调用回调函数
        if (handle->callback) {
            handle->callback(&notification);
        }
    }

    return num_events;
}

/**
 * @brief 处理信号事件
 * 提供基本的信号处理支持
 */
static int process_signal_events(event_loop_t* loop) {
    // 检查是否有待处理的信号
    // 注意：这是一个简化的实现
    // 在生产环境中，应该使用signalfd或类似的机制

    sigset_t pending;
    if (sigpending(&pending) == 0) {
        // 检查我们关心的信号
        if (sigismember(&pending, SIGINT)) {
            printf("DEBUG: Received SIGINT signal\n");
            // 处理SIGINT信号
            loop->should_stop = true;
            return 1;
        }

        if (sigismember(&pending, SIGTERM)) {
            printf("DEBUG: Received SIGTERM signal\n");
            // 处理SIGTERM信号
            loop->should_stop = true;
            return 1;
        }

        if (sigismember(&pending, SIGHUP)) {
            printf("DEBUG: Received SIGHUP signal\n");
            // 处理SIGHUP信号（通常用于重新加载配置）
            return 1;
        }
    }

    return 0;
}

/**
 * @brief 检查唤醒管道
 */
static bool check_wakeup_pipe(event_loop_t* loop) {
    char buf[1];
    ssize_t n = read(loop->wakeup_fd[0], buf, sizeof(buf));

    // 非阻塞读取：如果没有数据可读，read返回-1且errno为EAGAIN或EWOULDBLOCK
    if (n < 0) {
        if (errno == EAGAIN || errno == EWOULDBLOCK) {
            return false;  // 没有数据可读
        }
        // 其他错误，暂时忽略
        return false;
    }

    return n > 0;
}

// ============================================================================
// EventLoop 接口实现
// ============================================================================

/**
 * @brief 创建EventLoop
 */
event_loop_t* event_loop_create(void) {
    event_loop_t* loop = (event_loop_t*)malloc(sizeof(event_loop_t));
    if (!loop) return NULL;

    memset(loop, 0, sizeof(event_loop_t));

    // 初始化平台抽象层
    loop->platform_ops = get_platform_event_ops();
    if (loop->platform_ops) {
        loop->platform_state = loop->platform_ops->create();
        if (!loop->platform_state) {
            free(loop);
            return NULL;
        }
    }

    // 创建唤醒管道
    if (pipe(loop->wakeup_fd) != 0) {
        if (loop->platform_ops && loop->platform_state) {
            loop->platform_ops->destroy(loop->platform_state);
        }
        free(loop);
        return NULL;
    }

    // 设置读取端为非阻塞模式
    int flags = fcntl(loop->wakeup_fd[0], F_GETFL, 0);
    if (flags != -1) {
        fcntl(loop->wakeup_fd[0], F_SETFL, flags | O_NONBLOCK);
    }

    return loop;
}

/**
 * @brief 销毁EventLoop
 */
void event_loop_destroy(event_loop_t* loop) {
    if (!loop) return;

    // 停止循环
    event_loop_stop(loop);

    // 清理所有事件
    event_handle_t* curr = loop->events;
    while (curr) {
        event_handle_t* next = curr->next;
        destroy_event_handle(curr);
        curr = next;
    }

    // 关闭唤醒管道
    close(loop->wakeup_fd[0]);
    close(loop->wakeup_fd[1]);

    // 清理平台状态
    if (loop->platform_ops && loop->platform_state) {
        loop->platform_ops->destroy(loop->platform_state);
        loop->platform_state = NULL;
    }

    free(loop);
    printf("DEBUG: Destroyed EventLoop\n");
}

/**
 * @brief 启动EventLoop
 */
bool event_loop_start(event_loop_t* loop) {
    if (!loop || loop->running) return false;
    loop->running = true;
    return true;
}

/**
 * @brief 停止EventLoop
 */
void event_loop_stop(event_loop_t* loop) {
    if (!loop) return;
    loop->running = false;
    // 唤醒一次确保退出
    event_loop_wakeup(loop);
}

/**
 * @brief 轮询事件（非阻塞）
 */
int event_loop_poll(event_loop_t* loop, int timeout_ms) {
    if (!loop || !loop->running) return 0;

    int processed = 0;

    // 处理到期定时器
    processed += process_expired_timers(loop);

    // 处理文件描述符事件（使用平台抽象层）
    processed += process_fd_events(loop);

    // 处理信号事件
    processed += process_signal_events(loop);

    // 检查唤醒管道
    if (check_wakeup_pipe(loop)) {
        processed++;
    }

    return processed;
}

/**
 * @brief 唤醒EventLoop
 */
void event_loop_wakeup(event_loop_t* loop) {
    if (!loop) return;

    // 向唤醒管道写入数据
    char buf[1] = {1};
    write(loop->wakeup_fd[1], buf, sizeof(buf));
}

/**
 * @brief 注册定时器事件
 */
event_handle_t* event_loop_add_timer(
    event_loop_t* loop,
    struct timespec deadline,
    event_callback_t callback,
    void* user_data
) {
    if (!loop) return NULL;

    timer_event_t* timer = (timer_event_t*)malloc(sizeof(timer_event_t));
    if (!timer) return NULL;

    timer->handle = *create_event_handle(EVENT_TYPE_TIMER, callback, user_data);
    timer->deadline = deadline;
    timer->periodic = false;

    add_event_to_list(loop, &timer->handle);
    return &timer->handle;
}

/**
 * @brief 注册周期性定时器事件
 */
event_handle_t* event_loop_add_periodic_timer(
    event_loop_t* loop,
    struct timespec interval,
    event_callback_t callback,
    void* user_data
) {
    if (!loop) return NULL;

    timer_event_t* timer = (timer_event_t*)malloc(sizeof(timer_event_t));
    if (!timer) return NULL;

    // 设置第一次触发时间为当前时间+间隔
    struct timespec now = get_current_time();
    struct timespec deadline = {
        .tv_sec = now.tv_sec + interval.tv_sec,
        .tv_nsec = now.tv_nsec + interval.tv_nsec
    };

    if (deadline.tv_nsec >= 1000000000) {
        deadline.tv_sec += 1;
        deadline.tv_nsec -= 1000000000;
    }

    timer->handle = *create_event_handle(EVENT_TYPE_TIMER, callback, user_data);
    timer->deadline = deadline;
    timer->periodic = true;
    timer->interval = interval;

    add_event_to_list(loop, &timer->handle);
    return &timer->handle;
}

/**
 * @brief 注册文件描述符事件
 */
event_handle_t* event_loop_add_fd_event(
    event_loop_t* loop,
    int fd,
    uint32_t events,
    event_callback_t callback,
    void* user_data
) {
    if (!loop) return NULL;

    // 使用平台抽象层注册FD事件
    if (loop->platform_ops && loop->platform_state) {
        int result = loop->platform_ops->add_fd(
            loop->platform_state, fd, events, user_data
        );

        if (result != 0) {
            fprintf(stderr, "ERROR: Failed to add FD event for fd %d\n", fd);
            return NULL;
        }
    }

    // 创建本地事件句柄用于管理
    fd_event_t* fd_event = (fd_event_t*)malloc(sizeof(fd_event_t));
    if (!fd_event) return NULL;

    fd_event->handle = *create_event_handle(EVENT_FD, callback, user_data);
    fd_event->fd = fd;
    fd_event->events = events;
    fd_event->triggered_events = 0;

    add_event_to_list(loop, &fd_event->handle);
    return &fd_event->handle;
}

/**
 * @brief 注册信号事件
 */
event_handle_t* event_loop_add_signal_event(
    event_loop_t* loop,
    int signum,
    event_callback_t callback,
    void* user_data
) {
    if (!loop) return NULL;

    signal_event_t* sig_event = (signal_event_t*)malloc(sizeof(signal_event_t));
    if (!sig_event) return NULL;

    sig_event->handle = *create_event_handle(EVENT_TYPE_SIGNAL, callback, user_data);
    sig_event->signum = signum;

    add_event_to_list(loop, &sig_event->handle);
    return &sig_event->handle;
}

/**
 * @brief 移除事件
 */
bool event_loop_remove_event(event_loop_t* loop, event_handle_t* handle) {
    if (!loop || !handle) return false;

    // 使用平台抽象层移除事件
    if (loop->platform_ops && loop->platform_state) {
        if (handle->type == EVENT_FD) {
            fd_event_t* fd_event = (fd_event_t*)handle;
            loop->platform_ops->remove_fd(loop->platform_state, fd_event->fd);
        }
        // 其他事件类型可以在这里扩展
    }

    remove_event_from_list(loop, handle);
    destroy_event_handle(handle);
    return true;
}

/**
 * @brief 修改文件描述符事件
 */
bool event_loop_modify_fd_event(event_loop_t* loop, event_handle_t* handle, uint32_t new_events) {
    if (!loop || !handle) return false;

    // 使用平台抽象层修改FD事件
    if (loop->platform_ops && loop->platform_state) {
        fd_event_t* fd_event = (fd_event_t*)handle;
        int result = loop->platform_ops->modify_fd(
            loop->platform_state, fd_event->fd, new_events
        );

        if (result != 0) {
            fprintf(stderr, "ERROR: Failed to modify FD event for fd %d\n", fd_event->fd);
            return false;
        }
    }

    // 更新本地状态
    fd_event_t* fd_event = (fd_event_t*)handle;
    fd_event->events = new_events;

    // 更新事件类型
    if (new_events & EVENT_WRITE) {
        handle->type = EVENT_WRITE;
    } else {
        handle->type = EVENT_READ;
    }

    return true;
}

/**
 * @brief 获取统计信息
 */
void event_loop_get_stats(event_loop_t* loop, uint64_t* total, uint64_t* processed) {
    if (!loop) return;

    if (total) *total = loop->total_events;
    if (processed) *processed = loop->processed_events;
}
