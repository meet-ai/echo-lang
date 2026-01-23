#include "trigger_gc_command.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "../../domain/aggregates/garbage_collector.h"
#include "../../domain/aggregates/heap.h"
#include "../../domain/services/mark_service.h"
#include "../../domain/services/sweep_service.h"

// 创建命令处理器
echo_gc_command_handler_t* echo_gc_command_handler_create(
    struct echo_gc_collector* collector,
    struct echo_gc_heap* heap,
    struct echo_gc_mark_service* mark_service,
    struct echo_gc_sweep_service* sweep_service
) {
    echo_gc_command_handler_t* handler = calloc(1, sizeof(echo_gc_command_handler_t));
    if (!handler) {
        return NULL;
    }

    handler->collector = collector;
    handler->heap = heap;
    handler->mark_service = mark_service;
    handler->sweep_service = sweep_service;

    return handler;
}

// 销毁命令处理器
void echo_gc_command_handler_destroy(echo_gc_command_handler_t* handler) {
    if (handler) {
        free(handler);
    }
}

// 处理GC触发命令
echo_gc_error_t echo_gc_trigger_gc_command_handle(
    echo_gc_command_handler_t* handler,
    const echo_gc_trigger_command_t* command
) {
    if (!handler || !command) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 验证命令
    if (!echo_gc_trigger_command_validate(command)) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 检查GC是否可以触发
    if (!command->force_gc && !echo_gc_collector_can_trigger_gc(handler->collector)) {
        return ECHO_GC_ERROR_SYSTEM_ERROR;
    }

    // 准备GC
    echo_gc_error_t err = echo_gc_heap_prepare_for_gc(handler->heap);
    if (err != ECHO_GC_SUCCESS) {
        return err;
    }

    // 触发GC
    err = echo_gc_collector_trigger_gc(handler->collector, command->reason);
    if (err != ECHO_GC_SUCCESS) {
        echo_gc_heap_complete_gc(handler->heap);  // 清理
        return err;
    }

    // 开始标记阶段
    err = echo_gc_collector_start_mark_phase(handler->collector);
    if (err != ECHO_GC_SUCCESS) {
        echo_gc_heap_complete_gc(handler->heap);
        return err;
    }

    // 执行并发标记
    err = echo_gc_mark_service_start_concurrent_mark(handler->mark_service);
    if (err != ECHO_GC_SUCCESS) {
        echo_gc_heap_complete_gc(handler->heap);
        return err;
    }

    return ECHO_GC_SUCCESS;
}

// 处理GC完成命令
echo_gc_error_t echo_gc_complete_gc_command_handle(
    echo_gc_command_handler_t* handler,
    const echo_gc_complete_command_t* command
) {
    if (!handler || !command) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    // 验证命令
    if (!echo_gc_complete_command_validate(command)) {
        return ECHO_GC_ERROR_INVALID_CONFIG;
    }

    echo_gc_error_t err;

    if (command->success) {
        // 完成标记阶段
        uint64_t marked_objects, processed_bytes;
        err = echo_gc_mark_service_wait_for_completion(
            handler->mark_service, &marked_objects, &processed_bytes);
        if (err != ECHO_GC_SUCCESS) {
            goto cleanup;
        }

        err = echo_gc_collector_complete_mark_phase(handler->collector, marked_objects, processed_bytes);
        if (err != ECHO_GC_SUCCESS) {
            goto cleanup;
        }

        // 开始清扫阶段
        err = echo_gc_collector_start_sweep_phase(handler->collector);
        if (err != ECHO_GC_SUCCESS) {
            goto cleanup;
        }

        // 执行清扫
        uint64_t objects_collected, memory_reclaimed;
        err = echo_gc_sweep_service_perform_sweep(
            handler->sweep_service, &objects_collected, &memory_reclaimed);
        if (err != ECHO_GC_SUCCESS) {
            goto cleanup;
        }

        // 完成GC周期
        err = echo_gc_collector_complete_gc(handler->collector, &command->metrics);
        if (err != ECHO_GC_SUCCESS) {
            goto cleanup;
        }

    } else {
        // GC失败，记录错误
        printf("GC failed: %s\n", command->error_message);
        err = ECHO_GC_ERROR_SYSTEM_ERROR;
    }

cleanup:
    // 完成GC（无论成功失败）
    echo_gc_heap_complete_gc(handler->heap);

    return err;
}

// 命令验证
bool echo_gc_trigger_command_validate(const echo_gc_trigger_command_t* command) {
    if (!command) return false;

    // 验证触发原因
    switch (command->reason) {
        case ECHO_GC_TRIGGER_HEAP_USAGE:
        case ECHO_GC_TRIGGER_ALLOCATION_RATE:
        case ECHO_GC_TRIGGER_MANUAL:
        case ECHO_GC_TRIGGER_EMERGENCY:
            break;
        default:
            return false;
    }

    // 验证超时时间
    if (command->timeout_ms == 0) {
        return false;
    }

    return true;
}

bool echo_gc_complete_command_validate(const echo_gc_complete_command_t* command) {
    if (!command) return false;

    // 验证指标数据
    if (command->metrics.collected_at.tv_sec == 0) {
        return false;
    }

    return true;
}
