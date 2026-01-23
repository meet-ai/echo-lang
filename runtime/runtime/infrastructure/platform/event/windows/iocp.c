/**
 * @file iocp.c
 * @brief Windows IOCP (I/O Completion Ports) 实现
 *
 * IOCP是Windows上最高效的I/O多路复用机制
 * 支持真正的异步I/O操作
 */

#include "../../../../include/echo/reactor.h"
#include <windows.h>
#include <winsock2.h>
#include <mswsock.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// ============================================================================
// 基础设施层：平台特定数据结构
// ============================================================================

/**
 * @brief IOCP实现结构体
 */
typedef struct iocp_impl {
    HANDLE completion_port;             // IOCP句柄
    HANDLE wakeup_event;                // 用于跨线程唤醒的事件
    event_callback_t* callbacks;        // 回调函数数组
    int max_callbacks;                  // 回调数组大小
    CRITICAL_SECTION cs;                // 保护回调数组的临界区
} iocp_impl_t;

/**
 * @brief IOCP操作数据
 */
typedef struct iocp_operation {
    OVERLAPPED overlapped;
    event_callback_t callback;
    void* user_data;
    int fd;
    WSABUF buffer;
    char buffer_data[1024];  // 临时缓冲区
} iocp_operation_t;

// ============================================================================
// 基础设施层：内部工具函数
// ============================================================================

/**
 * @brief 扩展回调数组
 */
static int expand_callbacks(iocp_impl_t* impl, int new_size) {
    event_callback_t* new_callbacks = (event_callback_t*)realloc(
        impl->callbacks, new_size * sizeof(event_callback_t));

    if (!new_callbacks) {
        return -1;
    }

    memset(&new_callbacks[impl->max_callbacks], 0,
           (new_size - impl->max_callbacks) * sizeof(event_callback_t));

    impl->callbacks = new_callbacks;
    impl->max_callbacks = new_size;

    return 0;
}

/**
 * @brief 获取回调索引
 */
static int get_callback_index(iocp_impl_t* impl, event_callback_t callback) {
    EnterCriticalSection(&impl->cs);

    for (int i = 0; i < impl->max_callbacks; i++) {
        if (impl->callbacks[i] == callback) {
            LeaveCriticalSection(&impl->cs);
            return i;
        }
    }

    // 扩展数组
    if (expand_callbacks(impl, impl->max_callbacks * 2) != 0) {
        LeaveCriticalSection(&impl->cs);
        return -1;
    }

    // 找到空闲位置
    for (int i = 0; i < impl->max_callbacks; i++) {
        if (impl->callbacks[i] == NULL) {
            impl->callbacks[i] = callback;
            LeaveCriticalSection(&impl->cs);
            return i;
        }
    }

    LeaveCriticalSection(&impl->cs);
    return -1;
}

// ============================================================================
// 基础设施层：平台操作实现
// ============================================================================

/**
 * @brief 创建IOCP实现
 */
static void* iocp_create(void) {
    iocp_impl_t* impl = (iocp_impl_t*)malloc(sizeof(iocp_impl_t));
    if (!impl) {
        return NULL;
    }

    memset(impl, 0, sizeof(iocp_impl_t));

    // 创建IOCP
    impl->completion_port = CreateIoCompletionPort(
        INVALID_HANDLE_VALUE, NULL, 0, 0);

    if (impl->completion_port == NULL) {
        free(impl);
        return NULL;
    }

    // 创建唤醒事件
    impl->wakeup_event = CreateEvent(NULL, TRUE, FALSE, NULL);
    if (impl->wakeup_event == NULL) {
        CloseHandle(impl->completion_port);
        free(impl);
        return NULL;
    }

    // 初始化临界区
    InitializeCriticalSection(&impl->cs);

    // 初始化回调数组
    impl->max_callbacks = 1024;
    impl->callbacks = (event_callback_t*)calloc(impl->max_callbacks, sizeof(event_callback_t));
    if (!impl->callbacks) {
        DeleteCriticalSection(&impl->cs);
        CloseHandle(impl->wakeup_event);
        CloseHandle(impl->completion_port);
        free(impl);
        return NULL;
    }

    printf("DEBUG: Created IOCP implementation\n");
    return impl;
}

/**
 * @brief 添加文件描述符事件
 */
static int iocp_add_fd(void* state, int fd, uint32_t events, event_callback_t callback, void* user_data) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 获取回调索引
    int callback_index = get_callback_index(impl, callback);
    if (callback_index < 0) {
        return -1;
    }

    // 将socket关联到IOCP
    HANDLE result = CreateIoCompletionPort(
        (HANDLE)_get_osfhandle(fd),
        impl->completion_port,
        (ULONG_PTR)callback_index,
        0);

    if (result == NULL) {
        return -1;
    }

    // 如果是读事件，发起异步接收
    if (events & REACTOR_EVENT_READ) {
        iocp_operation_t* op = (iocp_operation_t*)malloc(sizeof(iocp_operation_t));
        if (!op) {
            return -1;
        }

        memset(op, 0, sizeof(iocp_operation_t));
        op->callback = callback;
        op->user_data = user_data;
        op->fd = fd;

        op->buffer.buf = op->buffer_data;
        op->buffer.len = sizeof(op->buffer_data);

        DWORD flags = 0;
        int ret = WSARecv(fd, &op->buffer, 1, NULL, &flags, &op->overlapped, NULL);
        if (ret == SOCKET_ERROR) {
            DWORD error = WSAGetLastError();
            if (error != WSA_IO_PENDING) {
                free(op);
                return -1;
            }
        }
    }

    printf("DEBUG: Added fd %d to IOCP (mask: 0x%x)\n", fd, mask.mask);
    return 0;
}

/**
 * @brief 移除文件描述符事件
 */
static int iocp_remove_fd(void* state, int fd) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // IOCP不支持直接移除，Windows会自动清理
    printf("DEBUG: Marked fd %d for removal from IOCP\n", fd);
    return 0;
}

/**
 * @brief 修改文件描述符事件
 */
static int iocp_modify_fd(void* state, int fd, uint32_t events) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl || fd < 0) {
        return -1;
    }

    // 先移除再添加
    iocp_remove_fd(state, fd);
    return iocp_add_fd(state, fd, mask, NULL, NULL);  // 简化实现
}

/**
 * @brief 轮询事件
 */
static int iocp_poll(void* state, reactor_event_t* events, int max_events, reactor_timeout_t timeout) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl) {
        return -1;
    }

    DWORD timeout_ms = INFINITE;
    // 简化实现：假设timeout.milliseconds为相对超时时间
    if (timeout.milliseconds > 0) {
        timeout_ms = timeout.milliseconds;
    }

    DWORD bytes_transferred;
    ULONG_PTR completion_key;
    LPOVERLAPPED overlapped;

    BOOL result = GetQueuedCompletionStatus(
        impl->completion_port,
        &bytes_transferred,
        &completion_key,
        &overlapped,
        timeout_ms);

    if (!result) {
        DWORD error = GetLastError();
        if (error == WAIT_TIMEOUT) {
            return 0;  // 超时
        }
        return -1;
    }

    if (overlapped == NULL) {
        // 特殊情况：唤醒事件
        ResetEvent(impl->wakeup_event);
        return 0;
    }

    // 处理正常事件
    iocp_operation_t* op = CONTAINING_RECORD(overlapped, iocp_operation_t, overlapped);

    if (bytes_transferred > 0) {
        events[0].fd = op->fd;
        events[0].type = REACTOR_EVENT_READ;  // 简化：假设是读事件
        events[0].user_data = op->user_data;

        // 重新发起异步接收
        DWORD flags = 0;
        WSARecv(op->fd, &op->buffer, 1, NULL, &flags, &op->overlapped, NULL);

        return 1;
    }

    return 0;
}

/**
 * @brief 唤醒事件循环
 */
static void iocp_wakeup(void* state) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl) {
        return;
    }

    // 发送特殊完成包
    PostQueuedCompletionStatus(impl->completion_port, 0, 0, NULL);
}

/**
 * @brief 销毁IOCP实现
 */
static void iocp_destroy(void* state) {
    iocp_impl_t* impl = (iocp_impl_t*)state;
    if (!impl) {
        return;
    }

    DeleteCriticalSection(&impl->cs);
    CloseHandle(impl->wakeup_event);
    CloseHandle(impl->completion_port);
    free(impl->callbacks);
    free(impl);

    printf("DEBUG: Destroyed IOCP implementation\n");
}

// ============================================================================
// 基础设施层：操作集导出
// ============================================================================

/**
 * @brief IOCP操作集
 */
static const struct platform_event_operations iocp_ops = {
    .create = iocp_create,
    .add_fd = iocp_add_fd,
    .remove_fd = iocp_remove_fd,
    .modify_fd = iocp_modify_fd,
    .poll = iocp_poll,
    .wakeup = iocp_wakeup,
    .destroy = iocp_destroy,
};

/**
 * @brief 获取IOCP操作集
 */
const struct platform_event_operations* get_iocp_ops(void) {
    return &iocp_ops;
}