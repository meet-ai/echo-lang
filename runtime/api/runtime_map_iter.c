/**
 * @file runtime_map_iter.c
 * @brief Map迭代运行时函数实现（独立文件，避免依赖runtime_init）
 * 
 * 这个文件包含map迭代所需的运行时函数，不依赖runtime_init等复杂初始化逻辑
 */

#include "runtime_api.h"
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/**
 * 注意：runtime_null_ptr 已在 runtime_api.c 中定义，这里不再重复定义
 */

/**
 * @brief 获取map的键值对数组
 * 从map中提取所有键值对，返回键数组和值数组
 * 
 * 注意：map的内部表示使用hash_table_t结构
 * map_ptr指向hash_table_t结构（从common_types.h定义）
 */
MapIterResult* runtime_map_get_keys(void* map_ptr) {
    if (!map_ptr) {
        return NULL;
    }
    
    // 分配结果结构体
    MapIterResult* result = (MapIterResult*)malloc(sizeof(MapIterResult));
    if (!result) {
        return NULL;
    }
    
    // 初始化
    result->keys = NULL;
    result->values = NULL;
    result->count = 0;
    
    // 将map_ptr转换为hash_table_t指针
    // 注意：hash_table_t定义在runtime/shared/types/common_types.h中
    // 由于runtime_api.c可能不直接包含该头文件，我们需要使用前向声明
    // 或者直接使用void*指针进行类型转换
    
    // 前向声明hash_table_t和hash_entry_t结构
    // 注意：这些结构定义在runtime/shared/types/common_types.h中
    // string_t是结构体：{ char* data; size_t length; size_t capacity; }
    typedef struct {
        char* data;
        size_t length;
        size_t capacity;
    } string_t;
    
    typedef struct hash_entry_internal {
        string_t key;          // string_t key
        void* value;          // void* value（对于map[string]string，value是string_t*）
        struct hash_entry_internal* next;
    } hash_entry_internal_t;
    
    typedef struct {
        hash_entry_internal_t** buckets;  // hash_entry_t** buckets
        size_t bucket_count;              // size_t bucket_count
        size_t entry_count;               // size_t entry_count
        void* hash_function;              // uint32_t (*hash_function)(const char* key)
    } hash_table_internal_t;
    
    hash_table_internal_t* map = (hash_table_internal_t*)map_ptr;
    
    // 如果map为空，返回空结果
    if (!map || map->entry_count == 0) {
        return result;
    }
    
    // 分配键数组和值数组
    // 注意：键和值都是char*（string类型）
    result->keys = (char**)malloc(sizeof(char*) * map->entry_count);
    result->values = (char**)malloc(sizeof(char*) * map->entry_count);
    if (!result->keys || !result->values) {
        if (result->keys) free(result->keys);
        if (result->values) free(result->values);
        free(result);
        return NULL;
    }
    
    // 遍历所有bucket和entry，收集键值对
    size_t index = 0;
    for (size_t i = 0; i < map->bucket_count; i++) {
        if (!map->buckets) break;
        hash_entry_internal_t* entry = map->buckets[i];
        while (entry) {
            if (index >= map->entry_count) {
                // 防止数组越界
                break;
            }
            
            // 复制键（string_t.data是char*）
            if (entry->key.data && entry->key.length > 0) {
                size_t key_len = entry->key.length + 1;
                result->keys[index] = (char*)malloc(key_len);
                if (result->keys[index]) {
                    memcpy(result->keys[index], entry->key.data, entry->key.length);
                    result->keys[index][entry->key.length] = '\0';
                }
            } else {
                result->keys[index] = NULL;
            }
            
            // 复制值（对于map[string]string，value是string_t*）
            if (entry->value) {
                string_t* value_str = (string_t*)entry->value;
                if (value_str->data && value_str->length > 0) {
                    size_t value_len = value_str->length + 1;
                    result->values[index] = (char*)malloc(value_len);
                    if (result->values[index]) {
                        memcpy(result->values[index], value_str->data, value_str->length);
                        result->values[index][value_str->length] = '\0';
                    }
                } else {
                    result->values[index] = NULL;
                }
            } else {
                result->values[index] = NULL;
            }
            
            index++;
            entry = entry->next;
        }
    }
    
    result->count = (int32_t)index;
    
    return result;
}
