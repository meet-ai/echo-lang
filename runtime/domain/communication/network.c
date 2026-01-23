#include "network.h"
#include "aggregate/network_connection.h"
#include "aggregate/network_listener.h"
#include "service/connection_manager_service.h"
#include "service/network_communication_service.h"
#include "service/network_monitoring_service.h"
#include "service/network_protocol_service.h"
#include "repository/connection_repository.h"
#include "repository/listener_repository.h"
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

// =============================================================================
// 值对象实现 (Value Objects)
// =============================================================================

// 网络地址值对象实现
network_address_t* network_address_create(const char* host, uint16_t port, const char* protocol) {
    network_address_t* address = (network_address_t*)malloc(sizeof(network_address_t));
    if (!address) return NULL;

    address->host = strdup(host);
    address->port = port;
    address->protocol = strdup(protocol);

    return address;
}

void network_address_destroy(network_address_t* address) {
    if (address) {
        free((void*)address->host);
        free((void*)address->protocol);
        free(address);
    }
}

// 超时配置默认值
network_timeout_config_t network_timeout_config_default(void) {
    return (network_timeout_config_t){
        .connect_timeout_ms = 5000,  // 5秒
        .read_timeout_ms = 30000,     // 30秒
        .write_timeout_ms = 30000,    // 30秒
        .keep_alive = true,
        .keep_alive_interval_s = 60   // 60秒
    };
}

network_timeout_config_t network_timeout_config_create(
    int64_t connect_timeout_ms,
    int64_t read_timeout_ms,
    int64_t write_timeout_ms
) {
    network_timeout_config_t config = network_timeout_config_default();
    config.connect_timeout_ms = connect_timeout_ms;
    config.read_timeout_ms = read_timeout_ms;
    config.write_timeout_ms = write_timeout_ms;
    return config;
}

// 网络统计初始化
network_stats_t network_stats_zero(void) {
    return (network_stats_t){
        .bytes_sent = 0,
        .bytes_received = 0,
        .packets_sent = 0,
        .packets_received = 0,
        .connection_duration_s = 0.0,
        .error_count = 0
    };
}

// =============================================================================
// 实体接口实现转发 (Entity Interface Forwarding)
// =============================================================================

// 网络连接实体接口 - 转发到聚合实现
network_connection_t* network_connection_create(const network_address_t* address) {
    return network_connection_aggregate_create(address);
}

void network_connection_destroy(network_connection_t* connection) {
    network_connection_aggregate_destroy(connection);
}

uint64_t network_connection_get_id(const network_connection_t* connection) {
    return network_connection_aggregate_get_id(connection);
}

const network_address_t* network_connection_get_local_address(const network_connection_t* connection) {
    return network_connection_aggregate_get_local_address(connection);
}

const network_address_t* network_connection_get_remote_address(const network_connection_t* connection) {
    return network_connection_aggregate_get_remote_address(connection);
}

connection_state_t network_connection_get_state(const network_connection_t* connection) {
    return network_connection_aggregate_get_state(connection);
}

network_stats_t network_connection_get_stats(const network_connection_t* connection) {
    return network_connection_aggregate_get_stats(connection);
}

int network_connection_connect(network_connection_t* connection, timeout_t timeout) {
    return network_connection_aggregate_connect(connection, timeout);
}

int network_connection_disconnect(network_connection_t* connection) {
    return network_connection_aggregate_disconnect(connection);
}

int network_connection_send(network_connection_t* connection, const void* data, size_t size) {
    return network_connection_aggregate_send(connection, data, size);
}

int network_connection_receive(network_connection_t* connection, void* buffer, size_t buffer_size, size_t* received_size) {
    return network_connection_aggregate_receive(connection, buffer, buffer_size, received_size);
}

int network_connection_send_async(
    network_connection_t* connection,
    const void* data,
    size_t size,
    void (*callback)(int result, void* user_data),
    void* user_data
) {
    return network_connection_aggregate_send_async(connection, data, size, callback, user_data);
}

int network_connection_receive_async(
    network_connection_t* connection,
    void* buffer,
    size_t buffer_size,
    void (*callback)(int result, size_t received_size, void* user_data),
    void* user_data
) {
    return network_connection_aggregate_receive_async(connection, buffer, buffer_size, callback, user_data);
}

void network_connection_set_timeout_config(network_connection_t* connection, const network_timeout_config_t* config) {
    network_connection_aggregate_set_timeout_config(connection, config);
}

const network_timeout_config_t* network_connection_get_timeout_config(const network_connection_t* connection) {
    return network_connection_aggregate_get_timeout_config(connection);
}

void network_connection_enable_ssl(network_connection_t* connection, const char* cert_path, const char* key_path) {
    network_connection_aggregate_enable_ssl(connection, cert_path, key_path);
}

bool network_connection_is_ssl_enabled(const network_connection_t* connection) {
    return network_connection_aggregate_is_ssl_enabled(connection);
}

// 网络监听器实体接口 - 转发到聚合实现
network_listener_t* network_listener_create(const network_address_t* address) {
    return network_listener_aggregate_create(address);
}

void network_listener_destroy(network_listener_t* listener) {
    network_listener_aggregate_destroy(listener);
}

uint64_t network_listener_get_id(const network_listener_t* listener) {
    return network_listener_aggregate_get_id(listener);
}

const network_address_t* network_listener_get_address(const network_listener_t* listener) {
    return network_listener_aggregate_get_address(listener);
}

bool network_listener_is_listening(const network_listener_t* listener) {
    return network_listener_aggregate_is_listening(listener);
}

int network_listener_start_listening(network_listener_t* listener) {
    return network_listener_aggregate_start_listening(listener);
}

int network_listener_stop_listening(network_listener_t* listener) {
    return network_listener_aggregate_stop_listening(listener);
}

network_connection_t* network_listener_accept(network_listener_t* listener, timeout_t timeout) {
    return network_listener_aggregate_accept(listener, timeout);
}

int network_listener_accept_async(
    network_listener_t* listener,
    void (*callback)(network_connection_t* connection, void* user_data),
    void* user_data
) {
    return network_listener_aggregate_accept_async(listener, callback, user_data);
}

void network_listener_set_backlog(network_listener_t* listener, int backlog) {
    network_listener_aggregate_set_backlog(listener, backlog);
}

int network_listener_get_backlog(const network_listener_t* listener) {
    return network_listener_aggregate_get_backlog(listener);
}

void network_listener_enable_ssl(network_listener_t* listener, const char* cert_path, const char* key_path) {
    network_listener_aggregate_enable_ssl(listener, cert_path, key_path);
}

// =============================================================================
// 领域服务接口实现转发 (Domain Service Interface Forwarding)
// =============================================================================

// 连接管理服务接口 - 转发到服务实现
connection_manager_service_t* connection_manager_service_create(void) {
    return connection_manager_service_create_impl();
}

void connection_manager_service_destroy(connection_manager_service_t* service) {
    connection_manager_service_destroy_impl(service);
}

network_connection_t* connection_manager_service_create_connection(
    connection_manager_service_t* service,
    const network_address_t* address
) {
    return connection_manager_service_create_connection_impl(service, address);
}

void connection_manager_service_destroy_connection(connection_manager_service_t* service, uint64_t connection_id) {
    connection_manager_service_destroy_connection_impl(service, connection_id);
}

network_connection_t* connection_manager_service_find_connection(connection_manager_service_t* service, uint64_t connection_id) {
    return connection_manager_service_find_connection_impl(service, connection_id);
}

void connection_manager_service_set_pool_config(connection_manager_service_t* service, size_t max_connections, timeout_t idle_timeout) {
    connection_manager_service_set_pool_config_impl(service, max_connections, idle_timeout);
}

size_t connection_manager_service_get_active_connections_count(const connection_manager_service_t* service) {
    return connection_manager_service_get_active_connections_count_impl(service);
}

size_t connection_manager_service_get_idle_connections_count(const connection_manager_service_t* service) {
    return connection_manager_service_get_idle_connections_count_impl(service);
}

void connection_manager_service_cleanup_idle_connections(connection_manager_service_t* service) {
    connection_manager_service_cleanup_idle_connections_impl(service);
}

// 网络通信服务接口 - 转发到服务实现
network_communication_service_t* network_communication_service_create(connection_manager_service_t* manager) {
    return network_communication_service_create_impl(manager);
}

void network_communication_service_destroy(network_communication_service_t* service) {
    network_communication_service_destroy_impl(service);
}

int network_communication_service_send_data(
    network_communication_service_t* service,
    uint64_t connection_id,
    const void* data,
    size_t size
) {
    return network_communication_service_send_data_impl(service, connection_id, data, size);
}

int network_communication_service_receive_data(
    network_communication_service_t* service,
    uint64_t connection_id,
    void* buffer,
    size_t buffer_size,
    size_t* received_size
) {
    return network_communication_service_receive_data_impl(service, connection_id, buffer, buffer_size, received_size);
}

int network_communication_service_send_data_async(
    network_communication_service_t* service,
    uint64_t connection_id,
    const void* data,
    size_t size,
    void (*callback)(int result, void* user_data),
    void* user_data
) {
    return network_communication_service_send_data_async_impl(service, connection_id, data, size, callback, user_data);
}

int network_communication_service_receive_data_async(
    network_communication_service_t* service,
    uint64_t connection_id,
    void* buffer,
    size_t buffer_size,
    void (*callback)(int result, size_t received_size, void* user_data),
    void* user_data
) {
    return network_communication_service_receive_data_async_impl(service, connection_id, buffer, buffer_size, callback, user_data);
}

network_listener_t* network_communication_service_create_listener(
    network_communication_service_t* service,
    const network_address_t* address
) {
    return network_communication_service_create_listener_impl(service, address);
}

void network_communication_service_destroy_listener(network_communication_service_t* service, uint64_t listener_id) {
    network_communication_service_destroy_listener_impl(service, listener_id);
}

int network_communication_service_start_server(network_communication_service_t* service, uint64_t listener_id) {
    return network_communication_service_start_server_impl(service, listener_id);
}

int network_communication_service_stop_server(network_communication_service_t* service, uint64_t listener_id) {
    return network_communication_service_stop_server_impl(service, listener_id);
}

// 网络监控服务接口 - 转发到服务实现
network_monitoring_service_t* network_monitoring_service_create(connection_manager_service_t* manager) {
    return network_monitoring_service_create_impl(manager);
}

void network_monitoring_service_destroy(network_monitoring_service_t* service) {
    network_monitoring_service_destroy_impl(service);
}

void network_monitoring_service_start_monitoring(network_monitoring_service_t* service) {
    network_monitoring_service_start_monitoring_impl(service);
}

void network_monitoring_service_stop_monitoring(network_monitoring_service_t* service) {
    network_monitoring_service_stop_monitoring_impl(service);
}

network_metrics_t network_monitoring_service_get_metrics(const network_monitoring_service_t* service) {
    return network_monitoring_service_get_metrics_impl(service);
}

network_health_status_t network_monitoring_service_check_health(const network_monitoring_service_t* service) {
    return network_monitoring_service_check_health_impl(service);
}

const char* network_monitoring_service_get_health_message(const network_monitoring_service_t* service) {
    return network_monitoring_service_get_health_message_impl(service);
}

// 网络协议服务接口 - 转发到服务实现
network_protocol_service_t* network_protocol_service_create(void) {
    return network_protocol_service_create_impl();
}

void network_protocol_service_destroy(network_protocol_service_t* service) {
    network_protocol_service_destroy_impl(service);
}

void network_protocol_service_register_protocol(
    network_protocol_service_t* service,
    protocol_type_t protocol,
    void (*parse_handler)(const void* data, size_t size, void* context),
    void* context
) {
    network_protocol_service_register_protocol_impl(service, protocol, parse_handler, context);
}

int network_protocol_service_parse_message(
    network_protocol_service_t* service,
    protocol_type_t protocol,
    const void* data,
    size_t size,
    void** parsed_message,
    size_t* parsed_size
) {
    return network_protocol_service_parse_message_impl(service, protocol, data, size, parsed_message, parsed_size);
}

void network_protocol_service_enable_ssl(
    network_protocol_service_t* service,
    const ssl_config_t* config
) {
    network_protocol_service_enable_ssl_impl(service, config);
}

bool network_protocol_service_is_ssl_enabled(const network_protocol_service_t* service) {
    return network_protocol_service_is_ssl_enabled_impl(service);
}

// =============================================================================
// 仓储接口实现转发 (Repository Interface Forwarding)
// =============================================================================

// 连接仓储接口 - 转发到仓储实现
connection_repository_t* connection_repository_create(void) {
    return connection_repository_create_impl();
}

void connection_repository_destroy(connection_repository_t* repository) {
    connection_repository_destroy_impl(repository);
}

void connection_repository_save(connection_repository_t* repository, const network_connection_t* connection) {
    connection_repository_save_impl(repository, connection);
}

network_connection_t* connection_repository_find_by_id(connection_repository_t* repository, uint64_t id) {
    return connection_repository_find_by_id_impl(repository, id);
}

void connection_repository_delete(connection_repository_t* repository, uint64_t id) {
    connection_repository_delete_impl(repository, id);
}

size_t connection_repository_count_active_connections(const connection_repository_t* repository) {
    return connection_repository_count_active_connections_impl(repository);
}

// 监听器仓储接口 - 转发到仓储实现
listener_repository_t* listener_repository_create(void) {
    return listener_repository_create_impl();
}

void listener_repository_destroy(listener_repository_t* repository) {
    listener_repository_destroy_impl(repository);
}

void listener_repository_save(listener_repository_t* repository, const network_listener_t* listener) {
    listener_repository_save_impl(repository, listener);
}

network_listener_t* listener_repository_find_by_id(listener_repository_t* repository, uint64_t id) {
    return listener_repository_find_by_id_impl(repository, id);
}

void listener_repository_delete(listener_repository_t* repository, uint64_t id) {
    listener_repository_delete_impl(repository, id);
}

size_t connection_repository_count_active_listeners(const listener_repository_t* repository) {
    return connection_repository_count_active_listeners_impl(repository);
}
