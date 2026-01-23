#include "async_dtos.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// FutureDTO 创建函数
FutureDTO* future_dto_create(uint64_t id, const char* state, void* result_data, size_t result_size, const char* error_message, time_t created_at, time_t resolved_at, time_t rejected_at, uint64_t execution_time_us, uint32_t poll_count) {
    FutureDTO* dto = (FutureDTO*)malloc(sizeof(FutureDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureDTO\n");
        return NULL;
    }

    dto->id = id;
    if (state) {
        strncpy(dto->state, state, sizeof(dto->state) - 1);
        dto->state[sizeof(dto->state) - 1] = '\0';
    } else {
        dto->state[0] = '\0';
    }

    if (result_data && result_size > 0) {
        dto->result_data = malloc(result_size);
        if (dto->result_data) {
            memcpy(dto->result_data, result_data, result_size);
            dto->result_size = result_size;
        } else {
            dto->result_data = NULL;
            dto->result_size = 0;
        }
    } else {
        dto->result_data = NULL;
        dto->result_size = 0;
    }

    if (error_message) {
        strncpy(dto->error_message, error_message, sizeof(dto->error_message) - 1);
        dto->error_message[sizeof(dto->error_message) - 1] = '\0';
    } else {
        dto->error_message[0] = '\0';
    }

    dto->created_at = created_at;
    dto->resolved_at = resolved_at;
    dto->rejected_at = rejected_at;
    dto->execution_time_us = execution_time_us;
    dto->poll_count = poll_count;

    return dto;
}

// FutureDTO 销毁函数
void future_dto_destroy(FutureDTO* dto) {
    if (dto) {
        if (dto->result_data) {
            free(dto->result_data);
        }
        free(dto);
    }
}

// FutureCreationResultDTO 创建函数
FutureCreationResultDTO* future_creation_result_dto_create(bool success, uint64_t future_id, const char* message, int error_code) {
    FutureCreationResultDTO* dto = (FutureCreationResultDTO*)malloc(sizeof(FutureCreationResultDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureCreationResultDTO\n");
        return NULL;
    }

    dto->success = success;
    dto->future_id = future_id;
    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
        dto->message[sizeof(dto->message) - 1] = '\0';
    } else {
        dto->message[0] = '\0';
    }
    dto->error_code = error_code;

    return dto;
}

// FutureCreationResultDTO 销毁函数
void future_creation_result_dto_destroy(FutureCreationResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 注意：AsyncOperationResultDTO 的创建和销毁函数已在 result_dtos.c 中实现

// FutureListDTO 创建函数
FutureListDTO* future_list_dto_create(FutureDTO* futures, size_t count, size_t total_count, uint32_t page, uint32_t page_size) {
    FutureListDTO* dto = (FutureListDTO*)malloc(sizeof(FutureListDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureListDTO\n");
        return NULL;
    }

    if (futures && count > 0) {
        dto->futures = (FutureDTO*)malloc(sizeof(FutureDTO) * count);
        if (dto->futures) {
            memcpy(dto->futures, futures, sizeof(FutureDTO) * count);
            dto->count = count;
        } else {
            dto->futures = NULL;
            dto->count = 0;
        }
    } else {
        dto->futures = NULL;
        dto->count = 0;
    }

    dto->total_count = total_count;
    dto->page = page;
    dto->page_size = page_size;

    return dto;
}

// FutureListDTO 销毁函数
void future_list_dto_destroy(FutureListDTO* dto) {
    if (dto) {
        if (dto->futures) {
            free(dto->futures);
        }
        free(dto);
    }
}

// FutureStatisticsDTO 创建函数
FutureStatisticsDTO* future_statistics_dto_create(uint32_t total, uint32_t pending, uint32_t resolved, uint32_t rejected, uint32_t cancelled, double avg_exec_time, uint64_t total_exec_time, time_t last_updated) {
    FutureStatisticsDTO* dto = (FutureStatisticsDTO*)malloc(sizeof(FutureStatisticsDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for FutureStatisticsDTO\n");
        return NULL;
    }

    dto->total_futures = total;
    dto->pending_futures = pending;
    dto->resolved_futures = resolved;
    dto->rejected_futures = rejected;
    dto->cancelled_futures = cancelled;
    dto->average_execution_time_us = avg_exec_time;
    dto->total_execution_time_us = total_exec_time;
    dto->last_updated = last_updated;

    return dto;
}

// FutureStatisticsDTO 销毁函数
void future_statistics_dto_destroy(FutureStatisticsDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 转换函数实现（简化版本，实际实现需要根据Future实体来转换）
FutureDTO* future_to_dto(const struct Future* future) {
    // 这里需要根据实际的Future实体实现
    // 暂时返回一个空的DTO作为占位符
    return future_dto_create(0, "PENDING", NULL, 0, NULL, time(NULL), 0, 0, 0, 0);
}

FutureListDTO* futures_to_list_dto(struct Future** futures, size_t count, size_t total, uint32_t page, uint32_t page_size) {
    // 这里需要根据实际的Future实体数组实现
    // 暂时返回一个空的列表作为占位符
    return future_list_dto_create(NULL, 0, total, page, page_size);
}

FutureStatisticsDTO* calculate_future_statistics(struct Future** futures, size_t count) {
    // 这里需要根据实际的Future实体数组计算统计信息
    // 暂时返回一个空的统计作为占位符
    return future_statistics_dto_create(0, 0, 0, 0, 0, 0.0, 0, time(NULL));
}
