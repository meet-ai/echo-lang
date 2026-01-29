#ifndef FILE_HANDLE_MANAGER_H
#define FILE_HANDLE_MANAGER_H

#include <stdint.h>
#include <stdbool.h>
#include <stdio.h>

// ============================================================================
// 文件句柄管理
// ============================================================================
// FileHandle 在编译器层面映射为 i32（文件描述符）
// 运行时系统需要管理文件句柄的生命周期

// 文件句柄类型（内部使用）
typedef int32_t FileHandle;

// 文件句柄表（用于管理打开的文件）
typedef struct {
    FILE* files[1024];      // 文件指针数组（最多支持 1024 个打开的文件）
    bool used[1024];         // 使用标记
    int32_t next_handle;     // 下一个可用的句柄 ID
} FileHandleTable;

// ============================================================================
// 文件句柄管理函数
// ============================================================================

/**
 * @brief 初始化文件句柄表
 * @return 文件句柄表指针，失败返回 NULL
 */
FileHandleTable* file_handle_table_create(void);

/**
 * @brief 销毁文件句柄表
 * @param table 文件句柄表指针
 */
void file_handle_table_destroy(FileHandleTable* table);

/**
 * @brief 分配文件句柄
 * @param table 文件句柄表
 * @param file 文件指针
 * @return 文件句柄（i32），失败返回 -1
 */
int32_t file_handle_allocate(FileHandleTable* table, FILE* file);

/**
 * @brief 获取文件指针
 * @param table 文件句柄表
 * @param handle 文件句柄
 * @return 文件指针，失败返回 NULL
 */
FILE* file_handle_get(FileHandleTable* table, int32_t handle);

/**
 * @brief 释放文件句柄
 * @param table 文件句柄表
 * @param handle 文件句柄
 * @return true 表示成功，false 表示失败
 */
bool file_handle_release(FileHandleTable* table, int32_t handle);

/**
 * @brief 检查文件句柄是否有效
 * @param table 文件句柄表
 * @param handle 文件句柄
 * @return true 表示有效，false 表示无效
 */
bool file_handle_is_valid(FileHandleTable* table, int32_t handle);

// ============================================================================
// 全局文件句柄表（单例模式）
// ============================================================================

/**
 * @brief 获取全局文件句柄表
 * @return 文件句柄表指针
 */
FileHandleTable* get_global_file_handle_table(void);

/**
 * @brief 初始化全局文件句柄表
 */
void init_global_file_handle_table(void);

/**
 * @brief 清理全局文件句柄表
 */
void cleanup_global_file_handle_table(void);

#endif // FILE_HANDLE_MANAGER_H
