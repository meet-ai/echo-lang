#include "channel.h"
// 注意：不再直接包含旧的coroutine.h，因为新的Channel聚合根已经包含了新的coroutine.h
// #include "../coroutine/coroutine.h"  // 已移除，避免与新的coroutine.h冲突
#include "../task_execution/aggregate/task.h"  // 使用新的Task聚合根
#include "../scheduler/scheduler.h"
#include "../channel_communication/adapter/channel_adapter.h"  // 使用适配层（会间接包含新的coroutine.h）
#include "../channel_communication/aggregate/channel.h"  // 使用新的Channel聚合根
#include "../task_execution/repository/task_repository.h"  // 使用TaskRepository
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// 创建通道
// TODO: 阶段4后续重构：应该直接使用新的Channel聚合根，而不是创建旧的Channel结构
// 当前暂时创建旧的Channel结构，同时创建新的Channel聚合根并注册映射
void* channel_create_impl() {
    printf("DEBUG: channel_create_impl called\n");
    
    // 创建新的Channel聚合根（使用新的工厂方法）
    extern Channel* channel_factory_create_unbuffered(struct EventBus* event_bus);
    // 从适配层获取EventBus
    extern EventBus* channel_adapter_get_event_bus(void);
    EventBus* event_bus = channel_adapter_get_event_bus();
    Channel* new_channel = channel_factory_create_unbuffered(event_bus);
    if (!new_channel) {
        printf("DEBUG: Failed to create new channel aggregate\n");
        return NULL;
    }
    
    // 创建旧的Channel结构（用于向后兼容）
    Channel* legacy_channel = (Channel*)malloc(sizeof(Channel));
    if (!legacy_channel) {
        printf("DEBUG: Failed to allocate legacy channel\n");
        channel_aggregate_destroy(new_channel);  // 清理新创建的聚合根
        return NULL;
    }
    
    // 从新的聚合根获取ID（用于兼容）
    uint64_t channel_id = channel_get_id(new_channel);
    legacy_channel->id = channel_id;
    legacy_channel->capacity = 0; // 无缓冲通道（默认）
    legacy_channel->size = 0;
    legacy_channel->read_pos = 0;
    legacy_channel->write_pos = 0;
    legacy_channel->state = CHANNEL_OPEN;
    legacy_channel->buffer = NULL; // 无缓冲区（无缓冲通道不需要）
    legacy_channel->temp_value = NULL; // 临时值存储
    legacy_channel->sender_queue = NULL;  // 旧的队列（不再使用，但保留用于兼容）
    legacy_channel->receiver_queue = NULL;  // 旧的队列（不再使用，但保留用于兼容）

    // 初始化同步机制（旧的Channel结构需要，但实际使用新的聚合根的同步机制）
    pthread_mutex_init(&legacy_channel->lock, NULL);
    pthread_cond_init(&legacy_channel->send_cond, NULL);
    pthread_cond_init(&legacy_channel->recv_cond, NULL);

    // 注册映射：将旧的Channel*映射到新的Channel聚合根
    channel_adapter_register_mapping(legacy_channel, new_channel);
    
    printf("DEBUG: Created unbuffered channel %llu (legacy: %p, new: %p)\n", 
           channel_id, legacy_channel, new_channel);
    return legacy_channel;
}

// 创建有缓冲通道
// TODO: 阶段4后续重构：应该直接使用新的Channel聚合根，而不是创建旧的Channel结构
// 当前暂时创建旧的Channel结构，同时创建新的Channel聚合根并注册映射
void* channel_create_buffered_impl(uint32_t capacity) {
    if (capacity == 0) {
        printf("DEBUG: Cannot create buffered channel with capacity 0, use channel_create_impl() instead\n");
        return NULL;
    }

    printf("DEBUG: channel_create_buffered_impl called with capacity %u\n", capacity);
    
    // 创建新的Channel聚合根（使用新的工厂方法）
    extern Channel* channel_factory_create_buffered(uint32_t capacity, struct EventBus* event_bus);
    // 从适配层获取EventBus
    extern EventBus* channel_adapter_get_event_bus(void);
    EventBus* event_bus = channel_adapter_get_event_bus();
    Channel* new_channel = channel_factory_create_buffered(capacity, event_bus);
    if (!new_channel) {
        printf("DEBUG: Failed to create new buffered channel aggregate\n");
        return NULL;
    }
    
    // 创建旧的Channel结构（用于向后兼容）
    Channel* legacy_channel = (Channel*)malloc(sizeof(Channel));
    if (!legacy_channel) {
        printf("DEBUG: Failed to allocate legacy channel\n");
        channel_aggregate_destroy(new_channel);  // 清理新创建的聚合根
        return NULL;
    }
    
    // 从新的聚合根获取ID和容量（用于兼容）
    uint64_t channel_id = channel_get_id(new_channel);
    uint32_t channel_capacity = channel_get_capacity(new_channel);
    legacy_channel->id = channel_id;
    legacy_channel->capacity = channel_capacity;
    legacy_channel->size = 0;
    legacy_channel->read_pos = 0;
    legacy_channel->write_pos = 0;
    legacy_channel->state = CHANNEL_OPEN;
    legacy_channel->temp_value = NULL;
    legacy_channel->sender_queue = NULL;  // 旧的队列（不再使用，但保留用于兼容）
    legacy_channel->receiver_queue = NULL;  // 旧的队列（不再使用，但保留用于兼容）

    // 分配环形缓冲区（旧的Channel结构需要，但实际使用新的聚合根的缓冲区）
    legacy_channel->buffer = (void**)malloc(sizeof(void*) * capacity);
    if (!legacy_channel->buffer) {
        printf("DEBUG: Failed to allocate buffer for legacy channel %llu\n", channel_id);
        free(legacy_channel);
        channel_aggregate_destroy(new_channel);  // 清理新创建的聚合根
        return NULL;
    }

    // 初始化缓冲区为 NULL
    for (uint32_t i = 0; i < capacity; i++) {
        legacy_channel->buffer[i] = NULL;
    }

    // 初始化同步机制（旧的Channel结构需要，但实际使用新的聚合根的同步机制）
    pthread_mutex_init(&legacy_channel->lock, NULL);
    pthread_cond_init(&legacy_channel->send_cond, NULL);
    pthread_cond_init(&legacy_channel->recv_cond, NULL);

    // 注册映射：将旧的Channel*映射到新的Channel聚合根
    channel_adapter_register_mapping(legacy_channel, new_channel);

    printf("DEBUG: Created buffered channel %llu with capacity %u (legacy: %p, new: %p)\n", 
           channel_id, capacity, legacy_channel, new_channel);
    return legacy_channel;
}

// 发送数据到通道（真正的阻塞实现）
// TODO: 阶段4后续重构：应该直接调用新的聚合根方法，而不是通过适配层
// 当前暂时通过适配层调用新的聚合根方法
void channel_send_impl(Channel* ch, void* value) {
    if (!ch) {
        printf("DEBUG: channel_send_impl called with NULL channel\n");
        return;
    }

    printf("DEBUG: channel_send_impl called on channel %llu with value %p\n", ch->id, value);
    
    // 获取TaskRepository（暂时使用全局TaskRepository）
    TaskRepository* task_repo = channel_adapter_get_task_repository();
    if (!task_repo) {
        printf("DEBUG: channel_send_impl: TaskRepository not available, creating temporary one\n");
        // TODO: 阶段4后续重构：应该通过依赖注入传递TaskRepository，而不是创建临时实例
        extern TaskRepository* task_repository_create(void);
        task_repo = task_repository_create();
        if (!task_repo) {
            printf("DEBUG: channel_send_impl: Failed to create TaskRepository\n");
            return;
        }
        channel_adapter_set_task_repository(task_repo);
    }
    
    // 通过适配层调用新的聚合根方法
    int result = channel_adapter_send(ch, task_repo, value);
    if (result != 0) {
        printf("DEBUG: channel_send_impl: Failed to send via adapter (result=%d)\n", result);
    }
    // 旧的实现逻辑已移除，现在通过适配层调用新的聚合根方法
    // 所有业务逻辑都在新的Channel聚合根中实现
}

// 从通道接收数据（真正的阻塞实现）
// TODO: 阶段4后续重构：应该直接调用新的聚合根方法，而不是通过适配层
// 当前暂时通过适配层调用新的聚合根方法
void* channel_receive_impl(Channel* ch) {
    if (!ch) {
        printf("DEBUG: channel_receive_impl called with NULL channel\n");
        return NULL;
    }

    printf("DEBUG: channel_receive_impl called on channel %llu\n", ch->id);
    
    // 获取TaskRepository（暂时使用全局TaskRepository）
    TaskRepository* task_repo = channel_adapter_get_task_repository();
    if (!task_repo) {
        printf("DEBUG: channel_receive_impl: TaskRepository not available, creating temporary one\n");
        // TODO: 阶段4后续重构：应该通过依赖注入传递TaskRepository，而不是创建临时实例
        extern TaskRepository* task_repository_create(void);
        task_repo = task_repository_create();
        if (!task_repo) {
            printf("DEBUG: channel_receive_impl: Failed to create TaskRepository\n");
            return NULL;
        }
        channel_adapter_set_task_repository(task_repo);
    }
    
    // 通过适配层调用新的聚合根方法
    void* result = channel_adapter_receive(ch, task_repo);
    if (!result) {
        printf("DEBUG: channel_receive_impl: Failed to receive via adapter\n");
    }
    // 旧的实现逻辑已移除，现在通过适配层调用新的聚合根方法
    // 所有业务逻辑都在新的Channel聚合根中实现
    return result;
}

// 旧的channel_receive_impl实现已移除，现在通过适配层调用新的聚合根方法
// 以下代码保留作为参考，但不再执行
#if 0
void* channel_receive_impl_old(Channel* ch) {
    pthread_mutex_lock(&ch->lock);
    
    // 对于无缓冲通道
    if (ch->capacity == 0) {
        // 检查是否有接收者在等待
        Coroutine* receiver = ch->receiver_queue;
        if (receiver) {
            // 从等待队列移除接收者
            ch->receiver_queue = receiver->next;
            receiver->next = NULL;

            printf("DEBUG: Sending value %p directly to receiver coroutine %llu on channel %llu\n", 
                   value, receiver->id, ch->id);

            // 将值存储到通道的临时值存储，供接收者获取
            ch->temp_value = value;
            ch->size = 1;  // 标记有数据

            // 唤醒接收者协程
            // 如果接收者有关联的任务，唤醒任务
            if (receiver->task) {
                Task* task = (Task*)receiver->task;  // 类型转换：旧的struct Task* -> 新的Task*
                // TODO: 阶段4需要更新为使用新的Task聚合根方法，而不是直接操作mutex和status
                pthread_mutex_lock(&task->mutex);  // 旧的方式，阶段4更新
                if (task->status == TASK_WAITING) {  // 旧的方式，阶段4更新
                    task->status = TASK_READY;  // 旧的方式，阶段4更新
                    TaskID task_id = task_get_id(task);  // 使用新的聚合根方法获取ID
                    printf("DEBUG: Waking task %llu for receiver coroutine %llu\n", 
                           task_id, receiver->id);
                    pthread_cond_signal(&task->cond);  // 旧的方式，阶段4更新
                }
                pthread_mutex_unlock(&task->mutex);  // 旧的方式，阶段4更新
            }

            // 恢复接收者协程状态
            receiver->state = COROUTINE_READY;
            printf("DEBUG: Set receiver coroutine %llu state to READY\n", receiver->id);

            // 将接收者任务重新加入调度队列
            if (receiver->task) {
                extern Scheduler* get_global_scheduler(void);
                Scheduler* scheduler = get_global_scheduler();
                if (scheduler) {
                    Processor* processor = receiver->bound_processor;
                    if (!processor && scheduler->num_processors > 0) {
                        processor = scheduler->processors[0];
                    }
                    
                    if (processor) {
                        extern bool processor_push_local(Processor* processor, Task* task);
                        processor_push_local(processor, receiver->task);
                        TaskID task_id = task_get_id((Task*)receiver->task);  // 使用新的聚合根方法
                        printf("DEBUG: Pushed receiver task %llu back to processor queue\n", 
                               task_id);
                    }
                }
            }
        } else {
            // 没有接收者，发送者需要阻塞
            // 获取当前协程
            // extern Task* current_task;  // 已在scheduler.h中声明
            Coroutine* sender = NULL;
            // TODO: 阶段4需要更新为使用新的Task聚合根方法：task_get_coroutine(current_task)
            if (current_task) {
                const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
                sender = (Coroutine*)coroutine;  // 类型转换
            }

            if (sender) {
                // 将值存储到通道的临时值存储，等待接收者
                ch->temp_value = value;
                ch->size = 1;  // 标记有数据

                // 将发送者加入等待队列
                sender->next = ch->sender_queue;
                ch->sender_queue = sender;
                sender->state = COROUTINE_SUSPENDED;
                printf("DEBUG: Sender coroutine %llu blocking on channel %llu with value %p\n", 
                       sender->id, ch->id, value);

                // 将任务状态设置为等待
                // TODO: 阶段4需要更新为使用新的Task聚合根方法
                // 新的Task聚合根没有mutex字段（同步在聚合根方法内部处理）
                // 应该使用task_wait_for_future()等方法，而不是直接访问字段
                if (current_task) {
                    // TODO: 阶段4更新为：task_wait_for_future(current_task, future_id) 或类似方法
                    // 当前暂时保留旧的方式（可能无法编译，需要在阶段4中一起更新）
                    // pthread_mutex_lock(&current_task->mutex);  // 旧的方式，暂时注释
                    // current_task->status = TASK_WAITING;  // 旧的方式，暂时注释
                    // pthread_mutex_unlock(&current_task->mutex);  // 旧的方式，暂时注释
                }
            }

            // 等待接收者（使用条件变量等待，但主要依赖协程调度）
            while (ch->receiver_queue == NULL && ch->state == CHANNEL_OPEN) {
                printf("DEBUG: Channel %llu has no receiver, sender blocking\n", ch->id);
                pthread_cond_wait(&ch->send_cond, &ch->lock);
            }

            if (ch->state == CHANNEL_CLOSED) {
                // 通道关闭，从等待队列移除发送者
                if (sender) {
                    // 从等待队列移除
                    if (ch->sender_queue == sender) {
                        ch->sender_queue = sender->next;
                        sender->next = NULL;
                    } else {
                        // 从队列中间移除
                        Coroutine* prev = ch->sender_queue;
                        while (prev && prev->next != sender) {
                            prev = prev->next;
                        }
                        if (prev) {
                            prev->next = sender->next;
                            sender->next = NULL;
                        }
                    }
                }
                ch->temp_value = NULL;
                ch->size = 0;
                printf("DEBUG: Channel %llu closed while waiting\n", ch->id);
                pthread_mutex_unlock(&ch->lock);
                return;
            }

            // 接收者已经接收了值，发送者可以继续
            // 值已经被接收者取走，清空临时值
            ch->temp_value = NULL;
            ch->size = 0;
        }
    } else {
        // 有缓冲区的通道
        // 检查缓冲区是否已满
        if (ch->size >= ch->capacity) {
            // 缓冲区已满，发送者需要阻塞
            // extern Task* current_task;  // 已在scheduler.h中声明
            Coroutine* sender = NULL;
            // TODO: 阶段4需要更新为使用新的Task聚合根方法：task_get_coroutine(current_task)
            if (current_task) {
                const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
                sender = (Coroutine*)coroutine;  // 类型转换
            }

            if (sender) {
                // 将发送者加入等待队列
                sender->next = ch->sender_queue;
                ch->sender_queue = sender;
                sender->state = COROUTINE_SUSPENDED;
                printf("DEBUG: Sender coroutine %llu blocking on full channel %llu (capacity=%u, size=%u)\n",
                       sender->id, ch->id, ch->capacity, ch->size);

                // 将任务状态设置为等待
                if (current_task) {
                    pthread_mutex_lock(&current_task->mutex);
                    current_task->status = TASK_WAITING;
                    pthread_mutex_unlock(&current_task->mutex);
                }
            }

            // 等待缓冲区有空间
            while (ch->size >= ch->capacity && ch->state == CHANNEL_OPEN) {
                printf("DEBUG: Channel %llu buffer full, sender blocking\n", ch->id);
                pthread_cond_wait(&ch->send_cond, &ch->lock);
            }

            if (ch->state == CHANNEL_CLOSED) {
                // 通道关闭，从等待队列移除发送者
                if (sender) {
                    if (ch->sender_queue == sender) {
                        ch->sender_queue = sender->next;
                        sender->next = NULL;
                    } else {
                        Coroutine* prev = ch->sender_queue;
                        while (prev && prev->next != sender) {
                            prev = prev->next;
                        }
                        if (prev) {
                            prev->next = sender->next;
                            sender->next = NULL;
                        }
                    }
                }
                printf("DEBUG: Channel %llu closed while waiting\n", ch->id);
                pthread_mutex_unlock(&ch->lock);
                return;
            }
        }

        // 缓冲区有空间，写入值
        ch->buffer[ch->write_pos] = value;
        ch->write_pos = (ch->write_pos + 1) % ch->capacity;
        ch->size++;
        printf("DEBUG: Wrote value %p to buffer position %u, size=%u\n",
               value, (ch->write_pos - 1 + ch->capacity) % ch->capacity, ch->size);

        // 如果有接收者在等待，唤醒一个接收者
        if (ch->receiver_queue) {
            Coroutine* receiver = ch->receiver_queue;
            ch->receiver_queue = receiver->next;
            receiver->next = NULL;

            if (receiver->task) {
                Task* task = (Task*)receiver->task;  // 类型转换
                // TODO: 阶段4需要更新为使用新的Task聚合根方法
                pthread_mutex_lock(&task->mutex);  // 旧的方式，阶段4更新
                if (task->status == TASK_WAITING) {  // 旧的方式，阶段4更新
                    task->status = TASK_READY;  // 旧的方式，阶段4更新
                    TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
                    printf("DEBUG: Waking receiver task %llu for coroutine %llu\n",
                           task_id, receiver->id);
                    pthread_cond_signal(&task->cond);  // 旧的方式，阶段4更新
                }
                pthread_mutex_unlock(&task->mutex);  // 旧的方式，阶段4更新
            }

            receiver->state = COROUTINE_READY;
            printf("DEBUG: Set receiver coroutine %llu state to READY\n", receiver->id);

            // 将接收者任务重新加入调度队列
            if (receiver->task) {
                extern Scheduler* get_global_scheduler(void);
                Scheduler* scheduler = get_global_scheduler();
                if (scheduler) {
                    Processor* processor = receiver->bound_processor;
                    if (!processor && scheduler->num_processors > 0) {
                        processor = scheduler->processors[0];
                    }
                    
                    if (processor) {
                        extern bool processor_push_local(Processor* processor, Task* task);
                        processor_push_local(processor, receiver->task);
                        TaskID task_id = task_get_id((Task*)receiver->task);  // 使用新的聚合根方法
                        printf("DEBUG: Pushed receiver task %llu back to processor queue\n",
                               task_id);
                    }
                }
            }
        }
    }

    // 唤醒等待的接收者（如果有）
    pthread_cond_broadcast(&ch->recv_cond);
    pthread_mutex_unlock(&ch->lock);

    printf("DEBUG: Sent value %p to channel %llu\n", value, ch->id);
}

// 从通道接收数据（真正的阻塞实现）
void* channel_receive_impl(Channel* ch) {
    if (!ch) {
        printf("DEBUG: channel_receive_impl called with NULL channel\n");
        return NULL;
    }

    printf("DEBUG: channel_receive_impl called on channel %llu\n", ch->id);

    pthread_mutex_lock(&ch->lock);

    // 对于无缓冲通道
    if (ch->capacity == 0) {
        // 检查是否有发送者在等待
        Coroutine* sender = ch->sender_queue;
        if (sender) {
            // 有发送者在等待，直接接收值
            // 从等待队列移除发送者
            ch->sender_queue = sender->next;
            sender->next = NULL;

            // 发送者应该已经将值存储在通道的临时值存储
            void* value = ch->temp_value;
            ch->temp_value = NULL;
            ch->size = 0;

            printf("DEBUG: Receiving value %p from sender coroutine %llu on channel %llu\n", 
                   value, sender->id, ch->id);

            // 唤醒发送者协程
            if (sender->task) {
                Task* task = (Task*)sender->task;  // 类型转换
                // TODO: 阶段4需要更新为使用新的Task聚合根方法
                pthread_mutex_lock(&task->mutex);  // 旧的方式，阶段4更新
                if (task->status == TASK_WAITING) {  // 旧的方式，阶段4更新
                    task->status = TASK_READY;  // 旧的方式，阶段4更新
                    TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
                    printf("DEBUG: Waking task %llu for sender coroutine %llu\n", 
                           task_id, sender->id);
                    pthread_cond_signal(&task->cond);  // 旧的方式，阶段4更新
                }
                pthread_mutex_unlock(&task->mutex);  // 旧的方式，阶段4更新
            }

            // 恢复发送者协程状态
            sender->state = COROUTINE_READY;
            printf("DEBUG: Set sender coroutine %llu state to READY\n", sender->id);

            // 将发送者任务重新加入调度队列
            if (sender->task) {
                extern Scheduler* get_global_scheduler(void);
                Scheduler* scheduler = get_global_scheduler();
                if (scheduler) {
                    Processor* processor = sender->bound_processor;
                    if (!processor && scheduler->num_processors > 0) {
                        processor = scheduler->processors[0];
                    }
                    
                    if (processor) {
                        extern bool processor_push_local(Processor* processor, Task* task);
                        processor_push_local(processor, sender->task);
                        printf("DEBUG: Pushed sender task %llu back to processor queue\n", 
                               sender->task->id);
                    }
                }
            }

            pthread_mutex_unlock(&ch->lock);
            printf("DEBUG: Received value %p from channel %llu\n", value, ch->id);
            return value;
        }

        // 没有发送者，接收者需要阻塞
        // 获取当前协程
        // extern Task* current_task;  // 已在scheduler.h中声明
        Coroutine* receiver = NULL;
        // TODO: 阶段4需要更新为使用新的Task聚合根方法：task_get_coroutine(current_task)
        if (current_task) {
            const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
            receiver = (Coroutine*)coroutine;  // 类型转换
        }

        if (receiver) {
            // 将接收者加入等待队列
            receiver->next = ch->receiver_queue;
            ch->receiver_queue = receiver;
            receiver->state = COROUTINE_SUSPENDED;
            printf("DEBUG: Receiver coroutine %llu blocking on channel %llu\n", 
                   receiver->id, ch->id);

            // 将任务状态设置为等待
            if (current_task) {
                pthread_mutex_lock(&current_task->mutex);
                current_task->status = TASK_WAITING;
                pthread_mutex_unlock(&current_task->mutex);
            }
        }

        // 等待有发送者（使用条件变量等待，但主要依赖协程调度）
        while (ch->size == 0 && ch->state == CHANNEL_OPEN) {
            printf("DEBUG: Channel %llu has no data, receiver blocking\n", ch->id);
            pthread_cond_wait(&ch->recv_cond, &ch->lock);
        }

        if (ch->state == CHANNEL_CLOSED && ch->size == 0) {
            // 通道关闭且无数据，从等待队列移除接收者
            if (receiver) {
                // 从等待队列移除
                if (ch->receiver_queue == receiver) {
                    ch->receiver_queue = receiver->next;
                    receiver->next = NULL;
                } else {
                    // 从队列中间移除
                    Coroutine* prev = ch->receiver_queue;
                    while (prev && prev->next != receiver) {
                        prev = prev->next;
                    }
                    if (prev) {
                        prev->next = receiver->next;
                        receiver->next = NULL;
                    }
                }
            }
            printf("DEBUG: Channel %llu closed and empty\n", ch->id);
            pthread_mutex_unlock(&ch->lock);
            return NULL; // 通道关闭且无数据
        }

        // 有数据，接收它（发送者已经设置好了）
        void* value = ch->temp_value;
        ch->size = 0;
        ch->temp_value = NULL;

        // 从等待队列移除接收者（如果还在队列中）
        if (receiver && ch->receiver_queue == receiver) {
            ch->receiver_queue = receiver->next;
            receiver->next = NULL;
        }

        // 唤醒等待的发送者（如果有）
        pthread_cond_broadcast(&ch->send_cond);

        pthread_mutex_unlock(&ch->lock);
        printf("DEBUG: Received value %p from channel %llu\n", value, ch->id);
        return value;
    } else {
        // 有缓冲区的通道
        // 检查缓冲区是否为空
        if (ch->size == 0) {
            // 缓冲区为空，接收者需要阻塞
            // extern Task* current_task;  // 已在scheduler.h中声明
            Coroutine* receiver = NULL;
            // TODO: 阶段4需要更新为使用新的Task聚合根方法：task_get_coroutine(current_task)
            if (current_task) {
                const struct Coroutine* coroutine = task_get_coroutine(current_task);  // 使用新的聚合根方法
                receiver = (Coroutine*)coroutine;  // 类型转换
            }

            if (receiver) {
                // 将接收者加入等待队列
                receiver->next = ch->receiver_queue;
                ch->receiver_queue = receiver;
                receiver->state = COROUTINE_SUSPENDED;
                printf("DEBUG: Receiver coroutine %llu blocking on empty channel %llu (capacity=%u, size=%u)\n",
                       receiver->id, ch->id, ch->capacity, ch->size);

                // 将任务状态设置为等待
                if (current_task) {
                    pthread_mutex_lock(&current_task->mutex);
                    current_task->status = TASK_WAITING;
                    pthread_mutex_unlock(&current_task->mutex);
                }
            }

            // 等待缓冲区有数据
            while (ch->size == 0 && ch->state == CHANNEL_OPEN) {
                printf("DEBUG: Channel %llu buffer empty, receiver blocking\n", ch->id);
                pthread_cond_wait(&ch->recv_cond, &ch->lock);
            }

            if (ch->state == CHANNEL_CLOSED && ch->size == 0) {
                // 通道关闭且无数据，从等待队列移除接收者
                if (receiver) {
                    if (ch->receiver_queue == receiver) {
                        ch->receiver_queue = receiver->next;
                        receiver->next = NULL;
                    } else {
                        Coroutine* prev = ch->receiver_queue;
                        while (prev && prev->next != receiver) {
                            prev = prev->next;
                        }
                        if (prev) {
                            prev->next = receiver->next;
                            receiver->next = NULL;
                        }
                    }
                }
                printf("DEBUG: Channel %llu closed and empty\n", ch->id);
                pthread_mutex_unlock(&ch->lock);
                return NULL;
            }
        }

        // 缓冲区有数据，读取值
        void* value = ch->buffer[ch->read_pos];
        ch->buffer[ch->read_pos] = NULL; // 清空位置
        ch->read_pos = (ch->read_pos + 1) % ch->capacity;
        ch->size--;
        printf("DEBUG: Read value %p from buffer position %u, size=%u\n",
               value, (ch->read_pos - 1 + ch->capacity) % ch->capacity, ch->size);

        // 如果有发送者在等待，唤醒一个发送者
        if (ch->sender_queue) {
            Coroutine* sender = ch->sender_queue;
            ch->sender_queue = sender->next;
            sender->next = NULL;

            if (sender->task) {
                Task* task = sender->task;
                pthread_mutex_lock(&task->mutex);
                if (task->status == TASK_WAITING) {
                    task->status = TASK_READY;
                    printf("DEBUG: Waking sender task %llu for coroutine %llu\n",
                           task->id, sender->id);
                    pthread_cond_signal(&task->cond);
                }
                pthread_mutex_unlock(&task->mutex);
            }

            sender->state = COROUTINE_READY;
            printf("DEBUG: Set sender coroutine %llu state to READY\n", sender->id);

            // 将发送者任务重新加入调度队列
            if (sender->task) {
                extern Scheduler* get_global_scheduler(void);
                Scheduler* scheduler = get_global_scheduler();
                if (scheduler) {
                    Processor* processor = sender->bound_processor;
                    if (!processor && scheduler->num_processors > 0) {
                        processor = scheduler->processors[0];
                    }
                    
                    if (processor) {
                        extern bool processor_push_local(Processor* processor, Task* task);
                        processor_push_local(processor, sender->task);
                        TaskID task_id = task_get_id((Task*)sender->task);  // 使用新的聚合根方法
                        printf("DEBUG: Pushed sender task %llu back to processor queue\n",
                               task_id);
                    }
                }
            }
        }

        // 唤醒等待的发送者（如果有）
        pthread_cond_broadcast(&ch->send_cond);
        pthread_mutex_unlock(&ch->lock);
        printf("DEBUG: Received value %p from channel %llu\n", value, ch->id);
        return value;
    }
}
#endif  // 关闭 channel_receive_impl_old 的 #if 0 块（第205行开始）

// select语句支持（真正的多路复用实现）
int32_t channel_select_impl(int32_t num_cases, Channel** channels, int32_t* operations, int32_t* result_ptr) {
    printf("DEBUG: channel_select_impl called with %d cases\n", num_cases);

    // 首先尝试非阻塞检查
    for (int32_t i = 0; i < num_cases; i++) {
        Channel* ch = channels[i];
        int32_t op = operations[i];

        if (!ch) continue;

        printf("DEBUG: Checking case %d: channel %llu, operation %d\n", i, ch->id, op);

        pthread_mutex_lock(&ch->lock);

        if (op == 0) { // 接收操作
            if (ch->size > 0 || ch->state == CHANNEL_CLOSED) {
                // 有数据可以接收或通道已关闭
                printf("DEBUG: Case %d (receive) ready\n", i);
                pthread_mutex_unlock(&ch->lock);
                if (result_ptr) *result_ptr = i;
                return i;
            }
        } else if (op == 1) { // 发送操作
            if (ch->capacity == 0) {
                // 无缓冲通道：如果有等待的接收者，就可以发送
                if (ch->receiver_queue != NULL) {
                    printf("DEBUG: Case %d (send) ready\n", i);
                    pthread_mutex_unlock(&ch->lock);
                    if (result_ptr) *result_ptr = i;
                    return i;
                }
            } else {
                // 有缓冲通道：如果缓冲区有空间
                if (ch->size < ch->capacity) {
                    printf("DEBUG: Case %d (send) ready\n", i);
                    pthread_mutex_unlock(&ch->lock);
                    if (result_ptr) *result_ptr = i;
                    return i;
                }
            }
        }

        pthread_mutex_unlock(&ch->lock);
    }

    // 如果没有case立即就绪，需要阻塞等待
    // TODO: 实现真正的select阻塞等待逻辑
    // 目前简化：返回-1表示无就绪case

    printf("DEBUG: No cases ready in select, would block\n");
    if (result_ptr) *result_ptr = -1;
    return -1;
}

// 关闭通道
// TODO: 阶段4后续重构：应该直接调用新的聚合根方法，而不是通过适配层
// 当前暂时通过适配层调用新的聚合根方法
void channel_close_impl(Channel* ch) {
    if (!ch) {
        printf("DEBUG: channel_close_impl called with NULL channel\n");
        return;
    }

    printf("DEBUG: channel_close_impl called on channel %llu\n", ch->id);
    
    // 获取TaskRepository（暂时使用全局TaskRepository）
    TaskRepository* task_repo = channel_adapter_get_task_repository();
    if (!task_repo) {
        printf("DEBUG: channel_close_impl: TaskRepository not available, creating temporary one\n");
        // TODO: 阶段4后续重构：应该通过依赖注入传递TaskRepository，而不是创建临时实例
        extern TaskRepository* task_repository_create(void);
        task_repo = task_repository_create();
        if (!task_repo) {
            printf("DEBUG: channel_close_impl: Failed to create TaskRepository\n");
            return;
        }
        channel_adapter_set_task_repository(task_repo);
    }
    
    // 通过适配层调用新的聚合根方法
    channel_adapter_close(ch, task_repo);
    // 旧的实现逻辑已移除，现在通过适配层调用新的聚合根方法
    // 所有业务逻辑都在新的Channel聚合根中实现
}

// 销毁通道
void channel_destroy_impl(Channel* ch) {
    if (!ch) return;

    printf("DEBUG: Destroying channel %llu\n", ch->id);

    // 确保通道已关闭
    if (ch->state != CHANNEL_CLOSED) {
        channel_close_impl(ch);
    }

    // 清理同步机制
    pthread_mutex_destroy(&ch->lock);
    pthread_cond_destroy(&ch->send_cond);
    pthread_cond_destroy(&ch->recv_cond);

    // 释放缓冲区（如果有的话，仅用于有缓冲通道）
    if (ch->buffer && ch->capacity > 0) {
        free(ch->buffer);
        ch->buffer = NULL;
    }

    // 清理临时值（如果有的话）
    ch->temp_value = NULL;

    free(ch);
    printf("DEBUG: Channel %llu destroyed\n", ch->id);
}

// 旧的channel_close_impl实现已移除，现在通过适配层调用新的聚合根方法
// 以下代码保留作为参考，但不再执行
#if 0
void channel_close_impl_old(Channel* ch) {
    pthread_mutex_lock(&ch->lock);
    ch->state = CHANNEL_CLOSED;

    // 唤醒所有等待的发送者协程
    Coroutine* sender = ch->sender_queue;
    while (sender) {
        Coroutine* next = sender->next;
        sender->next = NULL;

        if (sender->task) {
            Task* task = (Task*)sender->task;  // 类型转换
            // TODO: 阶段4需要更新为使用新的Task聚合根方法
            pthread_mutex_lock(&task->mutex);  // 旧的方式，阶段4更新
            if (task->status == TASK_WAITING) {  // 旧的方式，阶段4更新
                task->status = TASK_READY;  // 旧的方式，阶段4更新
                TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
                printf("DEBUG: Waking sender task %llu (coroutine %llu) due to channel close\n",
                       task_id, sender->id);
                pthread_cond_signal(&task->cond);  // 旧的方式，阶段4更新
            }
            pthread_mutex_unlock(&task->mutex);  // 旧的方式，阶段4更新
        }

        sender->state = COROUTINE_READY;
        printf("DEBUG: Set sender coroutine %llu state to READY due to channel close\n", sender->id);

        // 将发送者任务重新加入调度队列
        if (sender->task) {
            extern Scheduler* get_global_scheduler(void);
            Scheduler* scheduler = get_global_scheduler();
            if (scheduler) {
                Processor* processor = sender->bound_processor;
                if (!processor && scheduler->num_processors > 0) {
                    processor = scheduler->processors[0];
                }
                
                if (processor) {
                    extern bool processor_push_local(Processor* processor, Task* task);
                    processor_push_local(processor, sender->task);
                }
            }
        }

        sender = next;
    }
    ch->sender_queue = NULL;

    // 唤醒所有等待的接收者协程
    Coroutine* receiver = ch->receiver_queue;
    while (receiver) {
        Coroutine* next = receiver->next;
        receiver->next = NULL;

        if (receiver->task) {
            Task* task = (Task*)receiver->task;  // 类型转换
            // TODO: 阶段4需要更新为使用新的Task聚合根方法
            pthread_mutex_lock(&task->mutex);  // 旧的方式，阶段4更新
            if (task->status == TASK_WAITING) {  // 旧的方式，阶段4更新
                task->status = TASK_READY;  // 旧的方式，阶段4更新
                TaskID task_id = task_get_id(task);  // 使用新的聚合根方法
                printf("DEBUG: Waking receiver task %llu (coroutine %llu) due to channel close\n",
                       task_id, receiver->id);
                pthread_cond_signal(&task->cond);  // 旧的方式，阶段4更新
            }
            pthread_mutex_unlock(&task->mutex);  // 旧的方式，阶段4更新
        }

        receiver->state = COROUTINE_READY;
        printf("DEBUG: Set receiver coroutine %llu state to READY due to channel close\n", receiver->id);

        // 将接收者任务重新加入调度队列
        if (receiver->task) {
            extern Scheduler* get_global_scheduler(void);
            Scheduler* scheduler = get_global_scheduler();
            if (scheduler) {
                Processor* processor = receiver->bound_processor;
                if (!processor && scheduler->num_processors > 0) {
                    processor = scheduler->processors[0];
                }
                
                if (processor) {
                    extern bool processor_push_local(Processor* processor, Task* task);
                    processor_push_local(processor, receiver->task);
                }
            }
        }

        receiver = next;
    }
    ch->receiver_queue = NULL;

    // 广播条件变量唤醒所有等待的线程
    pthread_cond_broadcast(&ch->send_cond);
    pthread_cond_broadcast(&ch->recv_cond);

    pthread_mutex_unlock(&ch->lock);

    printf("DEBUG: Channel %llu closed, all waiting coroutines woken up\n", ch->id);
}
#endif  // 关闭 channel_close_impl_old 的 #if 0 块（第804行开始）

// 获取通道中的消息数量
uint32_t channel_get_message_count(Channel* channel) {
    if (!channel) return 0;
    return channel->size;
}

// 尝试非阻塞发送
int channel_try_send(Channel* channel, void* value) {
    if (!channel || !value) return -1;

    int result = pthread_mutex_trylock(&channel->lock);
    if (result != 0) return result; // 无法获取锁

    if (channel->state == CHANNEL_CLOSED) {
        pthread_mutex_unlock(&channel->lock);
        return -2; // 通道已关闭
    }

    if (channel->capacity == 0) {
        // 无缓冲通道：只有当有接收者在等待时才能发送
        if (channel->receiver_queue) {
            // 有接收者在等待，直接发送
            channel->size++;
            // TODO: 唤醒接收协程并传递值
            pthread_mutex_unlock(&channel->lock);
            return 0;
        } else {
            pthread_mutex_unlock(&channel->lock);
            return -3; // 没有接收者，无法发送
        }
    } else {
        // 有缓冲通道：检查缓冲区是否已满
        if (channel->size >= channel->capacity) {
            pthread_mutex_unlock(&channel->lock);
            return -4; // 缓冲区已满
        }
        // TODO: 实现缓冲区发送逻辑
        channel->size++;
        pthread_mutex_unlock(&channel->lock);
        return 0;
    }
}

// 尝试非阻塞接收
void* channel_try_receive(Channel* channel) {
    if (!channel) return NULL;

    int result = pthread_mutex_trylock(&channel->lock);
    if (result != 0) return NULL; // 无法获取锁

    if (channel->state == CHANNEL_CLOSED && channel->size == 0) {
        pthread_mutex_unlock(&channel->lock);
        return NULL; // 通道已关闭且无消息
    }

    if (channel->capacity == 0) {
        // 无缓冲通道：只有当有发送者在等待时才能接收
        if (channel->sender_queue) {
            // 有发送者在等待，直接接收
            channel->size--;
            // TODO: 从发送协程获取值
            void* value = (void*)0x12345678; // 临时占位符
            pthread_mutex_unlock(&channel->lock);
            return value;
        } else {
            pthread_mutex_unlock(&channel->lock);
            return NULL; // 没有发送者，无法接收
        }
    } else {
        // 有缓冲通道：检查缓冲区是否有消息
        if (channel->size == 0) {
            pthread_mutex_unlock(&channel->lock);
            return NULL; // 缓冲区为空
        }
        // TODO: 实现缓冲区接收逻辑
        channel->size--;
        void* value = (void*)0x87654321; // 临时占位符
        pthread_mutex_unlock(&channel->lock);
        return value;
    }
}
