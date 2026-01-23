#include "coroutine_dtos.h"
#include <stdlib.h>
#include <string.h>
#include <time.h>

// 协程DTO构造函数
CoroutineDTO* coroutine_dto_create(uint64_t coroutine_id, const char* name, const char* status) {
    CoroutineDTO* dto = (CoroutineDTO*)malloc(sizeof(CoroutineDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(CoroutineDTO));
    dto->coroutine_id = coroutine_id;

    if (name) {
        strncpy(dto->name, name, sizeof(dto->name) - 1);
    }

    if (status) {
        strncpy(dto->status, status, sizeof(dto->status) - 1);
    }

    return dto;
}

// 协程DTO销毁函数
void coroutine_dto_destroy(CoroutineDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 协程列表DTO构造函数
CoroutineListDTO* coroutine_list_dto_create(size_t count) {
    CoroutineListDTO* dto = (CoroutineListDTO*)malloc(sizeof(CoroutineListDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(CoroutineListDTO));
    dto->count = count;
    dto->total_count = count;

    if (count > 0) {
        dto->coroutines = (CoroutineDTO*)malloc(sizeof(CoroutineDTO) * count);
        if (dto->coroutines) {
            memset(dto->coroutines, 0, sizeof(CoroutineDTO) * count);
        } else {
            free(dto);
            return NULL;
        }
    }

    return dto;
}

// 协程列表DTO销毁函数
void coroutine_list_dto_destroy(CoroutineListDTO* dto) {
    if (dto) {
        if (dto->coroutines) {
            free(dto->coroutines);
        }
        free(dto);
    }
}

// 协程统计DTO构造函数
CoroutineStatisticsDTO* coroutine_statistics_dto_create(void) {
    CoroutineStatisticsDTO* dto = (CoroutineStatisticsDTO*)malloc(sizeof(CoroutineStatisticsDTO));
    if (!dto) return NULL;

    memset(dto, 0, sizeof(CoroutineStatisticsDTO));
    dto->last_updated = time(NULL);

    return dto;
}

// 协程统计DTO销毁函数
void coroutine_statistics_dto_destroy(CoroutineStatisticsDTO* dto) {
    if (dto) {
        free(dto);
    }
}
