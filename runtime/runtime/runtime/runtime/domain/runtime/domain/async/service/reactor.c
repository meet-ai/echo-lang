/**
 * @file reactor.c
 * @brief Reactor 聚合根实现
 *
 * Reactor 负责监听I/O事件并通过Waker唤醒任务。
 */

#include "reactor.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// 简单的Waker注册表实现
typedef struct WakerRegistry {
    waker_t** wakers;
    size_t size;
    size_t capacity;
} waker_registry_t;

static waker_registry_t* waker_registry_create(size_t capacity) {
    waker_registry_t* registry = (waker_registry_t*)malloc(sizeof(waker_registry_t));
    if (!registry) return NULL;

    registry->wakers = (waker_t**)calloc(capacity, sizeof(waker_t*));
    if (!registry->wakers) {
        free(registry);
        return NULL;
    }

    registry->size = 0;
    registry->capacity = capacity;
    return registry;
}

static void waker_registry_destroy(waker_registry_t* registry) {
    if (registry) {
        free(registry->wakers);
        free(registry);
    }
}

static bool waker_registry_set(waker_registry_t* registry, int fd, waker_t* waker) {
    if (fd >= (int)registry->capacity) {
        return false; // fd超出范围
    }
    registry->wakers[fd] = waker;
    return true;
}

static waker_t* waker_registry_get(waker_registry_t* registry, int fd) {
    if (fd >= (int)registry->capacity) {
        return NULL;
    }
    return registry->wakers[fd];
}

// 简单的事件循环实现（临时）
typedef struct EventLoop {
    uint64_t id;
    // 简化实现，实际应该有完整的事件循环逻辑
} event_loop_t;

static event_loop_t* event_loop_create(uint64_t id) {
    event_loop_t* loop = (event_loop_t*)malloc(sizeof(event_loop_t));
    if (loop) {
        loop->id = id;
    }
    return loop;
}

static void event_loop_destroy(event_loop_t* loop) {
    free(loop);
}

// ============================================================================
// Reactor 聚合根实现
// ============================================================================

/**
 * @brief 创建反应器
 */
reactor_t* reactor_create(struct EventLoop* event_loop) {
    reactor_t* reactor = (reactor_t*)malloc(sizeof(reactor_t));
    if (!reactor) {
        fprintf(stderr, "Failed to allocate reactor\n");
        return NULL;
    }

    reactor->id = 1; // 简化ID分配
    reactor->event_loop = event_loop ? event_loop : event_loop_create(1);
    reactor->waker_registry = waker_registry_create(MAX_FD_COUNT);
    reactor->event_dispatcher = NULL; // 暂时未实现

    if (!reactor->waker_registry) {
        fprintf(stderr, "Failed to create waker registry\n");
        if (reactor->event_loop) event_loop_destroy(reactor->event_loop);
        free(reactor);
        return NULL;
    }

    printf("DEBUG: Created Reactor %llu\n", reactor->id);
    return reactor;
}

/**
 * @brief 销毁反应器
 */
void reactor_destroy(reactor_t* reactor) {
    if (!reactor) return;

    if (reactor->waker_registry) {
        waker_registry_destroy((waker_registry_t*)reactor->waker_registry);
    }

    if (reactor->event_loop) {
        event_loop_destroy(reactor->event_loop);
    }

    printf("DEBUG: Destroyed Reactor %llu\n", reactor->id);
    free(reactor);
}

/**
 * @brief 注册Future到反应器
 * 核心业务逻辑：建立Future与Waker的关联
 */
void reactor_register_future(reactor_t* reactor, struct Future* future, waker_t* waker) {
    if (!reactor || !future || !waker) return;

    // 简化实现：假设Future有fd信息
    // 实际实现中需要从Future提取fd
    int fd = (int)future_get_id(future) % MAX_FD_COUNT; // 临时映射

    waker_registry_t* registry = (waker_registry_t*)reactor->waker_registry;
    if (waker_registry_set(registry, fd, waker)) {
        printf("DEBUG: Reactor registered future %llu with waker for fd %d\n",
               future_get_id(future), fd);
    } else {
        fprintf(stderr, "ERROR: Failed to register future, fd out of range\n");
    }
}

/**
 * @brief 取消注册Future
 */
void reactor_unregister_future(reactor_t* reactor, struct Future* future) {
    if (!reactor || !future) return;

    int fd = (int)future_get_id(future) % MAX_FD_COUNT; // 临时映射

    waker_registry_t* registry = (waker_registry_t*)reactor->waker_registry;
    if (waker_registry_set(registry, fd, NULL)) {
        printf("DEBUG: Reactor unregistered future %llu from fd %d\n",
               future_get_id(future), fd);
    }
}

/**
 * @brief 轮询事件
 * 核心业务逻辑：监听事件并唤醒相应任务
 */
int reactor_poll_events(reactor_t* reactor, timeout_t timeout) {
    if (!reactor) return -1;

    // 简化实现：暂时模拟一个事件
    // 实际实现应该调用真正的I/O事件循环

    static int call_count = 0;
    call_count++;

    // 每隔几次调用就模拟一个事件
    if (call_count % 5 == 0) {
        // 模拟fd=1有事件
        int simulated_fd = 1;
        waker_registry_t* registry = (waker_registry_t*)reactor->waker_registry;
        waker_t* waker = waker_registry_get(registry, simulated_fd);

        if (waker) {
            printf("DEBUG: Reactor detected event on fd %d, waking task\n", simulated_fd);
            waker_wake(waker);
            return 1; // 返回事件数量
        }
    }

    // 无事件
    return 0;
}

/**
 * @brief 唤醒反应器
 */
void reactor_wakeup(reactor_t* reactor) {
    // 简化实现，暂时无操作
    // 实际实现应该有跨线程唤醒机制
    (void)reactor;
    printf("DEBUG: Reactor wakeup called (not implemented yet)\n");
}

