#ifndef CHANNEL_APPLICATION_SERVICE_H
#define CHANNEL_APPLICATION_SERVICE_H

#include "../commands/channel_commands.h"
#include "../dtos/result_dtos.h"
#include "../dtos/channel_dtos.h"
#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct ChannelApplicationService;
typedef struct ChannelApplicationService ChannelApplicationService;

// 通道应用服务接口 - 负责通道相关的用例编排
typedef struct {
    // 通道管理用例
    ChannelCreationResultDTO* (*create_channel)(ChannelApplicationService* service, const CreateChannelCommand* cmd);
    OperationResultDTO* (*close_channel)(ChannelApplicationService* service, const CloseChannelCommand* cmd);

    // 通道通信用例
    SendResultDTO* (*send_message)(ChannelApplicationService* service, const SendMessageCommand* cmd);
    ReceiveResultDTO* (*receive_message)(ChannelApplicationService* service, const ReceiveMessageCommand* cmd);

    // 通道查询用例
    ChannelDTO* (*get_channel)(ChannelApplicationService* service, uint64_t channel_id);
    ChannelListDTO* (*get_channels)(ChannelApplicationService* service, const GetChannelStatusQuery* query);
    ChannelStatisticsDTO* (*get_channel_statistics)(ChannelApplicationService* service, const GetChannelStatusQuery* query);

    // 通道控制用例
    OperationResultDTO* (*set_channel_capacity)(ChannelApplicationService* service, const SetChannelCapacityCommand* cmd);
    OperationResultDTO* (*select_channels)(ChannelApplicationService* service, const SelectChannelsCommand* cmd);
} ChannelApplicationServiceInterface;

// 通道应用服务实现
struct ChannelApplicationService {
    ChannelApplicationServiceInterface* vtable;  // 虚函数表

    // 依赖的领域服务
    void* channel_manager;       // 通道管理器领域服务
    void* event_publisher;       // 事件发布器

    // 应用服务状态
    bool initialized;
    char service_name[256];
    time_t started_at;
};

// 构造函数和析构函数
ChannelApplicationService* channel_application_service_create(void);
void channel_application_service_destroy(ChannelApplicationService* service);

// 初始化和清理
bool channel_application_service_initialize(ChannelApplicationService* service,
                                          void* channel_manager,
                                          void* event_publisher);
void channel_application_service_cleanup(ChannelApplicationService* service);

// 服务状态查询
bool channel_application_service_is_healthy(const ChannelApplicationService* service);
const char* channel_application_service_get_name(const ChannelApplicationService* service);
time_t channel_application_service_get_started_at(const ChannelApplicationService* service);

// 便捷函数（直接调用虚函数表）
static inline ChannelCreationResultDTO* channel_application_service_create_channel(
    ChannelApplicationService* service, const CreateChannelCommand* cmd) {
    return service->vtable->create_channel(service, cmd);
}

static inline OperationResultDTO* channel_application_service_close_channel(
    ChannelApplicationService* service, const CloseChannelCommand* cmd) {
    return service->vtable->close_channel(service, cmd);
}

static inline SendResultDTO* channel_application_service_send_message(
    ChannelApplicationService* service, const SendMessageCommand* cmd) {
    return service->vtable->send_message(service, cmd);
}

static inline ReceiveResultDTO* channel_application_service_receive_message(
    ChannelApplicationService* service, const ReceiveMessageCommand* cmd) {
    return service->vtable->receive_message(service, cmd);
}

#endif // CHANNEL_APPLICATION_SERVICE_H
