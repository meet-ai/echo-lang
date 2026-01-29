/**
 * @file future_repository_memory.c
 * @brief FutureRepository 内存实现
 *
 * 这是一个简单的内存实现，用于开发和测试。
 * 生产环境应该使用数据库或其他持久化实现。
 *
 * 使用场景：
 * - 单元测试
 * - 开发环境
 * - 简单的内存存储需求
 */

#include "future_repository.h"
#include "../aggregate/future.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>

// ============================================================================
// 内存仓储实现
// ============================================================================

/**
 * @brief 内存仓储结构体（实际定义）
 */
struct FutureRepository {
    Future** futures;           // Future指针数组
    size_t capacity;            // 数组容量
    size_t count;              // 当前Future数量
    pthread_mutex_t mutex;     // 保护并发访问
};

/**
 * @brief 创建内存仓储实例
 */
FutureRepository* future_repository_create(void) {
    FutureRepository* repo = (FutureRepository*)calloc(1, sizeof(FutureRepository));
    if (!repo) {
        return NULL;
    }

    // 初始化数组
    repo->capacity = 16;
    repo->count = 0;
    repo->futures = (Future**)calloc(repo->capacity, sizeof(Future*));
    if (!repo->futures) {
        free(repo);
        return NULL;
    }

    // 初始化互斥锁
    if (pthread_mutex_init(&repo->mutex, NULL) != 0) {
        free(repo->futures);
        free(repo);
        return NULL;
    }

    return repo;
}

/**
 * @brief 根据ID查找Future
 */
Future* future_repository_find_by_id(FutureRepository* repo, FutureID id) {
    if (!repo || id == 0) {
        return NULL;
    }

    pthread_mutex_lock(&repo->mutex);

    for (size_t i = 0; i < repo->count; i++) {
        if (repo->futures[i] && future_get_id(repo->futures[i]) == id) {
            pthread_mutex_unlock(&repo->mutex);
            return repo->futures[i];
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return NULL;
}

/**
 * @brief 保存Future
 */
bool future_repository_save(FutureRepository* repo, Future* future) {
    if (!repo || !future) {
        return false;
    }

    pthread_mutex_lock(&repo->mutex);

    FutureID future_id = future_get_id(future);

    // 检查是否已存在
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->futures[i] && future_get_id(repo->futures[i]) == future_id) {
            // 更新现有Future
            repo->futures[i] = future;
            pthread_mutex_unlock(&repo->mutex);
            return true;
        }
    }

    // 添加新Future
    if (repo->count >= repo->capacity) {
        // 扩展数组
        size_t new_capacity = repo->capacity * 2;
        Future** new_futures = (Future**)realloc(repo->futures, new_capacity * sizeof(Future*));
        if (!new_futures) {
            pthread_mutex_unlock(&repo->mutex);
            return false;
        }
        repo->futures = new_futures;
        repo->capacity = new_capacity;
    }

    repo->futures[repo->count++] = future;
    pthread_mutex_unlock(&repo->mutex);
    return true;
}

/**
 * @brief 删除Future
 */
bool future_repository_delete(FutureRepository* repo, FutureID id) {
    if (!repo || id == 0) {
        return false;
    }

    pthread_mutex_lock(&repo->mutex);

    for (size_t i = 0; i < repo->count; i++) {
        if (repo->futures[i] && future_get_id(repo->futures[i]) == id) {
            // 移动最后一个元素到当前位置
            repo->futures[i] = repo->futures[repo->count - 1];
            repo->futures[repo->count - 1] = NULL;
            repo->count--;
            pthread_mutex_unlock(&repo->mutex);
            return true;
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return false; // 未找到
}

/**
 * @brief 获取所有Future
 */
bool future_repository_find_all(FutureRepository* repo, Future*** futures, size_t* count) {
    if (!repo || !futures || !count) {
        return false;
    }

    pthread_mutex_lock(&repo->mutex);

    *count = repo->count;
    if (repo->count == 0) {
        *futures = NULL;
        pthread_mutex_unlock(&repo->mutex);
        return true;
    }

    // 分配数组
    *futures = (Future**)malloc(repo->count * sizeof(Future*));
    if (!*futures) {
        pthread_mutex_unlock(&repo->mutex);
        return false;
    }

    // 复制指针
    memcpy(*futures, repo->futures, repo->count * sizeof(Future*));
    pthread_mutex_unlock(&repo->mutex);
    return true;
}

/**
 * @brief 根据状态查找Future列表
 */
bool future_repository_find_by_state(
    FutureRepository* repo,
    FutureState state,
    Future*** futures,
    size_t* count
) {
    if (!repo || !futures || !count) {
        return false;
    }

    pthread_mutex_lock(&repo->mutex);

    // 先统计数量
    size_t matching_count = 0;
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->futures[i] && future_get_state(repo->futures[i]) == state) {
            matching_count++;
        }
    }

    *count = matching_count;
    if (matching_count == 0) {
        *futures = NULL;
        pthread_mutex_unlock(&repo->mutex);
        return true;
    }

    // 分配数组
    *futures = (Future**)malloc(matching_count * sizeof(Future*));
    if (!*futures) {
        pthread_mutex_unlock(&repo->mutex);
        return false;
    }

    // 复制匹配的Future指针
    size_t idx = 0;
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->futures[i] && future_get_state(repo->futures[i]) == state) {
            (*futures)[idx++] = repo->futures[i];
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return true;
}

/**
 * @brief 销毁仓储实例
 */
void future_repository_destroy(FutureRepository* repo) {
    if (!repo) {
        return;
    }

    // 注意：不销毁Future对象，只释放数组
    if (repo->futures) {
        free(repo->futures);
    }

    pthread_mutex_destroy(&repo->mutex);
    free(repo);
}
