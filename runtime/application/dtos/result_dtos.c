#include "result_dtos.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

// 协程创建结果DTO构造函数
CoroutineCreationResultDTO* coroutine_creation_result_dto_create(bool success, uint64_t coroutine_id, const char* message) {
    CoroutineCreationResultDTO* dto = (CoroutineCreationResultDTO*)malloc(sizeof(CoroutineCreationResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(CoroutineCreationResultDTO));
    dto->success = success;
    dto->coroutine_id = coroutine_id;
    dto->created_at = time(NULL);

    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
    }

    return dto;
}

// 协程创建结果DTO销毁函数
void coroutine_creation_result_dto_destroy(CoroutineCreationResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 异步操作结果DTO构造函数
AsyncOperationResultDTO* async_operation_result_dto_create(uint64_t operation_id, const char* status) {
    AsyncOperationResultDTO* dto = (AsyncOperationResultDTO*)malloc(sizeof(AsyncOperationResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(AsyncOperationResultDTO));
    dto->operation_id = operation_id;
    dto->started_at = time(NULL);

    if (status) {
        strncpy(dto->status, status, sizeof(dto->status) - 1);
    }

    return dto;
}

// 异步操作结果DTO销毁函数
void async_operation_result_dto_destroy(AsyncOperationResultDTO* dto) {
    if (dto) {
        if (dto->result_data) {
            free(dto->result_data);
        }
        free(dto);
    }
}

// 批量操作结果DTO构造函数
BatchOperationResultDTO* batch_operation_result_dto_create(uint32_t total_ops) {
    BatchOperationResultDTO* dto = (BatchOperationResultDTO*)malloc(sizeof(BatchOperationResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(BatchOperationResultDTO));
    dto->total_operations = total_ops;
    dto->batch_started_at = time(NULL);

    if (total_ops > 0) {
        dto->results = (OperationResultDTO*)malloc(sizeof(OperationResultDTO) * total_ops);
        if (dto->results) {
            memset(dto->results, 0, sizeof(OperationResultDTO) * total_ops);
            dto->results_count = total_ops;
        }
    }

    return dto;
}

// 批量操作结果DTO销毁函数
void batch_operation_result_dto_destroy(BatchOperationResultDTO* dto) {
    if (dto) {
        if (dto->results) {
            free(dto->results);
        }
        free(dto);
    }
}

// 查询结果DTO构造函数
QueryResultDTO* query_result_dto_create(bool success, void* data, size_t data_size) {
    QueryResultDTO* dto = (QueryResultDTO*)malloc(sizeof(QueryResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(QueryResultDTO));
    dto->success = success;
    dto->data = data;
    dto->data_size = data_size;

    return dto;
}

// 查询结果DTO销毁函数
void query_result_dto_destroy(QueryResultDTO* dto) {
    if (dto) {
        if (dto->data) {
            free(dto->data);
        }
        free(dto);
    }
}

// 错误结果DTO构造函数
ErrorResultDTO* error_result_dto_create(int error_code, const char* error_message) {
    ErrorResultDTO* dto = (ErrorResultDTO*)malloc(sizeof(ErrorResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(ErrorResultDTO));
    dto->error_code = error_code;

    if (error_message) {
        strncpy(dto->error_message, error_message, sizeof(dto->error_message) - 1);
        dto->error_message[sizeof(dto->error_message) - 1] = '\0';
    }

    return dto;
}

// 错误结果DTO销毁函数
void error_result_dto_destroy(ErrorResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 成功结果DTO构造函数
SuccessResultDTO* success_result_dto_create(void* data, size_t data_size, const char* message) {
    SuccessResultDTO* dto = (SuccessResultDTO*)malloc(sizeof(SuccessResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(SuccessResultDTO));
    dto->data = data;
    dto->data_size = data_size;

    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
    }

    return dto;
}

// 成功结果DTO销毁函数
void success_result_dto_destroy(SuccessResultDTO* dto) {
    if (dto) {
        if (dto->data) {
            free(dto->data);
        }
        free(dto);
    }
}

// 验证结果DTO构造函数
ValidationResultDTO* validation_result_dto_create(bool valid) {
    ValidationResultDTO* dto = (ValidationResultDTO*)malloc(sizeof(ValidationResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(ValidationResultDTO));
    dto->valid = valid;

    return dto;
}

// 验证结果DTO销毁函数
void validation_result_dto_destroy(ValidationResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 操作结果DTO构造函数
OperationResultDTO* operation_result_dto_create(bool success, const char* message, int error_code) {
    OperationResultDTO* dto = (OperationResultDTO*)malloc(sizeof(OperationResultDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(OperationResultDTO));
    dto->success = success;
    dto->error_code = error_code;

    if (message) {
        strncpy(dto->error_details, message, sizeof(dto->error_details) - 1);
        dto->error_details[sizeof(dto->error_details) - 1] = '\0';
    }

    return dto;
}

// 操作结果DTO销毁函数
void operation_result_dto_destroy(OperationResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}
