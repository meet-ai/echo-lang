/**
 * @file scheduler.c
 * @brief TaskScheduling 聚合根实现
 *
 * 实现Scheduler聚合根的所有方法，遵循DDD原则：
 * - 任务队列存储TaskID列表，不直接存储Task指针
 * - 通过TaskRepository查找Task聚合
 * - Processor和Machine作为内部实体，只能通过聚合根访问
 */

#include "scheduler.h"
#include "../events/scheduler_events.h"  // 领域事件定义
#include "../../shared/events/bus.h"  // 事件总线接口
#include "../../task_execution/repository/task_repository.h"  // TaskRepository接口
#include "../../scheduler/concurrency/processor.h"  // Processor内部实体
#include "../../scheduler/concurrency/machine.h"   // Machine内部实体
// TODO: 阶段7后续重构：scheduler_stop和scheduler_steal_work应该移到聚合根方法中，而不是依赖适配层
#include "../adapter/scheduler_adapter.h"  // 临时包含，用于scheduler_stop和scheduler_steal_work
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdint.h>

// ============================================================================
// 领域事件列表管理（简单实现）
// ============================================================================

typedef struct DomainEventNode {
    void* event;  // 事件指针（TaskScheduledEvent或TaskCompletedEvent）
    struct DomainEventNode* next;
} DomainEventNode;

/**
 * @brief 添加领域事件到列表（向后兼容）
 */
static void scheduler_add_domain_event(Scheduler* scheduler, void* event) {
    if (!scheduler || !event) return;

    DomainEventNode* node = (DomainEventNode*)malloc(sizeof(DomainEventNode));
    if (!node) return;

    node->event = event;
    node->next = NULL;

    // 添加到事件列表头部
    if (scheduler->domain_events == NULL) {
        scheduler->domain_events = node;
    } else {
        DomainEventNode* head = (DomainEventNode*)scheduler->domain_events;
        node->next = head;
        scheduler->domain_events = node;
    }
}

/**
 * @brief 发布领域事件（通过事件总线或存储到列表）
 * 
 * 如果event_bus存在，通过EventBus发布事件。
 * 同时存储到domain_events列表（用于get_domain_events方法）。
 */
static void scheduler_publish_domain_event(Scheduler* scheduler, DomainEvent* domain_event) {
    if (!scheduler || !domain_event) return;

    // 如果事件总线存在，通过EventBus发布
    if (scheduler->event_bus && scheduler->event_bus->publish) {
        int result = scheduler->event_bus->publish(scheduler->event_bus, domain_event);
        if (result != 0) {
            fprintf(stderr, "WARNING: Failed to publish event through EventBus\n");
        }
    }

    // 同时存储到domain_events列表（用于get_domain_events方法）
    DomainEventNode* node = (DomainEventNode*)malloc(sizeof(DomainEventNode));
    if (!node) {
        // 如果存储失败，至少事件已经通过EventBus发布了
        return;
    }

    node->event = domain_event;  // 存储DomainEvent接口
    node->next = NULL;

    // 添加到事件列表头部
    if (scheduler->domain_events == NULL) {
        scheduler->domain_events = node;
    } else {
        DomainEventNode* head = (DomainEventNode*)scheduler->domain_events;
        node->next = head;
        scheduler->domain_events = node;
    }
}

// ============================================================================
// 工厂方法
// ============================================================================

Scheduler* scheduler_factory_create(uint32_t num_processors, TaskRepository* task_repository, struct EventBus* event_bus) {
    if (num_processors == 0) {
        fprintf(stderr, "ERROR: Cannot create scheduler with 0 processors\n");
        return NULL;
    }

    if (!task_repository) {
        fprintf(stderr, "ERROR: TaskRepository is required\n");
        return NULL;
    }

    Scheduler* scheduler = (Scheduler*)calloc(1, sizeof(Scheduler));
    if (!scheduler) {
        fprintf(stderr, "ERROR: Failed to allocate memory for scheduler\n");
        return NULL;
    }
    
    // 初始化事件总线（可选）
    scheduler->event_bus = event_bus;

    // 初始化聚合根属性
    scheduler->id = 1; // 暂时固定ID
    scheduler->num_processors = num_processors;
    scheduler->global_queue = NULL;  // TaskID列表
    scheduler->is_running = false;
    scheduler->task_repository = task_repository;
    scheduler->domain_events = NULL;  // TODO: 实现领域事件列表

    // 初始化锁
    if (pthread_mutex_init(&scheduler->global_lock, NULL) != 0) {
        fprintf(stderr, "ERROR: Failed to initialize scheduler lock\n");
        free(scheduler);
        return NULL;
    }

    // 创建处理器数组（内部实体）
    scheduler->processors = (Processor**)calloc(num_processors, sizeof(Processor*));
    if (!scheduler->processors) {
        fprintf(stderr, "ERROR: Failed to allocate processor array\n");
        pthread_mutex_destroy(&scheduler->global_lock);
        free(scheduler);
        return NULL;
    }

    // 初始化每个处理器
    for (uint32_t i = 0; i < num_processors; i++) {
        scheduler->processors[i] = processor_create(i, 256); // 每个处理器最大256个任务
        if (!scheduler->processors[i]) {
            fprintf(stderr, "ERROR: Failed to create processor %u\n", i);
            // 清理已创建的处理器
            for (uint32_t j = 0; j < i; j++) {
                processor_destroy(scheduler->processors[j]);
            }
            free(scheduler->processors);
            pthread_mutex_destroy(&scheduler->global_lock);
            free(scheduler);
            return NULL;
        }
    }

    // 创建Machine数组（每个Processor绑定一个Machine）
    scheduler->num_machines = num_processors;
    scheduler->machines = (Machine**)calloc(num_processors, sizeof(Machine*));
    if (!scheduler->machines) {
        fprintf(stderr, "ERROR: Failed to allocate machine array\n");
        // 清理已创建的处理器
        for (uint32_t j = 0; j < num_processors; j++) {
            processor_destroy(scheduler->processors[j]);
        }
        free(scheduler->processors);
        pthread_mutex_destroy(&scheduler->global_lock);
        free(scheduler);
        return NULL;
    }

    // 初始化每个Machine并绑定到Processor
    for (uint32_t i = 0; i < num_processors; i++) {
        scheduler->machines[i] = machine_create(i, scheduler);
        if (!scheduler->machines[i]) {
            fprintf(stderr, "ERROR: Failed to create machine %u\n", i);
            // 清理已创建的资源
            for (uint32_t j = 0; j < i; j++) {
                machine_destroy(scheduler->machines[j]);
            }
            for (uint32_t j = 0; j < num_processors; j++) {
                processor_destroy(scheduler->processors[j]);
            }
            free(scheduler->machines);
            free(scheduler->processors);
            pthread_mutex_destroy(&scheduler->global_lock);
            free(scheduler);
            return NULL;
        }

        // 绑定Machine到Processor
        scheduler->machines[i]->processor = scheduler->processors[i];
        scheduler->processors[i]->bound_machine = scheduler->machines[i];
    }

    printf("DEBUG: Created scheduler with %u processors and machines\n", num_processors);
    return scheduler;
}

void scheduler_aggregate_destroy(Scheduler* scheduler) {
    if (!scheduler) return;

    // 停止调度器
    scheduler_stop(scheduler);

    // 清理全局队列（TaskID列表）
    pthread_mutex_lock(&scheduler->global_lock);
    TaskIDNode* node = scheduler->global_queue;
    while (node) {
        TaskIDNode* next = node->next;
        free(node);
        node = next;
    }
    scheduler->global_queue = NULL;
    pthread_mutex_unlock(&scheduler->global_lock);

    // 销毁所有Machine
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        if (scheduler->machines[i]) {
            machine_destroy(scheduler->machines[i]);
            scheduler->machines[i] = NULL;
        }
    }

    // 清理Machine数组
    free(scheduler->machines);
    scheduler->machines = NULL;

    // 销毁所有处理器
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        if (scheduler->processors[i]) {
            processor_destroy(scheduler->processors[i]);
            scheduler->processors[i] = NULL;
        }
    }

    // 清理处理器数组
    free(scheduler->processors);
    scheduler->processors = NULL;

    // 清理锁
    pthread_mutex_destroy(&scheduler->global_lock);

    // 清理领域事件（TODO: 实现具体清理逻辑）
    scheduler->domain_events = NULL;

    printf("DEBUG: Destroyed scheduler\n");
    free(scheduler);
}

// ============================================================================
// 聚合根方法（业务操作）
// ============================================================================

int scheduler_aggregate_add_task(Scheduler* scheduler, TaskID task_id) {
    if (!scheduler) {
        fprintf(stderr, "ERROR: Scheduler is NULL\n");
        return -1;
    }

    // 验证TaskID有效性（通过TaskRepository）
    if (!task_repository_exists(scheduler->task_repository, task_id)) {
        fprintf(stderr, "ERROR: Task %llu does not exist in repository\n", task_id);
        return -1;
    }

    // 首先尝试添加到本地队列（轮询分配到处理器）
    static uint32_t next_processor = 0;

    for (uint32_t attempts = 0; attempts < scheduler->num_processors; attempts++) {
        uint32_t processor_idx = next_processor % scheduler->num_processors;
        next_processor++;

        Processor* processor = scheduler->processors[processor_idx];
        
        // TODO: 阶段5后续需要更新processor_push_local为接受TaskID
        // 临时方案：通过TaskRepository查找Task，然后传递给Processor
        Task* task = task_repository_find_by_id(scheduler->task_repository, task_id);
        if (!task) {
            fprintf(stderr, "ERROR: Failed to find task %llu in repository\n", task_id);
            continue;
        }

        if (processor_push_local(processor, task)) {
            printf("DEBUG: Added task %llu to processor %u local queue\n",
                   task_id, processor->id);
            scheduler->tasks_scheduled++;
            
            // 发布领域事件：TaskScheduled
            TaskScheduledEvent* event = task_scheduled_event_create(
                task_id,
                scheduler->id,
                processor->id
            );
            if (event) {
                // 转换为DomainEvent并发布
                DomainEvent* domain_event = task_scheduled_event_to_domain_event(event);
                if (domain_event) {
                    scheduler_publish_domain_event(scheduler, domain_event);
                } else {
                    // 如果转换失败，至少存储到列表
                    scheduler_add_domain_event(scheduler, event);
                }
            }
            
            return 0;
        }
    }

    // 如果所有本地队列都满，添加到全局队列（TaskID列表）
    pthread_mutex_lock(&scheduler->global_lock);

    // 创建新的TaskID节点
    TaskIDNode* new_node = (TaskIDNode*)malloc(sizeof(TaskIDNode));
    if (!new_node) {
        fprintf(stderr, "ERROR: Failed to allocate memory for TaskID node\n");
        pthread_mutex_unlock(&scheduler->global_lock);
        return -1;
    }

    new_node->task_id = task_id;
    new_node->next = NULL;

    // 添加到全局队列尾部
    if (scheduler->global_queue == NULL) {
        scheduler->global_queue = new_node;
    } else {
        TaskIDNode* tail = scheduler->global_queue;
        while (tail->next) {
            tail = tail->next;
        }
        tail->next = new_node;
    }

    scheduler->tasks_scheduled++;
    printf("DEBUG: Added task %llu to global queue\n", task_id);

    pthread_mutex_unlock(&scheduler->global_lock);
    
    // 发布领域事件：TaskScheduled（processor_id为UINT32_MAX表示全局队列）
    TaskScheduledEvent* event = task_scheduled_event_create(
        task_id,
        scheduler->id,
        UINT32_MAX  // 表示添加到全局队列
    );
    if (event) {
        // 转换为DomainEvent并发布
        DomainEvent* domain_event = task_scheduled_event_to_domain_event(event);
        if (domain_event) {
            scheduler_publish_domain_event(scheduler, domain_event);
        } else {
            // 如果转换失败，至少存储到列表
            scheduler_add_domain_event(scheduler, event);
        }
    }
    
    return 0;
}

int scheduler_aggregate_get_work(Scheduler* scheduler, Processor* processor, TaskID* task_id) {
    if (!scheduler || !processor || !task_id) {
        return -1;
    }

    // 首先尝试从本地队列获取
    // TODO: 阶段5后续需要更新processor_pop_local为返回TaskID
    // 临时方案：从Processor获取Task，然后提取TaskID
    Task* task = processor_pop_local(processor);
    if (task) {
        *task_id = task_get_id(task);
        printf("DEBUG: Processor %u got work from local queue: task %llu\n",
               processor->id, *task_id);
        return 0;
    }

    // 本地队列为空，尝试工作窃取
    // 注意：这里应该调用聚合根方法 scheduler_aggregate_steal_work，而不是适配层方法 scheduler_steal_work
    if (scheduler_aggregate_steal_work(scheduler, processor) == 0) {
        // 窃取成功，从本地队列重新获取
        task = processor_pop_local(processor);
        if (task) {
            *task_id = task_get_id(task);
            printf("DEBUG: Processor %u stole work: task %llu\n",
                   processor->id, *task_id);
            return 0;
        }
    }

    // 工作窃取失败，从全局队列获取（TaskID列表）
    pthread_mutex_lock(&scheduler->global_lock);

    if (scheduler->global_queue) {
        TaskIDNode* node = scheduler->global_queue;
        scheduler->global_queue = node->next;
        *task_id = node->task_id;
        free(node);

        printf("DEBUG: Processor %u got work from global queue: task %llu\n",
               processor->id, *task_id);
    } else {
        *task_id = 0;  // 表示没有任务
    }

    pthread_mutex_unlock(&scheduler->global_lock);
    return (*task_id != 0) ? 0 : -1;
}

int scheduler_aggregate_steal_work(Scheduler* scheduler, Processor* thief) {
    if (!scheduler || !thief) {
        return -1;
    }

    // 尝试从每个其他处理器窃取
    for (uint32_t i = 0; i < scheduler->num_processors; i++) {
        Processor* victim = scheduler->processors[i];
        if (victim == thief) continue; // 不从自己窃取

        // TODO: 阶段5后续需要更新processor_steal_local为返回TaskID
        // 临时方案：从Processor窃取Task，然后提取TaskID
        Task* stolen_task = processor_steal_local(victim);
        if (stolen_task) {
            TaskID stolen_task_id = task_get_id(stolen_task);
            
            // 将窃取的任务添加到自己的本地队列
            if (processor_push_local(thief, stolen_task)) {
                printf("DEBUG: Processor %u successfully stole task %llu from processor %u\n",
                       thief->id, stolen_task_id, victim->id);
                return 0;
            } else {
                // 如果添加失败，暂时放弃这个任务
                fprintf(stderr, "WARNING: Failed to add stolen task to thief queue\n");
            }
        }
    }

    return -1;
}

int scheduler_aggregate_start(Scheduler* scheduler) {
    if (!scheduler) {
        return -1;
    }

    if (scheduler->is_running) {
        fprintf(stderr, "WARNING: Scheduler is already running\n");
        return -1;
    }

    scheduler->is_running = true;

    // 启动所有Machine（OS线程）
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        if (scheduler->machines[i]) {
            // TODO: 实现machine_start方法
            // machine_start(scheduler->machines[i]);
        }
    }

    printf("DEBUG: Scheduler started\n");
    return 0;
}

int scheduler_aggregate_stop(Scheduler* scheduler) {
    if (!scheduler) {
        return -1;
    }

    if (!scheduler->is_running) {
        return 0;  // 已经停止
    }

    scheduler->is_running = false;

    // 停止所有Machine
    for (uint32_t i = 0; i < scheduler->num_machines; i++) {
        if (scheduler->machines[i]) {
            // TODO: 实现machine_stop方法
            // machine_stop(scheduler->machines[i]);
        }
    }

    printf("DEBUG: Scheduler stopped\n");
    return 0;
}

int scheduler_aggregate_notify_task_completed(Scheduler* scheduler, TaskID task_id) {
    if (!scheduler || task_id == 0) {
        return -1;
    }

    // 更新已完成任务计数
    scheduler->tasks_completed++;

    // 创建TaskCompleted事件
    // 注意：processor_id暂时使用UINT32_MAX表示未知（TODO: 从coroutine获取processor_id）
    TaskCompletedEvent* event = task_completed_event_create(
        task_id,
        scheduler->id,
        UINT32_MAX  // TODO: 从coroutine获取实际的processor_id
    );
    if (!event) {
        fprintf(stderr, "ERROR: Failed to create TaskCompletedEvent\n");
        return -1;
    }

    // 转换为DomainEvent并发布
    DomainEvent* domain_event = scheduler_task_completed_event_to_domain_event(event);
    if (domain_event) {
        scheduler_publish_domain_event(scheduler, domain_event);
    } else {
        // 如果转换失败，至少存储到列表
        scheduler_add_domain_event(scheduler, event);
    }

    printf("DEBUG: Task %llu completed, scheduler %u now has %llu completed tasks\n",
           task_id, scheduler->id, scheduler->tasks_completed);

    return 0;
}

// ============================================================================
// 查询方法
// ============================================================================

uint32_t scheduler_get_id(const Scheduler* scheduler) {
    return scheduler ? scheduler->id : 0;
}

bool scheduler_is_running(const Scheduler* scheduler) {
    return scheduler ? scheduler->is_running : false;
}

uint64_t scheduler_get_tasks_scheduled(const Scheduler* scheduler) {
    return scheduler ? scheduler->tasks_scheduled : 0;
}

uint64_t scheduler_get_tasks_completed(const Scheduler* scheduler) {
    return scheduler ? scheduler->tasks_completed : 0;
}

bool scheduler_has_work_in_global_queue(Scheduler* scheduler) {
    if (!scheduler) return false;
    
    pthread_mutex_lock((pthread_mutex_t*)&scheduler->global_lock);
    bool has_work = (scheduler->global_queue != NULL);
    pthread_mutex_unlock((pthread_mutex_t*)&scheduler->global_lock);
    
    return has_work;
}

void scheduler_print_stats(Scheduler* scheduler) {
    if (!scheduler) return;

    printf("=== Scheduler Stats ===\n");
    printf("  Scheduler ID: %u\n", scheduler_get_id(scheduler));
    printf("  Is running: %s\n", scheduler_is_running(scheduler) ? "yes" : "no");
    printf("  Processors: %u\n", scheduler->num_processors);
    printf("  Machines: %u\n", scheduler->num_machines);
    printf("  Tasks scheduled: %llu\n", scheduler_get_tasks_scheduled(scheduler));
    printf("  Tasks completed: %llu\n", scheduler_get_tasks_completed(scheduler));
    
    // TODO: 实现Processor统计信息的打印
}

// ============================================================================
// 不变条件验证
// ============================================================================

int scheduler_validate_invariants(const Scheduler* scheduler) {
    if (!scheduler) {
        return -1;
    }

    // 不变条件1：处理器数量必须大于0
    if (scheduler->num_processors == 0) {
        fprintf(stderr, "ERROR: Scheduler must have at least one processor\n");
        return -1;
    }

    // 不变条件2：机器数量必须等于处理器数量
    if (scheduler->num_machines != scheduler->num_processors) {
        fprintf(stderr, "ERROR: Number of machines must equal number of processors\n");
        return -1;
    }

    // 不变条件3：全局队列中的TaskID必须有效（通过TaskRepository验证）
    pthread_mutex_lock((pthread_mutex_t*)&scheduler->global_lock);
    TaskIDNode* node = scheduler->global_queue;
    while (node) {
        if (!task_repository_exists(scheduler->task_repository, node->task_id)) {
            fprintf(stderr, "ERROR: TaskID %llu in global queue does not exist in repository\n",
                   node->task_id);
            pthread_mutex_unlock((pthread_mutex_t*)&scheduler->global_lock);
            return -1;
        }
        node = node->next;
    }
    pthread_mutex_unlock((pthread_mutex_t*)&scheduler->global_lock);

    // 不变条件4：调度器运行状态必须与处理器状态一致
    // TODO: 实现更详细的验证逻辑

    return 0;
}

// ============================================================================
// 领域事件管理
// ============================================================================

void* scheduler_get_domain_events(Scheduler* scheduler) {
    if (!scheduler) {
        return NULL;
    }

    // 获取并清空领域事件列表
    void* events = scheduler->domain_events;
    scheduler->domain_events = NULL;
    
    // 注意：调用者负责释放事件节点和事件对象
    // 可以使用以下方式释放：
    // DomainEventNode* node = (DomainEventNode*)events;
    // while (node) {
    //     DomainEventNode* next = node->next;
    //     if (node->event) {
    //         // 根据事件类型调用相应的销毁函数
    //         // task_scheduled_event_destroy((TaskScheduledEvent*)node->event);
    //         // 或 task_completed_event_destroy((TaskCompletedEvent*)node->event);
    //     }
    //     free(node);
    //     node = next;
    // }
    
    return events;
}
