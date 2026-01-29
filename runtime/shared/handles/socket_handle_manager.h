#ifndef SOCKET_HANDLE_MANAGER_H
#define SOCKET_HANDLE_MANAGER_H

#include <stdint.h>
#include <stdbool.h>
#include <sys/socket.h>

// ============================================================================
// Socket 句柄管理
// ============================================================================
// SocketHandle 在编译器层面映射为 i32（套接字描述符）
// 运行时系统需要管理 Socket 句柄的生命周期

// Socket 句柄类型（内部使用）
typedef int32_t SocketHandle;

// Socket 句柄表（用于管理打开的套接字）
typedef struct {
    int sockets[1024];       // 套接字描述符数组（最多支持 1024 个打开的套接字）
    bool used[1024];         // 使用标记
    bool is_listener[1024];  // 是否为监听套接字
} SocketHandleTable;

// ============================================================================
// Socket 句柄管理函数
// ============================================================================

/**
 * @brief 初始化 Socket 句柄表
 * @return Socket 句柄表指针，失败返回 NULL
 */
SocketHandleTable* socket_handle_table_create(void);

/**
 * @brief 销毁 Socket 句柄表
 * @param table Socket 句柄表指针
 */
void socket_handle_table_destroy(SocketHandleTable* table);

/**
 * @brief 分配 Socket 句柄
 * @param table Socket 句柄表
 * @param socket_fd 套接字描述符
 * @param is_listener 是否为监听套接字
 * @return Socket 句柄（i32），失败返回 -1
 */
int32_t socket_handle_allocate(SocketHandleTable* table, int socket_fd, bool is_listener);

/**
 * @brief 获取套接字描述符
 * @param table Socket 句柄表
 * @param handle Socket 句柄
 * @return 套接字描述符，失败返回 -1
 */
int socket_handle_get(SocketHandleTable* table, int32_t handle);

/**
 * @brief 检查是否为监听套接字
 * @param table Socket 句柄表
 * @param handle Socket 句柄
 * @return true 表示是监听套接字，false 表示不是
 */
bool socket_handle_is_listener(SocketHandleTable* table, int32_t handle);

/**
 * @brief 释放 Socket 句柄
 * @param table Socket 句柄表
 * @param handle Socket 句柄
 * @return true 表示成功，false 表示失败
 */
bool socket_handle_release(SocketHandleTable* table, int32_t handle);

/**
 * @brief 检查 Socket 句柄是否有效
 * @param table Socket 句柄表
 * @param handle Socket 句柄
 * @return true 表示有效，false 表示无效
 */
bool socket_handle_is_valid(SocketHandleTable* table, int32_t handle);

// ============================================================================
// 全局 Socket 句柄表（单例模式）
// ============================================================================

/**
 * @brief 获取全局 Socket 句柄表
 * @return Socket 句柄表指针
 */
SocketHandleTable* get_global_socket_handle_table(void);

/**
 * @brief 初始化全局 Socket 句柄表
 */
void init_global_socket_handle_table(void);

/**
 * @brief 清理全局 Socket 句柄表
 */
void cleanup_global_socket_handle_table(void);

#endif // SOCKET_HANDLE_MANAGER_H
