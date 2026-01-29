#include "file_handle_manager.h"
#include <stdlib.h>
#include <string.h>

// 全局文件句柄表（单例）
static FileHandleTable* g_file_handle_table = NULL;
static bool g_initialized = false;

FileHandleTable* file_handle_table_create(void) {
    FileHandleTable* table = (FileHandleTable*)calloc(1, sizeof(FileHandleTable));
    if (!table) {
        return NULL;
    }

    // 初始化句柄表
    table->next_handle = 1; // 从 1 开始（0 表示无效句柄）
    for (int i = 0; i < 1024; i++) {
        table->files[i] = NULL;
        table->used[i] = false;
    }

    return table;
}

void file_handle_table_destroy(FileHandleTable* table) {
    if (!table) {
        return;
    }

    // 关闭所有打开的文件
    for (int i = 0; i < 1024; i++) {
        if (table->used[i] && table->files[i]) {
            fclose(table->files[i]);
        }
    }

    free(table);
}

int32_t file_handle_allocate(FileHandleTable* table, FILE* file) {
    if (!table || !file) {
        return -1;
    }

    // 查找空闲槽位
    for (int i = 0; i < 1024; i++) {
        if (!table->used[i]) {
            table->files[i] = file;
            table->used[i] = true;
            // 返回索引 + 1 作为句柄（句柄从 1 开始）
            int32_t handle = i + 1;
            return handle;
        }
    }

    return -1; // 表已满
}

FILE* file_handle_get(FileHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return NULL;
    }

    // 直接使用 handle 作为索引（handle 从 1 开始，数组从 0 开始）
    int index = handle - 1;
    if (index >= 0 && index < 1024 && table->used[index]) {
        return table->files[index];
    }

    return NULL;
}

bool file_handle_release(FileHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return false;
    }

    int index = handle - 1;
    if (index >= 0 && index < 1024 && table->used[index]) {
        if (table->files[index]) {
            fclose(table->files[index]);
        }
        table->files[index] = NULL;
        table->used[index] = false;
        return true;
    }

    return false;
}

bool file_handle_is_valid(FileHandleTable* table, int32_t handle) {
    if (!table || handle < 1 || handle > 1024) {
        return false;
    }

    int index = handle - 1;
    return (index >= 0 && index < 1024 && table->used[index] && table->files[index] != NULL);
}

// ============================================================================
// 全局文件句柄表管理
// ============================================================================

FileHandleTable* get_global_file_handle_table(void) {
    if (!g_initialized) {
        init_global_file_handle_table();
    }
    return g_file_handle_table;
}

void init_global_file_handle_table(void) {
    if (g_initialized) {
        return;
    }

    g_file_handle_table = file_handle_table_create();
    g_initialized = true;
}

void cleanup_global_file_handle_table(void) {
    if (g_file_handle_table) {
        file_handle_table_destroy(g_file_handle_table);
        g_file_handle_table = NULL;
    }
    g_initialized = false;
}
