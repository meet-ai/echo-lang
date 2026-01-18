#include "coroutine.h"
#include "context.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// 全局协程ID计数器
static uint64_t next_coroutine_id = 1;
static pthread_mutex_t coroutine_id_mutex = PTHREAD_MUTEX_INITIALIZER;

// 当前正在执行的协程
__thread Coroutine* current_coroutine = NULL;

// 生成协程ID
uint64_t coroutine_generate_id(void) {
    pthread_mutex_lock(&coroutine_id_mutex);
    uint64_t id = next_coroutine_id++;
    pthread_mutex_unlock(&coroutine_id_mutex);
    return id;
}

// 协程包装函数
void coroutine_wrapper(Coroutine* coroutine) {
    printf("DEBUG: coroutine_wrapper ENTERED with coroutine %llu\n", coroutine->id);

    // 设置当前协程
    current_coroutine = coroutine;

    if (coroutine->entry_point) {
        printf("DEBUG: Calling entry_point for coroutine %llu\n", coroutine->id);
        coroutine->entry_point(coroutine->arg);
        printf("DEBUG: entry_point returned for coroutine %llu\n", coroutine->id);
    }

    // 协程执行完成
    coroutine->state = COROUTINE_COMPLETED;
    printf("DEBUG: Coroutine %llu completed\n", coroutine->id);

    // 这里可以唤醒等待的Future或任务
    // 暂时留空，等待具体实现
}

// 创建协程
Coroutine* coroutine_create(void (*entry_point)(void*), void* arg, size_t stack_size) {
    Coroutine* coroutine = (Coroutine*)malloc(sizeof(Coroutine));
    if (!coroutine) {
        fprintf(stderr, "ERROR: Failed to allocate memory for coroutine\n");
        return NULL;
    }

    memset(coroutine, 0, sizeof(Coroutine));

    // 初始化协程属性
    coroutine->id = coroutine_generate_id();
    coroutine->state = COROUTINE_READY;
    coroutine->entry_point = entry_point;
    coroutine->arg = arg;
    coroutine->stack_size = stack_size;
    coroutine->bound_processor = NULL;
    coroutine->task = NULL;
    coroutine->next = NULL;

    // 分配栈内存
    if (stack_size > 0) {
        coroutine->stack = (char*)malloc(stack_size);
        if (!coroutine->stack) {
            fprintf(stderr, "ERROR: Failed to allocate stack for coroutine\n");
            free(coroutine);
            return NULL;
        }
    }

    // 初始化上下文
    coro_context_init(&coroutine->context, coroutine->stack, stack_size,
                     (void (*)(void*))coroutine_wrapper, coroutine);

    // 初始化同步机制
    pthread_mutex_init(&coroutine->mutex, NULL);

    printf("DEBUG: Created coroutine %llu with stack size %zu\n", coroutine->id, stack_size);
    return coroutine;
}

// 销毁协程
void coroutine_destroy(Coroutine* coroutine) {
    if (!coroutine) return;

    // 清理栈内存
    if (coroutine->stack) {
        free(coroutine->stack);
        coroutine->stack = NULL;
    }

    // 清理同步机制
    pthread_mutex_destroy(&coroutine->mutex);

    printf("DEBUG: Destroyed coroutine %llu\n", coroutine->id);
    free(coroutine);
}

// 恢复协程执行
void coroutine_resume(Coroutine* coroutine) {
    if (!coroutine) return;

    pthread_mutex_lock(&coroutine->mutex);

    if (coroutine->state == COROUTINE_READY || coroutine->state == COROUTINE_SUSPENDED) {
        coroutine->state = COROUTINE_RUNNING;
        printf("DEBUG: Resuming coroutine %llu\n", coroutine->id);

        // 如果有绑定的处理器，这里可以进行上下文切换
        // 暂时直接调用entry_point（简化实现）
        if (coroutine->entry_point) {
            coroutine->entry_point(coroutine->arg);
            coroutine->state = COROUTINE_COMPLETED;
        }
    }

    pthread_mutex_unlock(&coroutine->mutex);
}

// 挂起协程执行
void coroutine_suspend(Coroutine* coroutine) {
    if (!coroutine) return;

    pthread_mutex_lock(&coroutine->mutex);

    if (coroutine->state == COROUTINE_RUNNING) {
        coroutine->state = COROUTINE_SUSPENDED;
        printf("DEBUG: Suspended coroutine %llu\n", coroutine->id);
    }

    pthread_mutex_unlock(&coroutine->mutex);
}

// 检查协程是否完成
bool coroutine_is_complete(Coroutine* coroutine) {
    if (!coroutine) return true;

    pthread_mutex_lock(&coroutine->mutex);
    bool complete = (coroutine->state == COROUTINE_COMPLETED || coroutine->state == COROUTINE_CANCELLED);
    pthread_mutex_unlock(&coroutine->mutex);

    return complete;
}
