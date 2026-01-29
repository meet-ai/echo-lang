#include "status_dtos.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <time.h>

// ==================== RuntimeStatusDTO ====================

RuntimeStatusDTO* runtime_status_dto_create(void) {
    RuntimeStatusDTO* dto = (RuntimeStatusDTO*)malloc(sizeof(RuntimeStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(RuntimeStatusDTO));
    strncpy(dto->version, "1.0.0", sizeof(dto->version) - 1);
    strncpy(dto->status, "running", sizeof(dto->status) - 1);
    dto->started_at = time(NULL);
    dto->last_updated = time(NULL);

    return dto;
}

void runtime_status_dto_destroy(RuntimeStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== TaskStatusDTO ====================

TaskStatusDTO* task_status_dto_create(uint64_t task_id, const char* name) {
    TaskStatusDTO* dto = (TaskStatusDTO*)malloc(sizeof(TaskStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(TaskStatusDTO));
    dto->task_id = task_id;
    if (name) {
        strncpy(dto->name, name, sizeof(dto->name) - 1);
    }
    strncpy(dto->status, "pending", sizeof(dto->status) - 1);
    dto->created_at = time(NULL);

    return dto;
}

void task_status_dto_destroy(TaskStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== CoroutineStatusDTO ====================

CoroutineStatusDTO* coroutine_status_dto_create(uint64_t coroutine_id, const char* name) {
    CoroutineStatusDTO* dto = (CoroutineStatusDTO*)malloc(sizeof(CoroutineStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(CoroutineStatusDTO));
    dto->coroutine_id = coroutine_id;
    if (name) {
        strncpy(dto->name, name, sizeof(dto->name) - 1);
    }
    strncpy(dto->status, "pending", sizeof(dto->status) - 1);
    dto->created_at = time(NULL);

    return dto;
}

void coroutine_status_dto_destroy(CoroutineStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== ChannelStatusDTO ====================

ChannelStatusDTO* channel_status_dto_create(uint64_t channel_id) {
    ChannelStatusDTO* dto = (ChannelStatusDTO*)malloc(sizeof(ChannelStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(ChannelStatusDTO));
    dto->channel_id = channel_id;
    strncpy(dto->status, "open", sizeof(dto->status) - 1);
    dto->created_at = time(NULL);

    return dto;
}

void channel_status_dto_destroy(ChannelStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== FutureStatusDTO ====================

FutureStatusDTO* future_status_dto_create(uint64_t future_id) {
    FutureStatusDTO* dto = (FutureStatusDTO*)malloc(sizeof(FutureStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(FutureStatusDTO));
    dto->future_id = future_id;
    strncpy(dto->status, "pending", sizeof(dto->status) - 1);
    dto->created_at = time(NULL);

    return dto;
}

void future_status_dto_destroy(FutureStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== GCStatusDTO ====================

GCStatusDTO* gc_status_dto_create(void) {
    GCStatusDTO* dto = (GCStatusDTO*)malloc(sizeof(GCStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(GCStatusDTO));
    strncpy(dto->status, "idle", sizeof(dto->status) - 1);
    strncpy(dto->algorithm, "mark-sweep", sizeof(dto->algorithm) - 1);

    return dto;
}

void gc_status_dto_destroy(GCStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== ConcurrencyStatusDTO ====================

ConcurrencyStatusDTO* concurrency_status_dto_create(void) {
    ConcurrencyStatusDTO* dto = (ConcurrencyStatusDTO*)malloc(sizeof(ConcurrencyStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(ConcurrencyStatusDTO));
    dto->last_updated = time(NULL);

    return dto;
}

void concurrency_status_dto_destroy(ConcurrencyStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== MemoryStatusDTO ====================

MemoryStatusDTO* memory_status_dto_create(void) {
    MemoryStatusDTO* dto = (MemoryStatusDTO*)malloc(sizeof(MemoryStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(MemoryStatusDTO));

    return dto;
}

void memory_status_dto_destroy(MemoryStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== SystemStatusDTO ====================

SystemStatusDTO* system_status_dto_create(void) {
    SystemStatusDTO* dto = (SystemStatusDTO*)malloc(sizeof(SystemStatusDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(SystemStatusDTO));
    dto->last_updated = time(NULL);

    return dto;
}

void system_status_dto_destroy(SystemStatusDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ==================== DTO序列化（简化实现）====================

char* runtime_status_dto_to_json(const RuntimeStatusDTO* dto) {
    if (!dto) return NULL;
    
    // 简化实现：返回基本JSON字符串
    char* json = (char*)malloc(512);
    if (!json) return NULL;
    
    snprintf(json, 512,
        "{\"version\":\"%s\",\"status\":\"%s\",\"uptime_seconds\":%llu}",
        dto->version, dto->status, (unsigned long long)dto->uptime_seconds);
    
    return json;
}

char* task_status_dto_to_json(const TaskStatusDTO* dto) {
    if (!dto) return NULL;
    
    char* json = (char*)malloc(512);
    if (!json) return NULL;
    
    snprintf(json, 512,
        "{\"task_id\":%llu,\"name\":\"%s\",\"status\":\"%s\"}",
        (unsigned long long)dto->task_id, dto->name, dto->status);
    
    return json;
}

char* gc_status_dto_to_json(const GCStatusDTO* dto) {
    if (!dto) return NULL;
    
    char* json = (char*)malloc(512);
    if (!json) return NULL;
    
    snprintf(json, 512,
        "{\"status\":\"%s\",\"algorithm\":\"%s\",\"heap_size\":%llu}",
        dto->status, dto->algorithm, (unsigned long long)dto->heap_size);
    
    return json;
}

// ==================== JSON反序列化（简化实现）====================

RuntimeStatusDTO* runtime_status_dto_from_json(const char* json) {
    // 简化实现：创建默认DTO
    (void)json; // 暂时未使用
    return runtime_status_dto_create();
}

TaskStatusDTO* task_status_dto_from_json(const char* json) {
    // 简化实现：创建默认DTO
    (void)json; // 暂时未使用
    return task_status_dto_create(0, "unknown");
}
