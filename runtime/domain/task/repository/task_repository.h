#ifndef TASK_REPOSITORY_H
#define TASK_REPOSITORY_H

#include "../entity/task.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct TaskRepository;

// 任务仓储接口 - 定义任务持久化操作
typedef struct TaskRepository {
    // 保存任务
    bool (*save)(struct TaskRepository* repo, const Task* task);

    // 根据ID查找任务
    Task* (*find_by_id)(struct TaskRepository* repo, uint64_t task_id);

    // 查找所有任务
    Task** (*find_all)(struct TaskRepository* repo, size_t* count);

    // 根据状态查找任务
    Task** (*find_by_status)(struct TaskRepository* repo, TaskStatusEnum status, size_t* count);

    // 删除任务
    bool (*delete)(struct TaskRepository* repo, uint64_t task_id);

    // 更新任务
    bool (*update)(struct TaskRepository* repo, const Task* task);

    // 检查任务是否存在
    bool (*exists)(struct TaskRepository* repo, uint64_t task_id);

    // 获取任务总数
    size_t (*count)(struct TaskRepository* repo);

    // 根据状态获取任务数
    size_t (*count_by_status)(struct TaskRepository* repo, TaskStatusEnum status);
} TaskRepository;

// 仓储工厂函数类型
typedef TaskRepository* (*TaskRepositoryFactory)(void);

// 内存实现工厂（用于测试）
TaskRepository* create_memory_task_repository(void);

// 文件实现工厂（用于简单持久化）
TaskRepository* create_file_task_repository(const char* filename);

// 销毁仓储
void destroy_task_repository(TaskRepository* repo);

#endif // TASK_REPOSITORY_H
