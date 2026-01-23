#include "file_system.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <dirent.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <errno.h>
#include <pthread.h>
#include <time.h>

// 内部实现结构体
typedef struct FileSystemImpl {
    FileSystem base;

    // 配置
    FileSystemConfig config;

    // 统计信息
    FileSystemStats stats;

    // 同步原语
    pthread_mutex_t mutex;
    bool initialized;
} FileSystemImpl;

// 文件句柄实现
typedef struct FileHandleImpl {
    FileHandle base;
    int fd;                    // 文件描述符
    char path[4096];           // 文件路径
    int flags;                 // 打开标志
    time_t opened_at;          // 打开时间
    uint64_t bytes_read;       // 已读取字节数
    uint64_t bytes_written;    // 已写入字节数
} FileHandleImpl;

// 目录句柄实现
typedef struct DirectoryHandleImpl {
    DirectoryHandle base;
    DIR* dir;                  // 目录流
    char path[4096];           // 目录路径
    time_t opened_at;          // 打开时间
    uint64_t entries_read;     // 已读取条目数
} DirectoryHandleImpl;

// 路径验证和规范化
static bool validate_and_normalize_path(const FileSystemImpl* fs, const char* input_path, char* output_path, size_t output_size) {
    if (!fs || !input_path || !output_path) return false;

    // 检查路径长度
    if (strlen(input_path) >= output_size) {
        return false;
    }

    // 复制路径
    strcpy(output_path, input_path);

    // 如果配置了根路径，确保路径在根目录内
    if (fs->config.root_path[0] != '\0') {
        char full_path[8192];
        if (input_path[0] == '/') {
            // 绝对路径
            snprintf(full_path, sizeof(full_path), "%s%s", fs->config.root_path, input_path);
        } else {
            // 相对路径
            snprintf(full_path, sizeof(full_path), "%s/%s", fs->config.root_path, input_path);
        }

        // 规范化路径（解析..和.）
        char* resolved = realpath(full_path, NULL);
        if (!resolved) return false;

        // 确保在根目录内
        size_t root_len = strlen(fs->config.root_path);
        if (strncmp(resolved, fs->config.root_path, root_len) != 0) {
            free(resolved);
            return false;
        }

        strcpy(output_path, resolved + root_len);
        if (output_path[0] == '\0') strcpy(output_path, "/");
        free(resolved);
    }

    return true;
}

// 创建文件系统
FileSystem* file_system_create(const FileSystemConfig* config) {
    FileSystemImpl* impl = (FileSystemImpl*)malloc(sizeof(FileSystemImpl));
    if (!impl) {
        fprintf(stderr, "ERROR: Failed to allocate memory for file system\n");
        return NULL;
    }

    memset(impl, 0, sizeof(FileSystemImpl));

    // 设置默认配置
    if (config) {
        memcpy(&impl->config, config, sizeof(FileSystemConfig));
    } else {
        strcpy(impl->config.root_path, "");
        impl->config.max_file_size = 100 * 1024 * 1024; // 100MB
        impl->config.max_open_files = 1024;
        impl->config.max_open_dirs = 256;
        impl->config.enable_caching = true;
        impl->config.cache_size = 10 * 1024 * 1024; // 10MB
        impl->config.cache_ttl_seconds = 300; // 5分钟
        impl->config.enable_compression = false;
        impl->config.enable_encryption = false;
        impl->config.enable_logging = true;
        strcpy(impl->config.log_file, "/tmp/file_system.log");
    }

    // 初始化统计信息
    memset(&impl->stats, 0, sizeof(FileSystemStats));
    impl->stats.created_at = time(NULL);

    // 初始化同步原语
    pthread_mutex_init(&impl->mutex, NULL);

    impl->initialized = true;

    return (FileSystem*)impl;
}

// 销毁文件系统
void file_system_destroy(FileSystem* fs) {
    if (!fs) return;

    FileSystemImpl* impl = (FileSystemImpl*)fs;

    if (impl->initialized) {
        pthread_mutex_destroy(&impl->mutex);
    }

    free(impl);
}

// 打开文件
FileHandle* file_system_open_file(FileSystem* fs, const char* path, FileOpenMode mode, FileOpenFlags flags) {
    if (!fs || !path) return NULL;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return NULL;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return NULL;
    }

    // 转换为系统调用标志
    int sys_flags = O_RDONLY;
    switch (mode) {
        case FILE_MODE_READ:
            sys_flags = O_RDONLY;
            break;
        case FILE_MODE_WRITE:
            sys_flags = O_WRONLY | O_CREAT | O_TRUNC;
            break;
        case FILE_MODE_APPEND:
            sys_flags = O_WRONLY | O_CREAT | O_APPEND;
            break;
        case FILE_MODE_READ_WRITE:
            sys_flags = O_RDWR | O_CREAT;
            break;
    }

    if (flags & FILE_FLAG_CREATE) sys_flags |= O_CREAT;
    if (flags & FILE_FLAG_EXCLUSIVE) sys_flags |= O_EXCL;
    if (flags & FILE_FLAG_TRUNCATE) sys_flags |= O_TRUNC;

    // 检查文件大小限制
    if (mode != FILE_MODE_READ) {
        struct stat st;
        if (stat(normalized_path, &st) == 0) {
            if (st.st_size > impl->config.max_file_size) {
                return NULL; // 文件过大
            }
        }
    }

    int fd = open(normalized_path, sys_flags, 0644);
    if (fd < 0) {
        return NULL;
    }

    FileHandleImpl* handle = (FileHandleImpl*)malloc(sizeof(FileHandleImpl));
    if (!handle) {
        close(fd);
        return NULL;
    }

    memset(handle, 0, sizeof(FileHandleImpl));
    handle->fd = fd;
    strcpy(handle->path, normalized_path);
    handle->flags = sys_flags;
    handle->opened_at = time(NULL);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_files_opened++;
    impl->stats.currently_open_files++;
    pthread_mutex_unlock(&impl->mutex);

    return (FileHandle*)handle;
}

// 关闭文件
bool file_system_close_file(FileSystem* fs, FileHandle* handle) {
    if (!fs || !handle) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    FileHandleImpl* handle_impl = (FileHandleImpl*)handle;

    int result = close(handle_impl->fd);

    if (result == 0) {
        // 更新统计信息
        pthread_mutex_lock(&impl->mutex);
        impl->stats.currently_open_files--;
        impl->stats.total_bytes_read += handle_impl->bytes_read;
        impl->stats.total_bytes_written += handle_impl->bytes_written;
        pthread_mutex_unlock(&impl->mutex);

        free(handle_impl);
        return true;
    }

    free(handle_impl);
    return false;
}

// 读取文件
size_t file_handle_read(FileHandle* handle, void* buffer, size_t size) {
    if (!handle || !buffer || size == 0) return 0;

    FileHandleImpl* impl = (FileHandleImpl*)handle;

    ssize_t result = read(impl->fd, buffer, size);
    if (result > 0) {
        impl->bytes_read += result;
    }

    return result > 0 ? (size_t)result : 0;
}

// 写入文件
size_t file_handle_write(FileHandle* handle, const void* buffer, size_t size) {
    if (!handle || !buffer || size == 0) return 0;

    FileHandleImpl* impl = (FileHandleImpl*)handle;

    ssize_t result = write(impl->fd, buffer, size);
    if (result > 0) {
        impl->bytes_written += result;
    }

    return result > 0 ? (size_t)result : 0;
}

// 获取文件位置
long long file_handle_tell(FileHandle* handle) {
    if (!handle) return -1;

    FileHandleImpl* impl = (FileHandleImpl*)handle;
    return lseek(impl->fd, 0, SEEK_CUR);
}

// 设置文件位置
bool file_handle_seek(FileHandle* handle, long long offset, FileSeekOrigin origin) {
    if (!handle) return false;

    FileHandleImpl* impl = (FileHandleImpl*)handle;

    int whence;
    switch (origin) {
        case FILE_SEEK_SET: whence = SEEK_SET; break;
        case FILE_SEEK_CUR: whence = SEEK_CUR; break;
        case FILE_SEEK_END: whence = SEEK_END; break;
        default: return false;
    }

    return lseek(impl->fd, offset, whence) != -1;
}

// 截断文件
bool file_handle_truncate(FileHandle* handle, long long size) {
    if (!handle) return false;

    FileHandleImpl* impl = (FileHandleImpl*)handle;
    return ftruncate(impl->fd, size) == 0;
}

// 刷新文件
bool file_handle_flush(FileHandle* handle) {
    if (!handle) return false;

    FileHandleImpl* impl = (FileHandleImpl*)handle;

    // 对于普通文件，fsync可以确保数据写入磁盘
    return fsync(impl->fd) == 0;
}

// 获取文件大小
long long file_handle_size(FileHandle* handle) {
    if (!handle) return -1;

    FileHandleImpl* impl = (FileHandleImpl*)handle;

    struct stat st;
    if (fstat(impl->fd, &st) != 0) {
        return -1;
    }

    return st.st_size;
}

// 检查文件是否可读
bool file_handle_can_read(FileHandle* handle) {
    if (!handle) return false;

    FileHandleImpl* impl = (FileHandleImpl*)handle;
    return (impl->flags & O_RDONLY) || (impl->flags & O_RDWR);
}

// 检查文件是否可写
bool file_handle_can_write(FileHandle* handle) {
    if (!handle) return false;

    FileHandleImpl* impl = (FileHandleImpl*)handle;
    return (impl->flags & O_WRONLY) || (impl->flags & O_RDWR) || (impl->flags & O_APPEND);
}

// 打开目录
DirectoryHandle* file_system_open_directory(FileSystem* fs, const char* path) {
    if (!fs || !path) return NULL;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return NULL;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return NULL;
    }

    DIR* dir = opendir(normalized_path);
    if (!dir) {
        return NULL;
    }

    DirectoryHandleImpl* handle = (DirectoryHandleImpl*)malloc(sizeof(DirectoryHandleImpl));
    if (!handle) {
        closedir(dir);
        return NULL;
    }

    memset(handle, 0, sizeof(DirectoryHandleImpl));
    handle->dir = dir;
    strcpy(handle->path, normalized_path);
    handle->opened_at = time(NULL);

    // 更新统计信息
    pthread_mutex_lock(&impl->mutex);
    impl->stats.total_directories_opened++;
    impl->stats.currently_open_directories++;
    pthread_mutex_unlock(&impl->mutex);

    return (DirectoryHandle*)handle;
}

// 关闭目录
bool file_system_close_directory(FileSystem* fs, DirectoryHandle* handle) {
    if (!fs || !handle) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    DirectoryHandleImpl* handle_impl = (DirectoryHandleImpl*)handle;

    int result = closedir(handle_impl->dir);

    if (result == 0) {
        // 更新统计信息
        pthread_mutex_lock(&impl->mutex);
        impl->stats.currently_open_directories--;
        impl->stats.total_directory_entries_read += handle_impl->entries_read;
        pthread_mutex_unlock(&impl->mutex);

        free(handle_impl);
        return true;
    }

    free(handle_impl);
    return false;
}

// 读取目录条目
bool directory_handle_read_entry(DirectoryHandle* handle, DirectoryEntry* entry) {
    if (!handle || !entry) return false;

    DirectoryHandleImpl* impl = (DirectoryHandleImpl*)handle;

    struct dirent* dirent = readdir(impl->dir);
    if (!dirent) return false;

    memset(entry, 0, sizeof(DirectoryEntry));
    strcpy(entry->name, dirent->d_name);
    entry->type = (dirent->d_type == DT_DIR) ? DIRECTORY_ENTRY_DIRECTORY : DIRECTORY_ENTRY_FILE;

    impl->entries_read++;

    return true;
}

// 创建目录
bool file_system_create_directory(FileSystem* fs, const char* path, bool recursive) {
    if (!fs || !path) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return false;
    }

    // 如果是递归创建，需要逐级创建
    if (recursive) {
        char temp_path[4096];
        char* p = NULL;
        size_t len;

        snprintf(temp_path, sizeof(temp_path), "%s", normalized_path);
        len = strlen(temp_path);

        if (temp_path[len - 1] == '/') {
            temp_path[len - 1] = '\0';
        }

        for (p = temp_path + 1; *p; p++) {
            if (*p == '/') {
                *p = '\0';
                mkdir(temp_path, 0755);
                *p = '/';
            }
        }
    }

    int result = mkdir(normalized_path, 0755);

    if (result == 0) {
        pthread_mutex_lock(&impl->mutex);
        impl->stats.total_directories_created++;
        pthread_mutex_unlock(&impl->mutex);
    }

    return result == 0;
}

// 删除文件或目录
bool file_system_remove(FileSystem* fs, const char* path, bool recursive) {
    if (!fs || !path) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return false;
    }

    struct stat st;
    if (stat(normalized_path, &st) != 0) {
        return false;
    }

    bool success = false;

    if (S_ISDIR(st.st_mode)) {
        // 删除目录
        if (recursive) {
            // 递归删除（简化实现，实际应该遍历删除子项）
            success = (rmdir(normalized_path) == 0);
        } else {
            success = (rmdir(normalized_path) == 0);
        }

        if (success) {
            pthread_mutex_lock(&impl->mutex);
            impl->stats.total_directories_removed++;
            pthread_mutex_unlock(&impl->mutex);
        }
    } else {
        // 删除文件
        success = (unlink(normalized_path) == 0);

        if (success) {
            pthread_mutex_lock(&impl->mutex);
            impl->stats.total_files_removed++;
            pthread_mutex_unlock(&impl->mutex);
        }
    }

    return success;
}

// 移动/重命名文件或目录
bool file_system_move(FileSystem* fs, const char* old_path, const char* new_path) {
    if (!fs || !old_path || !new_path) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char old_normalized[4096], new_normalized[4096];
    if (!validate_and_normalize_path(impl, old_path, old_normalized, sizeof(old_normalized)) ||
        !validate_and_normalize_path(impl, new_path, new_normalized, sizeof(new_normalized))) {
        return false;
    }

    return rename(old_normalized, new_normalized) == 0;
}

// 复制文件
bool file_system_copy(FileSystem* fs, const char* src_path, const char* dst_path) {
    if (!fs || !src_path || !dst_path) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char src_normalized[4096], dst_normalized[4096];
    if (!validate_and_normalize_path(impl, src_path, src_normalized, sizeof(src_normalized)) ||
        !validate_and_normalize_path(impl, dst_path, dst_normalized, sizeof(dst_normalized))) {
        return false;
    }

    // 打开源文件
    int src_fd = open(src_normalized, O_RDONLY);
    if (src_fd < 0) return false;

    // 打开目标文件
    int dst_fd = open(dst_normalized, O_WRONLY | O_CREAT | O_TRUNC, 0644);
    if (dst_fd < 0) {
        close(src_fd);
        return false;
    }

    // 复制数据
    char buffer[8192];
    ssize_t bytes_read;
    bool success = true;

    while ((bytes_read = read(src_fd, buffer, sizeof(buffer))) > 0) {
        if (write(dst_fd, buffer, bytes_read) != bytes_read) {
            success = false;
            break;
        }
    }

    close(src_fd);
    close(dst_fd);

    if (success) {
        pthread_mutex_lock(&impl->mutex);
        impl->stats.total_files_copied++;
        pthread_mutex_unlock(&impl->mutex);
    }

    return success && (bytes_read == 0);
}

// 获取文件/目录信息
bool file_system_get_info(FileSystem* fs, const char* path, FileStats* stats) {
    if (!fs || !path || !stats) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return false;
    }

    struct stat st;
    if (stat(normalized_path, &st) != 0) {
        return false;
    }

    memset(stats, 0, sizeof(FileStats));
    stats->size = st.st_size;
    stats->created_at = st.st_ctime;
    stats->modified_at = st.st_mtime;
    stats->accessed_at = st.st_atime;
    stats->permissions = st.st_mode;
    stats->owner_uid = st.st_uid;
    stats->owner_gid = st.st_gid;
    stats->is_directory = S_ISDIR(st.st_mode);
    stats->is_symlink = S_ISLNK(st.st_mode);
    stats->block_size = st.st_blksize;
    stats->blocks = st.st_blocks;

    return true;
}

// 检查文件/目录是否存在
bool file_system_exists(FileSystem* fs, const char* path) {
    if (!fs || !path) return false;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return false;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return false;
    }

    return access(normalized_path, F_OK) == 0;
}

// 获取文件系统统计信息
bool file_system_get_stats(const FileSystem* fs, FileSystemStats* stats) {
    if (!fs || !stats) return false;

    const FileSystemImpl* impl = (FileSystemImpl*)fs;

    pthread_mutex_lock(&impl->mutex);
    memcpy(stats, &impl->stats, sizeof(FileSystemStats));
    pthread_mutex_unlock(&impl->mutex);

    return true;
}

// 列出目录内容
DirectoryEntry* file_system_list_directory(FileSystem* fs, const char* path, size_t* count) {
    if (!fs || !path || !count) return NULL;

    FileSystemImpl* impl = (FileSystemImpl*)fs;
    if (!impl->initialized) return NULL;

    char normalized_path[4096];
    if (!validate_and_normalize_path(impl, path, normalized_path, sizeof(normalized_path))) {
        return NULL;
    }

    DIR* dir = opendir(normalized_path);
    if (!dir) return NULL;

    // 先计算条目数量
    size_t entry_count = 0;
    struct dirent* entry;
    while ((entry = readdir(dir)) != NULL) {
        if (strcmp(entry->d_name, ".") != 0 && strcmp(entry->d_name, "..") != 0) {
            entry_count++;
        }
    }

    rewinddir(dir);

    DirectoryEntry* entries = (DirectoryEntry*)malloc(sizeof(DirectoryEntry) * entry_count);
    if (!entries) {
        closedir(dir);
        return NULL;
    }

    size_t index = 0;
    while ((entry = readdir(dir)) != NULL && index < entry_count) {
        if (strcmp(entry->d_name, ".") != 0 && strcmp(entry->d_name, "..") != 0) {
            strcpy(entries[index].name, entry->d_name);
            entries[index].type = (entry->d_type == DT_DIR) ?
                                 DIRECTORY_ENTRY_DIRECTORY : DIRECTORY_ENTRY_FILE;
            index++;
        }
    }

    closedir(dir);
    *count = entry_count;

    return entries;
}

// 释放目录条目列表
void file_system_free_directory_entries(DirectoryEntry* entries) {
    free(entries);
}

// 创建配置文件
FileSystemConfig* file_system_config_create(void) {
    FileSystemConfig* config = (FileSystemConfig*)malloc(sizeof(FileSystemConfig));
    if (config) {
        memset(config, 0, sizeof(FileSystemConfig));
    }
    return config;
}

// 销毁配置文件
void file_system_config_destroy(FileSystemConfig* config) {
    if (config) {
        free(config->encryption_key);
        free(config);
    }
}
