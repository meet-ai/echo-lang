#ifndef FILE_SYSTEM_H
#define FILE_SYSTEM_H

#include <stdint.h>
#include <stdbool.h>
#include <time.h>
#include <sys/stat.h>

// 前向声明
struct FileSystem;
struct FileHandle;
struct DirectoryHandle;
struct FileSystemConfig;
struct FileStats;
struct DirectoryEntry;

// 文件系统配置
typedef struct FileSystemConfig {
    char root_path[4096];        // 根路径
    size_t max_file_size;        // 最大文件大小
    uint32_t max_open_files;     // 最大打开文件数
    uint32_t max_open_dirs;      // 最大打开目录数
    bool enable_caching;         // 是否启用缓存
    size_t cache_size;           // 缓存大小
    uint32_t cache_ttl_seconds;  // 缓存TTL
    bool enable_compression;     // 是否启用压缩
    bool enable_encryption;      // 是否启用加密
    char* encryption_key;        // 加密密钥
    bool enable_logging;         // 是否启用日志
    char log_file[512];          // 日志文件路径
} FileSystemConfig;

// 文件统计信息
typedef struct FileStats {
    uint64_t size;               // 文件大小
    time_t created_at;           // 创建时间
    time_t modified_at;          // 修改时间
    time_t accessed_at;          // 访问时间
    mode_t permissions;          // 权限
    uid_t owner_uid;             // 所有者UID
    gid_t owner_gid;             // 所有者GID
    bool is_directory;           // 是否为目录
    bool is_symlink;             // 是否为符号链接
    uint64_t block_size;         // 块大小
    uint64_t blocks;             // 块数量
    char mime_type[128];         // MIME类型
    uint32_t reference_count;    // 引用计数
} FileStats;

// 目录条目
typedef struct DirectoryEntry {
    char name[256];              // 条目名称
    char full_path[4096];        // 完整路径
    struct FileStats stats;      // 统计信息
    bool is_hidden;              // 是否隐藏
} DirectoryEntry;

// 文件句柄
typedef struct FileHandle {
    void* internal_handle;       // 内部文件句柄
    char file_path[4096];        // 文件路径
    char open_mode[16];          // 打开模式
    time_t opened_at;            // 打开时间
    uint64_t position;           // 当前位置
    bool is_open;                // 是否打开
    struct FileStats stats;      // 文件统计
} FileHandle;

// 目录句柄
typedef struct DirectoryHandle {
    void* internal_handle;       // 内部目录句柄
    char dir_path[4096];         // 目录路径
    time_t opened_at;            // 打开时间
    bool is_open;                // 是否打开
    struct FileStats stats;      // 目录统计
} DirectoryHandle;

// 文件系统接口
typedef struct {
    // 文件操作
    bool (*open_file)(struct FileSystem* fs, const char* path, const char* mode, struct FileHandle** handle);
    bool (*close_file)(struct FileSystem* fs, struct FileHandle* handle);
    bool (*read_file)(struct FileSystem* fs, struct FileHandle* handle, void* buffer, size_t size, size_t* bytes_read);
    bool (*write_file)(struct FileSystem* fs, struct FileHandle* handle, const void* buffer, size_t size, size_t* bytes_written);
    bool (*seek_file)(struct FileSystem* fs, struct FileHandle* handle, int64_t offset, int whence);
    bool (*tell_file)(struct FileSystem* fs, struct FileHandle* handle, uint64_t* position);
    bool (*flush_file)(struct FileSystem* fs, struct FileHandle* handle);
    bool (*truncate_file)(struct FileSystem* fs, struct FileHandle* handle, uint64_t size);

    // 目录操作
    bool (*open_directory)(struct FileSystem* fs, const char* path, struct DirectoryHandle** handle);
    bool (*close_directory)(struct FileSystem* fs, struct DirectoryHandle* handle);
    bool (*read_directory)(struct FileSystem* fs, struct DirectoryHandle* handle, struct DirectoryEntry* entry);
    bool (*rewind_directory)(struct FileSystem* fs, struct DirectoryHandle* handle);

    // 文件系统操作
    bool (*create_file)(struct FileSystem* fs, const char* path, mode_t mode);
    bool (*create_directory)(struct FileSystem* fs, const char* path, mode_t mode);
    bool (*remove_file)(struct FileSystem* fs, const char* path);
    bool (*remove_directory)(struct FileSystem* fs, const char* path);
    bool (*rename_file)(struct FileSystem* fs, const char* old_path, const char* new_path);
    bool (*copy_file)(struct FileSystem* fs, const char* src_path, const char* dst_path);
    bool (*move_file)(struct FileSystem* fs, const char* src_path, const char* dst_path);

    // 文件信息
    bool (*get_file_stats)(struct FileSystem* fs, const char* path, struct FileStats* stats);
    bool (*set_file_permissions)(struct FileSystem* fs, const char* path, mode_t mode);
    bool (*set_file_times)(struct FileSystem* fs, const char* path, time_t access_time, time_t modify_time);
    bool (*get_file_contents)(struct FileSystem* fs, const char* path, void** buffer, size_t* size);
    bool (*set_file_contents)(struct FileSystem* fs, const char* path, const void* buffer, size_t size);

    // 目录操作
    bool (*list_directory)(struct FileSystem* fs, const char* path, struct DirectoryEntry** entries, size_t* count);
    bool (*get_directory_stats)(struct FileSystem* fs, const char* path, struct FileStats* stats);

    // 路径操作
    bool (*path_exists)(struct FileSystem* fs, const char* path);
    bool (*is_file)(struct FileSystem* fs, const char* path);
    bool (*is_directory)(struct FileSystem* fs, const char* path);
    bool (*is_symlink)(struct FileSystem* fs, const char* path);
    bool (*get_absolute_path)(struct FileSystem* fs, const char* path, char* absolute_path, size_t size);
    bool (*get_relative_path)(struct FileSystem* fs, const char* base_path, const char* target_path, char* relative_path, size_t size);

    // 高级操作
    bool (*find_files)(struct FileSystem* fs, const char* pattern, char*** paths, size_t* count);
    bool (*calculate_directory_size)(struct FileSystem* fs, const char* path, uint64_t* size);
    bool (*sync_directory)(struct FileSystem* fs, const char* src_path, const char* dst_path);

    // 监控和缓存
    bool (*watch_file)(struct FileSystem* fs, const char* path,
                      void (*callback)(const char* path, int event_type, void* user_data),
                      void* user_data, void** watch_handle);
    bool (*unwatch_file)(struct FileSystem* fs, void* watch_handle);
    bool (*invalidate_cache)(struct FileSystem* fs, const char* path);
    bool (*clear_cache)(struct FileSystem* fs);

    // 统计和监控
    bool (*get_fs_stats)(struct FileSystem* fs, void* stats_buffer, size_t buffer_size);
    bool (*get_open_files)(struct FileSystem* fs, struct FileHandle** handles, size_t* count);
} FileSystemInterface;

// 文件系统实现
typedef struct FileSystem {
    FileSystemInterface* vtable;  // 虚函数表

    // 配置
    struct FileSystemConfig config;

    // 状态
    bool initialized;
    time_t mounted_at;
    char fs_name[128];

    // 内部数据结构
    void* file_table;            // 文件表
    void* directory_table;       // 目录表
    void* cache;                 // 缓存
    void* watch_table;           // 监控表

    // 同步原语
    void* mutex;

    // 统计
    uint64_t total_operations;
    uint64_t cache_hits;
    uint64_t cache_misses;
} FileSystem;

// 文件系统工厂函数
FileSystem* file_system_create(const struct FileSystemConfig* config);
void file_system_destroy(FileSystem* fs);

// 便捷构造函数
struct FileSystemConfig* file_system_config_create(void);
void file_system_config_destroy(struct FileSystemConfig* config);
struct FileHandle* file_handle_create(void);
void file_handle_destroy(struct FileHandle* handle);
struct DirectoryHandle* directory_handle_create(void);
void directory_handle_destroy(struct DirectoryHandle* handle);

// 预定义配置
struct FileSystemConfig* file_system_config_create_default(void);
struct FileSystemConfig* file_system_config_create_high_performance(void);
struct FileSystemConfig* file_system_config_create_secure(void);

// 便捷函数（直接调用虚函数表）
static inline bool file_system_open_file(FileSystem* fs, const char* path, const char* mode, struct FileHandle** handle) {
    return fs->vtable->open_file(fs, path, mode, handle);
}

static inline bool file_system_close_file(FileSystem* fs, struct FileHandle* handle) {
    return fs->vtable->close_file(fs, handle);
}

static inline bool file_system_read_file(FileSystem* fs, struct FileHandle* handle, void* buffer, size_t size, size_t* bytes_read) {
    return fs->vtable->read_file(fs, handle, buffer, size, bytes_read);
}

static inline bool file_system_write_file(FileSystem* fs, struct FileHandle* handle, const void* buffer, size_t size, size_t* bytes_written) {
    return fs->vtable->write_file(fs, handle, buffer, size, bytes_written);
}

static inline bool file_system_get_file_stats(FileSystem* fs, const char* path, struct FileStats* stats) {
    return fs->vtable->get_file_stats(fs, path, stats);
}

// 全局文件系统实例（可选）
extern FileSystem* g_global_file_system;

// 全局便捷函数
static inline bool file_system_read_file_global(const char* path, void* buffer, size_t size, size_t* bytes_read) {
    if (g_global_file_system) {
        struct FileHandle* handle;
        if (!file_system_open_file(g_global_file_system, path, "r", &handle)) {
            return false;
        }
        bool result = file_system_read_file(g_global_file_system, handle, buffer, size, bytes_read);
        file_system_close_file(g_global_file_system, handle);
        return result;
    }
    return false;
}

#endif // FILE_SYSTEM_H
