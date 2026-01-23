#include "mutex.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <pthread.h>
#include <time.h>

// 内部实现结构体
typedef struct MutexImpl {
    Mutex base;
    pthread_mutex_t mutex;
    pthread_mutexattr_t attr;
    bool recursive;
    char name[256];
    time_t created_at;
    uint64_t lock_count;
    uint64_t unlock_count;
    uint64_t wait_time_ms;
    pthread_t owner;
} MutexImpl;

// 创建互斥锁
Mutex* mutex_create(bool recursive, const char* name) {
    MutexImpl* impl = (MutexImpl*)malloc(sizeof(MutexImpl));
    if (!impl) {
        fprintf(stderr, "ERROR: Failed to allocate memory for mutex\n");
        return NULL;
    }

    memset(impl, 0, sizeof(MutexImpl));

    // 初始化基础属性
    impl->base.vtable = NULL; // 暂时不使用虚函数表
    impl->recursive = recursive;
    impl->created_at = time(NULL);
    impl->lock_count = 0;
    impl->unlock_count = 0;
    impl->wait_time_ms = 0;

    if (name) {
        strncpy(impl->name, name, sizeof(impl->name) - 1);
        impl->name[sizeof(impl->name) - 1] = '\0';
    } else {
        strcpy(impl->name, "unnamed_mutex");
    }

    // 初始化互斥锁属性
    pthread_mutexattr_init(&impl->attr);

    if (recursive) {
        pthread_mutexattr_settype(&impl->attr, PTHREAD_MUTEX_RECURSIVE);
    } else {
        pthread_mutexattr_settype(&impl->attr, PTHREAD_MUTEX_NORMAL);
    }

    // 初始化互斥锁
    if (pthread_mutex_init(&impl->mutex, &impl->attr) != 0) {
        free(impl);
        fprintf(stderr, "ERROR: Failed to initialize mutex\n");
        return NULL;
    }

    return (Mutex*)impl;
}

// 销毁互斥锁
void mutex_destroy(Mutex* mutex) {
    if (!mutex) return;

    MutexImpl* impl = (MutexImpl*)mutex;

    // 销毁互斥锁
    pthread_mutex_destroy(&impl->mutex);

    // 销毁属性
    pthread_mutexattr_destroy(&impl->attr);

    free(impl);
}

// 上锁
bool mutex_lock(Mutex* mutex) {
    if (!mutex) return false;

    MutexImpl* impl = (MutexImpl*)mutex;

    time_t start_time = time(NULL);

    int result = pthread_mutex_lock(&impl->mutex);

    if (result == 0) {
        impl->lock_count++;
        impl->owner = pthread_self();

        time_t end_time = time(NULL);
        impl->wait_time_ms += (uint64_t)(end_time - start_time) * 1000;

        return true;
    }

    return false;
}

// 尝试上锁
bool mutex_try_lock(Mutex* mutex) {
    if (!mutex) return false;

    MutexImpl* impl = (MutexImpl*)mutex;

    int result = pthread_mutex_trylock(&impl->mutex);

    if (result == 0) {
        impl->lock_count++;
        impl->owner = pthread_self();
        return true;
    }

    return false;
}

// 解锁
bool mutex_unlock(Mutex* mutex) {
    if (!mutex) return false;

    MutexImpl* impl = (MutexImpl*)mutex;

    int result = pthread_mutex_unlock(&impl->mutex);

    if (result == 0) {
        impl->unlock_count++;
        impl->owner = 0; // 清除所有者
        return true;
    }

    return false;
}

// 获取互斥锁统计信息
bool mutex_get_stats(const Mutex* mutex, MutexStats* stats) {
    if (!mutex || !stats) return false;

    const MutexImpl* impl = (const MutexImpl*)mutex;

    stats->lock_count = impl->lock_count;
    stats->unlock_count = impl->unlock_count;
    stats->wait_time_ms = impl->wait_time_ms;
    stats->is_locked = (impl->owner != 0);
    stats->created_at = impl->created_at;

    return true;
}

// 获取互斥锁名称
const char* mutex_get_name(const Mutex* mutex) {
    return mutex ? ((MutexImpl*)mutex)->name : NULL;
}

// 检查是否是递归锁
bool mutex_is_recursive(const Mutex* mutex) {
    return mutex ? ((MutexImpl*)mutex)->recursive : false;
}

// 获取当前所有者
pthread_t mutex_get_owner(const Mutex* mutex) {
    return mutex ? ((MutexImpl*)mutex)->owner : 0;
}

// 检查互斥锁是否被当前线程持有
bool mutex_is_owned_by_current_thread(const Mutex* mutex) {
    if (!mutex) return false;

    const MutexImpl* impl = (const MutexImpl*)mutex;

    return impl->owner == pthread_self();
}
