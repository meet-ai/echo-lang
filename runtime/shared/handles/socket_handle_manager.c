#include "socket_handle_manager.h"
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

// 全局 Socket 句柄表（单例）
static SocketHandleTable* g_socket_handle_table = NULL;
static bool g_initialized = false;

SocketHandleTable* socket_handle_table_create(void) {
    SocketHandleTable* table = (SocketHandleTable*)calloc(1, sizeof(SocketHandleTable));
    if (!table) {
        return NULL;
    }

    // 初始化句柄表
    for (int i = 0; i < 1024; i++) {
        table->sockets[i] = -1; // -1 表示无效套接字
        table->used[i] = false;
        table->is_listener[i] = false;
    }

    return table;
}

void socket_handle_table_destroy(SocketHandleTable* table) {
    if (!table) {
        return;
    }

    // 关闭所有打开的套接字
    for (int i = 0; i < 1024; i++) {
        if (table->used[i] && table->sockets[i] >= 0) {
            close(table->sockets[i]);
        }
    }

    free(table);
}

int32_t socket_handle_allocate(SocketHandleTable* table, int socket_fd, bool is_listener) {
    if (!table || socket_fd < 0) {
        return -1;
    }

    // 查找空闲槽位
    for (int i = 0; i < 1024; i++) {
        if (!table->used[i]) {
            table->sockets[i] = socket_fd;
            table->used[i] = true;
            table->is_listener[i] = is_listener;
            // 返回索引 + 1 作为句柄（句柄从 1 开始）
            int32_t handle = i + 1;
            return handle;
        }
    }

    return -1; // 表已满
}

int socket_handle_get(SocketHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return -1;
    }

    int index = handle - 1;
    if (index >= 0 && index < 1024 && table->used[index]) {
        return table->sockets[index];
    }

    return -1;
}

bool socket_handle_is_listener(SocketHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return false;
    }

    int index = handle - 1;
    return (index >= 0 && index < 1024 && table->used[index] && table->is_listener[index]);
}

bool socket_handle_release(SocketHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return false;
    }

    int index = handle - 1;
    if (index >= 0 && index < 1024 && table->used[index]) {
        if (table->sockets[index] >= 0) {
            close(table->sockets[index]);
        }
        table->sockets[index] = -1;
        table->used[index] = false;
        table->is_listener[index] = false;
        return true;
    }

    return false;
}

bool socket_handle_is_valid(SocketHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return false;
    }

    int index = handle - 1;
    return (index >= 0 && index < 1024 && table->used[index] && table->sockets[index] >= 0);
}

// ============================================================================
// 全局 Socket 句柄表管理
// ============================================================================

SocketHandleTable* get_global_socket_handle_table(void) {
    if (!g_initialized) {
        init_global_socket_handle_table();
    }
    return g_socket_handle_table;
}

void init_global_socket_handle_table(void) {
    if (g_initialized) {
        return;
    }

    g_socket_handle_table = socket_handle_table_create();
    g_initialized = true;
}

void cleanup_global_socket_handle_table(void) {
    if (g_socket_handle_table) {
        socket_handle_table_destroy(g_socket_handle_table);
        g_socket_handle_table = NULL;
    }
    g_initialized = false;
}
