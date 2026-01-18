#ifndef ECHO_RUNTIME_NETWORK_H
#define ECHO_RUNTIME_NETWORK_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

// 前向声明
typedef struct network_connection_t network_connection_t;
typedef struct network_listener_t network_listener_t;

// =============================================================================
// 值对象定义 (Value Objects)
// =============================================================================

// 网络地址值对象
typedef struct {
    const char* host;        // 主机名或IP地址
    uint16_t port;           // 端口号
    const char* protocol;    // 协议 ("tcp", "udp", "unix")
} network_address_t;

// 连接状态值对象
typedef enum {
    CONNECTION_CLOSED,
    CONNECTION_CONNECTING,
    CONNECTION_CONNECTED,
    CONNECTION_CLOSING,
    CONNECTION_ERROR
} connection_state_t;

// 网络统计值对象
typedef struct {
    size_t bytes_sent;           // 发送字节数
    size_t bytes_received;       // 接收字节数
    size_t packets_sent;         // 发送包数
    size_t packets_received;     // 接收包数
    double connection_duration_s; // 连接持续时间(秒)
    size_t error_count;          // 错误次数
} network_stats_t;

// 超时配置值对象
typedef struct {
    int64_t connect_timeout_ms;   // 连接超时 (毫秒)
    int64_t read_timeout_ms;      // 读取超时 (毫秒)
    int64_t write_timeout_ms;     // 写入超时 (毫秒)
    bool keep_alive;             // 是否保持连接
    int64_t keep_alive_interval_s; // 保活间隔 (秒)
} network_timeout_config_t;

// =============================================================================
// 实体接口定义 (Entity Interfaces)
// =============================================================================

// 网络连接实体接口
typedef struct network_connection_t network_connection_t;

network_connection_t* network_connection_create(const network_address_t* address);
void network_connection_destroy(network_connection_t* connection);
uint64_t network_connection_get_id(const network_connection_t* connection);
const network_address_t* network_connection_get_local_address(const network_connection_t* connection);
const network_address_t* network_connection_get_remote_address(const network_connection_t* connection);
connection_state_t network_connection_get_state(const network_connection_t* connection);
network_stats_t network_connection_get_stats(const network_connection_t* connection);

// 连接操作接口
int network_connection_connect(network_connection_t* connection, timeout_t timeout);
int network_connection_disconnect(network_connection_t* connection);
int network_connection_send(network_connection_t* connection, const void* data, size_t size);
int network_connection_receive(network_connection_t* connection, void* buffer, size_t buffer_size, size_t* received_size);
int network_connection_send_async(
    network_connection_t* connection,
    const void* data,
    size_t size,
    void (*callback)(int result, void* user_data),
    void* user_data
);
int network_connection_receive_async(
    network_connection_t* connection,
    void* buffer,
    size_t buffer_size,
    void (*callback)(int result, size_t received_size, void* user_data),
    void* user_data
);

// 连接配置接口
void network_connection_set_timeout_config(network_connection_t* connection, const network_timeout_config_t* config);
const network_timeout_config_t* network_connection_get_timeout_config(const network_connection_t* connection);
void network_connection_enable_ssl(network_connection_t* connection, const char* cert_path, const char* key_path);
bool network_connection_is_ssl_enabled(const network_connection_t* connection);

// 网络监听器实体接口
typedef struct network_listener_t network_listener_t;

network_listener_t* network_listener_create(const network_address_t* address);
void network_listener_destroy(network_listener_t* listener);
uint64_t network_listener_get_id(const network_listener_t* listener);
const network_address_t* network_listener_get_address(const network_listener_t* listener);
bool network_listener_is_listening(const network_listener_t* listener);

// 监听器操作接口
int network_listener_start_listening(network_listener_t* listener);
int network_listener_stop_listening(network_listener_t* listener);
network_connection_t* network_listener_accept(network_listener_t* listener, timeout_t timeout);
int network_listener_accept_async(
    network_listener_t* listener,
    void (*callback)(network_connection_t* connection, void* user_data),
    void* user_data
);

// 监听器配置接口
void network_listener_set_backlog(network_listener_t* listener, int backlog);
int network_listener_get_backlog(const network_listener_t* listener);
void network_listener_enable_ssl(network_listener_t* listener, const char* cert_path, const char* key_path);

// =============================================================================
// 领域服务接口定义 (Domain Services)
// =============================================================================

// 连接管理服务接口
typedef struct connection_manager_service_t connection_manager_service_t;

connection_manager_service_t* connection_manager_service_create(void);
void connection_manager_service_destroy(connection_manager_service_t* service);

// 连接生命周期管理
network_connection_t* connection_manager_service_create_connection(
    connection_manager_service_t* service,
    const network_address_t* address
);
void connection_manager_service_destroy_connection(connection_manager_service_t* service, uint64_t connection_id);
network_connection_t* connection_manager_service_find_connection(connection_manager_service_t* service, uint64_t connection_id);

// 连接池管理
void connection_manager_service_set_pool_config(connection_manager_service_t* service, size_t max_connections, timeout_t idle_timeout);
size_t connection_manager_service_get_active_connections_count(const connection_manager_service_t* service);
size_t connection_manager_service_get_idle_connections_count(const connection_manager_service_t* service);
void connection_manager_service_cleanup_idle_connections(connection_manager_service_t* service);

// 网络通信服务接口
typedef struct network_communication_service_t network_communication_service_t;

network_communication_service_t* network_communication_service_create(connection_manager_service_t* manager);
void network_communication_service_destroy(network_communication_service_t* service);

// 同步通信操作
int network_communication_service_send_data(
    network_communication_service_t* service,
    uint64_t connection_id,
    const void* data,
    size_t size
);
int network_communication_service_receive_data(
    network_communication_service_t* service,
    uint64_t connection_id,
    void* buffer,
    size_t buffer_size,
    size_t* received_size
);

// 异步通信操作
int network_communication_service_send_data_async(
    network_communication_service_t* service,
    uint64_t connection_id,
    const void* data,
    size_t size,
    void (*callback)(int result, void* user_data),
    void* user_data
);
int network_communication_service_receive_data_async(
    network_communication_service_t* service,
    uint64_t connection_id,
    void* buffer,
    size_t buffer_size,
    void (*callback)(int result, size_t received_size, void* user_data),
    void* user_data
);

// 服务器端操作
network_listener_t* network_communication_service_create_listener(
    network_communication_service_t* service,
    const network_address_t* address
);
void network_communication_service_destroy_listener(network_communication_service_t* service, uint64_t listener_id);
int network_communication_service_start_server(network_communication_service_t* service, uint64_t listener_id);
int network_communication_service_stop_server(network_communication_service_t* service, uint64_t listener_id);

// 网络监控服务接口
typedef struct network_monitoring_service_t network_monitoring_service_t;

network_monitoring_service_t* network_monitoring_service_create(connection_manager_service_t* manager);
void network_monitoring_service_destroy(network_monitoring_service_t* service);
void network_monitoring_service_start_monitoring(network_monitoring_service_t* service);
void network_monitoring_service_stop_monitoring(network_monitoring_service_t* service);

// 网络监控指标
typedef struct network_metrics_t {
    size_t active_connections;
    size_t total_connections_created;
    size_t total_bytes_sent;
    size_t total_bytes_received;
    size_t connection_errors;
    size_t timeout_errors;
    double avg_connection_duration_s;
} network_metrics_t;

network_metrics_t network_monitoring_service_get_metrics(const network_monitoring_service_t* service);

// 网络健康检查
typedef enum {
    NETWORK_HEALTH_GOOD,
    NETWORK_HEALTH_WARNING,
    NETWORK_HEALTH_CRITICAL
} network_health_status_t;

network_health_status_t network_monitoring_service_check_health(const network_monitoring_service_t* service);
const char* network_monitoring_service_get_health_message(const network_monitoring_service_t* service);

// 网络协议服务接口
typedef struct network_protocol_service_t network_protocol_service_t;

network_protocol_service_t* network_protocol_service_create(void);
void network_protocol_service_destroy(network_protocol_service_t* service);

// 协议解析
typedef enum {
    PROTOCOL_TCP,
    PROTOCOL_UDP,
    PROTOCOL_HTTP,
    PROTOCOL_WEBSOCKET,
    PROTOCOL_CUSTOM
} protocol_type_t;

void network_protocol_service_register_protocol(
    network_protocol_service_t* service,
    protocol_type_t protocol,
    void (*parse_handler)(const void* data, size_t size, void* context),
    void* context
);

int network_protocol_service_parse_message(
    network_protocol_service_t* service,
    protocol_type_t protocol,
    const void* data,
    size_t size,
    void** parsed_message,
    size_t* parsed_size
);

// SSL/TLS管理
typedef struct ssl_config_t {
    const char* cert_file;
    const char* key_file;
    const char* ca_file;
    bool verify_peer;
    const char* cipher_suites;
} ssl_config_t;

void network_protocol_service_enable_ssl(
    network_protocol_service_t* service,
    const ssl_config_t* config
);
bool network_protocol_service_is_ssl_enabled(const network_protocol_service_t* service);

// =============================================================================
// 仓储接口定义 (Repository Interfaces)
// =============================================================================

// 连接仓储接口
typedef struct connection_repository_t connection_repository_t;

connection_repository_t* connection_repository_create(void);
void connection_repository_destroy(connection_repository_t* repository);
void connection_repository_save(connection_repository_t* repository, const network_connection_t* connection);
network_connection_t* connection_repository_find_by_id(connection_repository_t* repository, uint64_t id);
void connection_repository_delete(connection_repository_t* repository, uint64_t id);
size_t connection_repository_count_active_connections(const connection_repository_t* repository);

// 监听器仓储接口
typedef struct listener_repository_t listener_repository_t;

listener_repository_t* listener_repository_create(void);
void listener_repository_destroy(listener_repository_t* repository);
void listener_repository_save(listener_repository_t* repository, const network_listener_t* listener);
network_listener_t* listener_repository_find_by_id(listener_repository_t* repository, uint64_t id);
void listener_repository_delete(listener_repository_t* repository, uint64_t id);
size_t connection_repository_count_active_listeners(const listener_repository_t* repository);

// =============================================================================
// 工具函数
// =============================================================================

// 网络地址创建函数
network_address_t* network_address_create(const char* host, uint16_t port, const char* protocol);
void network_address_destroy(network_address_t* address);

// 超时配置创建函数
network_timeout_config_t network_timeout_config_default(void);
network_timeout_config_t network_timeout_config_create(
    int64_t connect_timeout_ms,
    int64_t read_timeout_ms,
    int64_t write_timeout_ms
);

// 网络统计创建函数
network_stats_t network_stats_zero(void);

#endif // ECHO_RUNTIME_NETWORK_H
