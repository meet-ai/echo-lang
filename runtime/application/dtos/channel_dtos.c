#include "channel_dtos.h"
#include <stdlib.h>
#include <string.h>
#include <stdio.h>

// ChannelDTO 创建函数
ChannelDTO* channel_dto_create(uint64_t id, const char* name, const char* type, uint32_t capacity, uint32_t size, bool is_closed, time_t created_at) {
    ChannelDTO* dto = (ChannelDTO*)malloc(sizeof(ChannelDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for ChannelDTO\n");
        return NULL;
    }

    dto->id = id;
    strncpy(dto->name, name, sizeof(dto->name) - 1);
    dto->name[sizeof(dto->name) - 1] = '\0';
    strncpy(dto->type, type, sizeof(dto->type) - 1);
    dto->type[sizeof(dto->type) - 1] = '\0';
    dto->capacity = capacity;
    dto->size = size;
    dto->is_closed = is_closed;
    dto->created_at = created_at;

    return dto;
}

// ChannelDTO 销毁函数
void channel_dto_destroy(ChannelDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ChannelCreationResultDTO 创建函数
ChannelCreationResultDTO* channel_creation_result_dto_create(bool success, uint64_t channel_id, const char* message, int error_code) {
    ChannelCreationResultDTO* dto = (ChannelCreationResultDTO*)malloc(sizeof(ChannelCreationResultDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for ChannelCreationResultDTO\n");
        return NULL;
    }

    dto->success = success;
    dto->channel_id = channel_id;
    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
        dto->message[sizeof(dto->message) - 1] = '\0';
    } else {
        dto->message[0] = '\0';
    }
    dto->error_code = error_code;

    return dto;
}

// ChannelCreationResultDTO 销毁函数
void channel_creation_result_dto_destroy(ChannelCreationResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// SendResultDTO 创建函数
SendResultDTO* send_result_dto_create(bool success, const char* message, int error_code, uint32_t bytes_sent) {
    SendResultDTO* dto = (SendResultDTO*)malloc(sizeof(SendResultDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for SendResultDTO\n");
        return NULL;
    }

    dto->success = success;
    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
        dto->message[sizeof(dto->message) - 1] = '\0';
    } else {
        dto->message[0] = '\0';
    }
    dto->error_code = error_code;
    dto->bytes_sent = bytes_sent;

    return dto;
}

// SendResultDTO 销毁函数
void send_result_dto_destroy(SendResultDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// ReceiveResultDTO 创建函数
ReceiveResultDTO* receive_result_dto_create(bool success, void* data, size_t data_size, const char* message, int error_code, uint32_t bytes_received) {
    ReceiveResultDTO* dto = (ReceiveResultDTO*)malloc(sizeof(ReceiveResultDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for ReceiveResultDTO\n");
        return NULL;
    }

    dto->success = success;
    if (data && data_size > 0) {
        dto->data = malloc(data_size);
        if (dto->data) {
            memcpy(dto->data, data, data_size);
            dto->data_size = data_size;
        } else {
            dto->data = NULL;
            dto->data_size = 0;
        }
    } else {
        dto->data = NULL;
        dto->data_size = 0;
    }

    if (message) {
        strncpy(dto->message, message, sizeof(dto->message) - 1);
        dto->message[sizeof(dto->message) - 1] = '\0';
    } else {
        dto->message[0] = '\0';
    }
    dto->error_code = error_code;
    dto->bytes_received = bytes_received;

    return dto;
}

// ReceiveResultDTO 销毁函数
void receive_result_dto_destroy(ReceiveResultDTO* dto) {
    if (dto) {
        if (dto->data) {
            free(dto->data);
        }
        free(dto);
    }
}

// ChannelListDTO 创建函数
ChannelListDTO* channel_list_dto_create(ChannelDTO* channels, size_t count, size_t total_count, uint32_t page, uint32_t page_size) {
    ChannelListDTO* dto = (ChannelListDTO*)malloc(sizeof(ChannelListDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for ChannelListDTO\n");
        return NULL;
    }

    if (channels && count > 0) {
        dto->channels = (ChannelDTO*)malloc(sizeof(ChannelDTO) * count);
        if (dto->channels) {
            memcpy(dto->channels, channels, sizeof(ChannelDTO) * count);
            dto->count = count;
        } else {
            dto->channels = NULL;
            dto->count = 0;
        }
    } else {
        dto->channels = NULL;
        dto->count = 0;
    }

    dto->total_count = total_count;
    dto->page = page;
    dto->page_size = page_size;

    return dto;
}

// ChannelListDTO 销毁函数
void channel_list_dto_destroy(ChannelListDTO* dto) {
    if (dto) {
        if (dto->channels) {
            free(dto->channels);
        }
        free(dto);
    }
}

// ChannelStatisticsDTO 创建函数
ChannelStatisticsDTO* channel_statistics_dto_create(uint32_t total, uint32_t active, uint32_t closed, uint64_t sent_msgs, uint64_t recv_msgs, uint64_t sent_bytes, uint64_t recv_bytes, time_t last_updated) {
    ChannelStatisticsDTO* dto = (ChannelStatisticsDTO*)malloc(sizeof(ChannelStatisticsDTO));
    if (!dto) {
        fprintf(stderr, "ERROR: Failed to allocate memory for ChannelStatisticsDTO\n");
        return NULL;
    }

    dto->total_channels = total;
    dto->active_channels = active;
    dto->closed_channels = closed;
    dto->total_messages_sent = sent_msgs;
    dto->total_messages_received = recv_msgs;
    dto->total_bytes_sent = sent_bytes;
    dto->total_bytes_received = recv_bytes;
    dto->last_updated = last_updated;

    return dto;
}

// ChannelStatisticsDTO 销毁函数
void channel_statistics_dto_destroy(ChannelStatisticsDTO* dto) {
    if (dto) {
        free(dto);
    }
}

// 转换函数实现（简化版本，实际实现需要根据Channel实体来转换）
ChannelDTO* channel_to_dto(const struct Channel* channel) {
    // 这里需要根据实际的Channel实体实现
    // 暂时返回一个空的DTO作为占位符
    return channel_dto_create(0, "placeholder", "unbuffered", 0, 0, false, time(NULL));
}

ChannelListDTO* channels_to_list_dto(struct Channel** channels, size_t count, size_t total, uint32_t page, uint32_t page_size) {
    // 这里需要根据实际的Channel实体数组实现
    // 暂时返回一个空的列表作为占位符
    return channel_list_dto_create(NULL, 0, total, page, page_size);
}

ChannelStatisticsDTO* calculate_channel_statistics(struct Channel** channels, size_t count) {
    // 这里需要根据实际的Channel实体数组计算统计信息
    // 暂时返回一个空的统计作为占位符
    return channel_statistics_dto_create(0, 0, 0, 0, 0, 0, 0, time(NULL));
}
