#ifndef CHANNEL_COMMANDS_H
#define CHANNEL_COMMANDS_H

#include <stdint.h>
#include <stdbool.h>

// 前向声明
struct ChannelOptions;

// 创建通道命令
typedef struct {
    size_t element_size;            // 元素大小
    struct ChannelOptions* options; // 通道选项
    char name[256];                 // 通道名称（可选）
} CreateChannelCommand;

// 关闭通道命令
typedef struct {
    uint64_t channel_id;            // 通道ID
    bool force;                     // 是否强制关闭
} CloseChannelCommand;

// 发送消息命令
typedef struct {
    uint64_t channel_id;            // 通道ID
    void* message;                  // 消息数据
    size_t message_size;            // 消息大小
    uint32_t timeout_ms;            // 发送超时时间
    bool blocking;                  // 是否阻塞发送
} SendMessageCommand;

// 接收消息命令
typedef struct {
    uint64_t channel_id;            // 通道ID
    void* buffer;                   // 接收缓冲区
    size_t buffer_size;             // 缓冲区大小
    uint32_t timeout_ms;            // 接收超时时间
    bool blocking;                  // 是否阻塞接收
} ReceiveMessageCommand;

// 选择操作命令
typedef struct {
    uint64_t* channel_ids;          // 通道ID列表
    size_t channel_count;           // 通道数量
    void* send_data;                // 发送数据（发送操作时使用）
    void* receive_buffer;           // 接收缓冲区（接收操作时使用）
    size_t buffer_size;             // 缓冲区大小
    uint32_t timeout_ms;            // 操作超时时间
    bool is_send_operation;         // 是否为发送操作
    size_t* selected_index;         // 选中的通道索引（输出）
} SelectOperationCommand;

// 获取通道状态命令
typedef struct {
    uint64_t channel_id;            // 通道ID
} GetChannelStatusCommand;

// 设置通道选项命令
typedef struct {
    uint64_t channel_id;            // 通道ID
    struct ChannelOptions* options; // 新选项
} SetChannelOptionsCommand;

// 设置通道容量命令
typedef struct {
    uint64_t channel_id;            // 通道ID
    uint32_t capacity;              // 新容量
} SetChannelCapacityCommand;

// 选择通道命令
typedef struct {
    uint64_t* channel_ids;          // 通道ID列表
    size_t channel_count;           // 通道数量
    uint32_t timeout_ms;            // 选择超时时间
} SelectChannelsCommand;

// 广播消息命令
typedef struct {
    uint64_t* channel_ids;          // 目标通道ID列表
    size_t channel_count;           // 通道数量
    void* message;                  // 广播消息
    size_t message_size;            // 消息大小
    bool async;                     // 是否异步广播
} BroadcastMessageCommand;

// 通道选项（重复定义以避免循环依赖）
struct ChannelOptions {
    size_t buffer_size;             // 缓冲区大小
    bool blocking;                  // 是否阻塞模式
    uint32_t timeout_ms;            // 超时时间
    void* user_data;                // 用户数据
};

// 默认通道选项
extern const struct ChannelOptions CHANNEL_DEFAULT_OPTIONS;

// 命令验证函数
bool validate_create_channel_command(const CreateChannelCommand* cmd);
bool validate_close_channel_command(const CloseChannelCommand* cmd);
bool validate_send_message_command(const SendMessageCommand* cmd);
bool validate_receive_message_command(const ReceiveMessageCommand* cmd);
bool validate_select_operation_command(const SelectOperationCommand* cmd);
bool validate_get_status_command(const GetChannelStatusCommand* cmd);
bool validate_set_options_command(const SetChannelOptionsCommand* cmd);
bool validate_broadcast_command(const BroadcastMessageCommand* cmd);

// 命令构建函数
CreateChannelCommand* create_channel_command_create(size_t element_size, const struct ChannelOptions* options);
void create_channel_command_destroy(CreateChannelCommand* cmd);

SelectOperationCommand* select_operation_command_create(
    uint64_t* channel_ids,
    size_t channel_count,
    bool is_send_operation,
    uint32_t timeout_ms
);
void select_operation_command_destroy(SelectOperationCommand* cmd);

BroadcastMessageCommand* broadcast_message_command_create(
    uint64_t* channel_ids,
    size_t channel_count,
    void* message,
    size_t message_size
);
void broadcast_message_command_destroy(BroadcastMessageCommand* cmd);

#endif // CHANNEL_COMMANDS_H
