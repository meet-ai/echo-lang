/**
 * @file task_repository_memory.c
 * @brief TaskRepository 内存实现
 *
 * 这是一个简单的内存实现，用于开发和测试。
 * 生产环境应该使用数据库或其他持久化实现。
 *
 * 使用场景：
 * - 单元测试
 * - 开发环境
 * - 简单的内存存储需求
 */

#include "task_repository.h"
#include "../aggregate/task.h"
#include <stdlib.h>
#include <string.h>
#include <pthread.h>

// ============================================================================
// 内存仓储实现
// ============================================================================

/**
 * @brief 内存仓储结构体
 */
struct TaskRepository {
    Task** tasks;              // Task指针数组
    size_t capacity;           // 数组容量
    size_t count;              // 当前Task数量
    pthread_mutex_t mutex;     // 保护并发访问
};

/**
 * @brief 创建内存仓储实例
 */
TaskRepository* task_repository_create(void) {
    TaskRepository* repo = (TaskRepository*)calloc(1, sizeof(TaskRepository));
    if (!repo) {
        return NULL;
    }

    // 初始化数组
    repo->capacity = 16;
    repo->count = 0;
    repo->tasks = (Task**)calloc(repo->capacity, sizeof(Task*));
    if (!repo->tasks) {
        free(repo);
        return NULL;
    }

    // 初始化互斥锁
    if (pthread_mutex_init(&repo->mutex, NULL) != 0) {
        free(repo->tasks);
        free(repo);
        return NULL;
    }

    return repo;
}

/**
 * @brief 根据ID查找Task
 */
Task* task_repository_find_by_id(TaskRepository* repo, TaskID task_id) {
    if (!repo || task_id == 0) {
        return NULL;
    }

    pthread_mutex_lock(&repo->mutex);

    for (size_t i = 0; i < repo->count; i++) {
        if (repo->tasks[i] && task_get_id(repo->tasks[i]) == task_id) {
            pthread_mutex_unlock(&repo->mutex);
            return repo->tasks[i];
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return NULL;
}

/**
 * @brief 保存Task
 */
int task_repository_save(TaskRepository* repo, Task* task) {
    if (!repo || !task) {
        return -1;
    }

    pthread_mutex_lock(&repo->mutex);

    TaskID task_id = task_get_id(task);

    // 检查是否已存在
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->tasks[i] && task_get_id(repo->tasks[i]) == task_id) {
            // 更新现有Task
            repo->tasks[i] = task;
            pthread_mutex_unlock(&repo->mutex);
            return 0;
        }
    }

    // 添加新Task
    if (repo->count >= repo->capacity) {
        // 扩展数组
        size_t new_capacity = repo->capacity * 2;
        Task** new_tasks = (Task**)realloc(repo->tasks, new_capacity * sizeof(Task*));
        if (!new_tasks) {
            pthread_mutex_unlock(&repo->mutex);
            return -1;
        }
        repo->tasks = new_tasks;
        repo->capacity = new_capacity;
    }

    repo->tasks[repo->count++] = task;
    pthread_mutex_unlock(&repo->mutex);
    return 0;
}

/**
 * @brief 删除Task
 */
int task_repository_delete(TaskRepository* repo, TaskID task_id) {
    if (!repo || task_id == 0) {
        return -1;
    }

    pthread_mutex_lock(&repo->mutex);

    for (size_t i = 0; i < repo->count; i++) {
        if (repo->tasks[i] && task_get_id(repo->tasks[i]) == task_id) {
            // 移动最后一个元素到当前位置
            repo->tasks[i] = repo->tasks[repo->count - 1];
            repo->tasks[repo->count - 1] = NULL;
            repo->count--;
            pthread_mutex_unlock(&repo->mutex);
            return 0;
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return -1; // 未找到
}

/**
 * @brief 检查Task是否存在
 */
bool task_repository_exists(TaskRepository* repo, TaskID task_id) {
    return task_repository_find_by_id(repo, task_id) != NULL;
}

/**
 * @brief 获取所有Task
 */
int task_repository_find_all(TaskRepository* repo, Task*** tasks, size_t* count) {
    if (!repo || !tasks || !count) {
        return -1;
    }

    pthread_mutex_lock(&repo->mutex);

    *count = repo->count;
    if (repo->count == 0) {
        *tasks = NULL;
        pthread_mutex_unlock(&repo->mutex);
        return 0;
    }

    // 分配数组
    *tasks = (Task**)malloc(repo->count * sizeof(Task*));
    if (!*tasks) {
        pthread_mutex_unlock(&repo->mutex);
        return -1;
    }

    // 复制指针
    memcpy(*tasks, repo->tasks, repo->count * sizeof(Task*));
    pthread_mutex_unlock(&repo->mutex);
    return 0;
}

/**
 * @brief 根据状态查找Task列表
 */
int task_repository_find_by_status(
    TaskRepository* repo,
    TaskStatus status,
    Task*** tasks,
    size_t* count
) {
    if (!repo || !tasks || !count) {
        return -1;
    }

    pthread_mutex_lock(&repo->mutex);

    // 先统计数量
    size_t matching_count = 0;
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->tasks[i] && task_get_status(repo->tasks[i]) == status) {
            matching_count++;
        }
    }

    *count = matching_count;
    if (matching_count == 0) {
        *tasks = NULL;
        pthread_mutex_unlock(&repo->mutex);
        return 0;
    }

    // 分配数组
    *tasks = (Task**)malloc(matching_count * sizeof(Task*));
    if (!*tasks) {
        pthread_mutex_unlock(&repo->mutex);
        return -1;
    }

    // 复制匹配的Task指针
    size_t idx = 0;
    for (size_t i = 0; i < repo->count; i++) {
        if (repo->tasks[i] && task_get_status(repo->tasks[i]) == status) {
            (*tasks)[idx++] = repo->tasks[i];
        }
    }

    pthread_mutex_unlock(&repo->mutex);
    return 0;
}

/**
 * @brief 销毁仓储实例
 */
void task_repository_destroy(TaskRepository* repo) {
    if (!repo) {
        return;
    }

    // 注意：不销毁Task对象，只释放数组
    if (repo->tasks) {
        free(repo->tasks);
    }

    pthread_mutex_destroy(&repo->mutex);
    free(repo);
}
