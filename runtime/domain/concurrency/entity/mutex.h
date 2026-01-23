#ifndef MUTEX_H
#define MUTEX_H

#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>

// 前向声明
struct Mutex;

// 互斥锁实体 - 实现临界区访问控制
typedef struct Mutex {
    uint64_t id;                    // 互斥锁唯一标识
    pthread_mutex_t mutex;          // POSIX互斥锁
    bool is_recursive;              // 是否为递归锁
    uint64_t owner_thread_id;       // 拥有者线程ID（用于递归锁）
    uint32_t lock_count;            // 递归锁的锁定次数

    // 统计信息
    uint64_t lock_count_total;      // 总锁定次数
    uint64_t unlock_count_total;    // 总解锁次数
    uint64_t contention_count;      // 争用次数
    time_t created_at;              // 创建时间
} Mutex;

// 互斥锁选项
typedef struct {
    bool recursive;                 // 是否递归锁
} MutexOptions;

// 默认选项
extern const MutexOptions MUTEX_DEFAULT_OPTIONS;

// 互斥锁方法
Mutex* mutex_create(const MutexOptions* options);
void mutex_destroy(Mutex* mutex);

// 锁定操作
bool mutex_lock(Mutex* mutex);
bool mutex_try_lock(Mutex* mutex);
bool mutex_timed_lock(Mutex* mutex, uint32_t timeout_ms);

// 解锁操作
bool mutex_unlock(Mutex* mutex);

// 状态查询
bool mutex_is_locked(const Mutex* mutex);
uint64_t mutex_get_owner(const Mutex* mutex);
uint64_t mutex_get_lock_count_total(const Mutex* mutex);
uint64_t mutex_get_contention_count(const Mutex* mutex);

#endif // MUTEX_H
