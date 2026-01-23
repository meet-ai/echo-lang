#include "task.h"
#include "../../shared/types/common_types.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <time.h>
#include <stdatomic.h>

// 全局任务ID生成器
static atomic_uint_fast64_t g_next_task_id = ATOMIC_VAR_INIT(1);

// 获取下一个任务ID
static uint64_t generate_task_id(void) {
    return atomic_fetch_add(&g_next_task_id, 1);
}

// 计算任务执行时间
static uint64_t calculate_execution_time(const Task* task) {
    if (task->started_at == 0 || task->completed_at == 0) {
        return 0;
    }
    return (uint64_t)(task->completed_at - task->started_at) * 1000000; // 转换为微秒
}

// 创建新任务
Task* task_create(const char* name, void* entry_point, void* arg, size_t stack_size) {
    if (!name || !entry_point) {
        return NULL;
    }

    // 验证参数
    if (strlen(name) == 0) {
        return NULL; // 名称不能为空
    }

    if (stack_size == 0) {
        stack_size = 64 * 1024; // 默认64KB栈大小
    } else if (stack_size < 4096) {
        return NULL; // 最小栈大小4KB
    }

    Task* task = (Task*)malloc(sizeof(Task));
    if (!task) {
        return NULL;
    }

    // 初始化任务
    task->id = generate_task_id(); // 生成唯一ID
    strncpy(task->name, name, sizeof(task->name) - 1);
    task->name[sizeof(task->name) - 1] = '\0';

    // 初始化描述为空
    task->description[0] = '\0';

    // 初始化状态和优先级
    task->status = TASK_STATUS_CREATED;
    task->priority = TASK_PRIORITY_NORMAL;

    // 设置执行参数
    task->entry_point = entry_point;
    task->arg = arg;
    task->stack_size = stack_size;

    // 初始化时间戳
    task->created_at = time(NULL);
    task->scheduled_at = 0;
    task->started_at = 0;
    task->completed_at = 0;

    // 初始化执行结果
    task->exit_code = 0;
    task->error_message[0] = '\0';

    // 初始化统计信息
    memset(&task->stats, 0, sizeof(TaskStats));

    // 初始化其他字段
    task->user_data = NULL;
    task->is_cancelled = false;

    return task;
}

// 销毁任务
void task_destroy(Task* task) {
    if (task) {
        free(task);
    }
}

// 检查任务是否可以调度
bool task_can_schedule(const Task* task) {
    return task && task->status == TASK_STATUS_CREATED;
}

// 检查任务是否可以取消
bool task_can_cancel(const Task* task) {
    return task &&
           (task->status == TASK_STATUS_CREATED ||
            task->status == TASK_STATUS_READY ||
            task->status == TASK_STATUS_SUSPENDED);
}

// 检查任务是否已完成
bool task_is_completed(const Task* task) {
    return task &&
           (task->status == TASK_STATUS_COMPLETED ||
            task->status == TASK_STATUS_FAILED ||
            task->status == TASK_STATUS_CANCELLED);
}

// 更新任务状态
void task_update_status(Task* task, TaskStatus new_status) {
    if (!task) {
        return;
    }

    // 状态转换验证
    switch (task->status) {
        case TASK_STATUS_CREATED:
            if (new_status != TASK_STATUS_READY &&
                new_status != TASK_STATUS_CANCELLED) {
                return; // 无效转换
            }
            break;

        case TASK_STATUS_READY:
            if (new_status != TASK_STATUS_RUNNING &&
                new_status != TASK_STATUS_CANCELLED) {
                return; // 无效转换
            }
            break;

        case TASK_STATUS_RUNNING:
            if (new_status != TASK_STATUS_COMPLETED &&
                new_status != TASK_STATUS_FAILED &&
                new_status != TASK_STATUS_SUSPENDED) {
                return; // 无效转换
            }
            break;

        case TASK_STATUS_SUSPENDED:
            if (new_status != TASK_STATUS_READY &&
                new_status != TASK_STATUS_CANCELLED) {
                return; // 无效转换
            }
            break;

        default:
            return; // 终态无法转换
    }

    // 更新状态和时间戳
    task->status = new_status;

    time_t now = time(NULL);
    switch (new_status) {
        case TASK_STATUS_READY:
            task->scheduled_at = now;
            break;
        case TASK_STATUS_RUNNING:
            task->started_at = now;
            break;
        case TASK_STATUS_COMPLETED:
        case TASK_STATUS_FAILED:
        case TASK_STATUS_CANCELLED:
            task->completed_at = now;
            break;
        default:
            break;
    }
}

// 获取状态字符串
const char* task_status_string(TaskStatus status) {
    switch (status) {
        case TASK_STATUS_CREATED: return "CREATED";
        case TASK_STATUS_READY: return "READY";
        case TASK_STATUS_RUNNING: return "RUNNING";
        case TASK_STATUS_SUSPENDED: return "SUSPENDED";
        case TASK_STATUS_COMPLETED: return "COMPLETED";
        case TASK_STATUS_FAILED: return "FAILED";
        case TASK_STATUS_CANCELLED: return "CANCELLED";
        default: return "UNKNOWN";
    }
}

// 设置任务描述
bool task_set_description(Task* task, const char* description) {
    if (!task || !description) {
        return false;
    }

    strncpy(task->description, description, sizeof(task->description) - 1);
    task->description[sizeof(task->description) - 1] = '\0';
    return true;
}

// 设置任务优先级
bool task_set_priority(Task* task, TaskPriority priority) {
    if (!task) {
        return false;
    }

    // 验证优先级值
    if (priority < TASK_PRIORITY_LOW || priority > TASK_PRIORITY_CRITICAL) {
        return false;
    }

    task->priority = priority;
    return true;
}

// 获取任务优先级字符串
const char* task_priority_string(TaskPriority priority) {
    switch (priority) {
        case TASK_PRIORITY_LOW: return "LOW";
        case TASK_PRIORITY_NORMAL: return "NORMAL";
        case TASK_PRIORITY_HIGH: return "HIGH";
        case TASK_PRIORITY_CRITICAL: return "CRITICAL";
        default: return "UNKNOWN";
    }
}

// 检查任务是否正在运行
bool task_is_running(const Task* task) {
    return task && task->status == TASK_STATUS_RUNNING;
}

// 检查任务是否可重启
bool task_can_restart(const Task* task) {
    return task && task_is_completed(task) &&
           task->status != TASK_STATUS_CANCELLED;
}

// 获取任务执行时间（微秒）
uint64_t task_get_execution_time_us(const Task* task) {
    if (!task) {
        return 0;
    }
    return calculate_execution_time(task);
}

// 获取任务总生命周期时间（微秒）
uint64_t task_get_total_time_us(const Task* task) {
    if (!task || task->created_at == 0) {
        return 0;
    }

    time_t end_time = task->completed_at > 0 ? task->completed_at : time(NULL);
    return (uint64_t)(end_time - task->created_at) * 1000000;
}

// 设置任务退出代码
bool task_set_exit_code(Task* task, int exit_code) {
    if (!task) {
        return false;
    }

    task->exit_code = exit_code;

    // 根据退出代码自动更新状态
    if (exit_code == 0 && task->status == TASK_STATUS_RUNNING) {
        task_update_status(task, TASK_STATUS_COMPLETED);
    } else if (exit_code != 0 && task->status == TASK_STATUS_RUNNING) {
        task_update_status(task, TASK_STATUS_FAILED);
    }

    return true;
}

// 设置任务错误信息
bool task_set_error_message(Task* task, const char* error_message) {
    if (!task) {
        return false;
    }

    if (error_message) {
        strncpy(task->error_message, error_message, sizeof(task->error_message) - 1);
        task->error_message[sizeof(task->error_message) - 1] = '\0';
    } else {
        task->error_message[0] = '\0';
    }

    return true;
}

// 重置任务状态（用于重启）
bool task_reset(Task* task) {
    if (!task || !task_is_completed(task)) {
        return false; // 只能重置已完成的任务
    }

    // 重置时间戳
    task->scheduled_at = 0;
    task->started_at = 0;
    task->completed_at = 0;

    // 重置执行结果
    task->exit_code = 0;
    task->error_message[0] = '\0';

    // 重置状态为创建状态
    task->status = TASK_STATUS_CREATED;

    return true;
}

// 克隆任务
Task* task_clone(const Task* original) {
    if (!original) {
        return NULL;
    }

    Task* clone = task_create(original->name, original->entry_point,
                             original->arg, original->stack_size);
    if (!clone) {
        return NULL;
    }

    // 复制描述和优先级
    task_set_description(clone, original->description);
    clone->priority = original->priority;

    return clone;
}

// 任务比较函数（用于排序）
int task_compare_by_priority(const Task* a, const Task* b) {
    if (!a || !b) {
        return 0;
    }
    return b->priority - a->priority; // 优先级高的排前面
}

int task_compare_by_creation_time(const Task* a, const Task* b) {
    if (!a || !b) {
        return 0;
    }
    return (int)(a->created_at - b->created_at); // 创建时间早的排前面
}

// 获取任务统计信息
bool task_get_statistics(const Task* task, TaskStats* stats) {
    if (!task || !stats) {
        return false;
    }

    // 复制统计信息
    *stats = task->stats;
    return true;
}

// 验证任务状态一致性
bool task_validate_state(const Task* task) {
    if (!task) {
        return false;
    }

    // 检查时间戳一致性
    if (task->started_at > 0 && task->created_at > task->started_at) {
        return false; // 创建时间不能晚于开始时间
    }

    if (task->completed_at > 0 && task->started_at > task->completed_at) {
        return false; // 开始时间不能晚于完成时间
    }

    // 检查状态和时间戳的对应关系
    switch (task->status) {
        case TASK_STATUS_CREATED:
            if (task->started_at > 0 || task->completed_at > 0) {
                return false; // 创建状态不应该有开始或完成时间
            }
            break;
        case TASK_STATUS_READY:
            if (task->started_at > 0) {
                return false; // 就绪状态不应该有开始时间
            }
            break;
        case TASK_STATUS_RUNNING:
            if (task->started_at == 0) {
                return false; // 运行状态必须有开始时间
            }
            break;
        case TASK_STATUS_COMPLETED:
        case TASK_STATUS_FAILED:
        case TASK_STATUS_CANCELLED:
            if (task->completed_at == 0) {
                return false; // 完成状态必须有完成时间
            }
            break;
        default:
            break;
    }

    return true;
}
