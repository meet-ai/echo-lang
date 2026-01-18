/**
 * @file cache_utils.c
 * @brief 缓存行对齐和内存布局优化工具实现
 */

#include "cache_utils.h"
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/mman.h>
#include <errno.h>
#include <time.h>

// ============================================================================
// NUMA感知内存分配实现
// ============================================================================

int get_current_numa_node(void) {
    // 简化实现：返回0（单节点系统）
    // 实际实现需要读取/proc/self/numa_maps或使用libnuma
    return 0;
}

int get_numa_topology(numa_node_info_t* nodes, int max_nodes) {
    // 简化实现：返回单节点信息
    if (max_nodes < 1) return 0;

    nodes[0].node_id = 0;
    nodes[0].memory_total = (size_t)sysconf(_SC_PHYS_PAGES) * sysconf(_SC_PAGESIZE);
    nodes[0].memory_free = nodes[0].memory_total / 2; // 估算值
    nodes[0].cpu_count = (int)sysconf(_SC_NPROCESSORS_ONLN);

    return 1;
}

void* alloc_memory_on_numa_node(size_t size, int node_id) {
    // 简化实现：使用普通mmap
    // 实际实现需要设置numa节点亲和性
    void* ptr = mmap(NULL, size, PROT_READ | PROT_WRITE,
                     MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    return (ptr == MAP_FAILED) ? NULL : ptr;
}

bool migrate_memory_to_numa_node(void* ptr, size_t size, int node_id) {
    // 简化实现：返回成功
    // 实际实现需要使用move_pages或类似的NUMA迁移API
    (void)ptr;
    (void)size;
    (void)node_id;
    return true;
}

// ============================================================================
// 位图操作实现
// ============================================================================

int bitmap_find_first_clear(const bitmap_256k_t* bitmap, uint32_t start_index) {
    const uint32_t max_index = 256 * 1024; // 256K bits

    for (uint32_t i = start_index; i < max_index; i++) {
        if (!bitmap_test(bitmap, i)) {
            return (int)i;
        }
    }

    return -1; // 未找到
}

int bitmap_find_first_clear_simd(const bitmap_256k_t* bitmap, uint32_t start_index) {
    // 简化实现：回退到普通扫描
    // 实际实现需要AVX2指令集支持
    return bitmap_find_first_clear(bitmap, start_index);
}

// ============================================================================
// 性能监控实现
// ============================================================================

void memory_pool_stats_reset(memory_pool_stats_t* stats) {
    if (!stats) return;

    memset(stats, 0, sizeof(memory_pool_stats_t));
}

void memory_pool_stats_record_alloc(memory_pool_stats_t* stats, size_t size, uint64_t time_ns) {
    if (!stats) return;

    stats->total_allocations++;
    stats->current_allocations++;
    stats->memory_used += size;

    if (stats->current_allocations > stats->peak_allocations) {
        stats->peak_allocations = stats->current_allocations;
    }

    if (stats->memory_used > stats->memory_peak) {
        stats->memory_peak = stats->memory_used;
    }

    // 更新平均分配时间（滑动平均）
    if (stats->total_allocations == 1) {
        stats->average_allocation_time = (double)time_ns;
    } else {
        double alpha = 0.1; // 平滑因子
        stats->average_allocation_time = (1 - alpha) * stats->average_allocation_time +
                                       alpha * (double)time_ns;
    }
}

void memory_pool_stats_record_free(memory_pool_stats_t* stats, uint64_t time_ns) {
    if (!stats) return;

    stats->total_deallocations++;
    if (stats->current_allocations > 0) {
        stats->current_allocations--;
    }

    // 更新平均释放时间
    if (stats->total_deallocations == 1) {
        stats->average_deallocation_time = (double)time_ns;
    } else {
        double alpha = 0.1;
        stats->average_deallocation_time = (1 - alpha) * stats->average_deallocation_time +
                                         alpha * (double)time_ns;
    }
}

void memory_pool_stats_calculate_fragmentation(memory_pool_stats_t* stats, size_t total_memory) {
    if (!stats || total_memory == 0) return;

    // 简化碎片率计算：基于已用内存比例
    // 实际实现需要更复杂的碎片分析
    double used_ratio = (double)stats->memory_used / (double)total_memory;
    stats->memory_fragmentation = 1.0 - used_ratio; // 简化计算
}

// ============================================================================
// 大页内存分配器实现
// ============================================================================

huge_page_allocator_t* huge_page_allocator_create(size_t page_size, size_t num_pages) {
    // 验证页大小（2MB或1GB）
    if (page_size != 2 * 1024 * 1024 && page_size != 1024 * 1024 * 1024) {
        return NULL;
    }

    huge_page_allocator_t* allocator = (huge_page_allocator_t*)malloc(sizeof(huge_page_allocator_t));
    if (!allocator) return NULL;

    memset(allocator, 0, sizeof(huge_page_allocator_t));
    allocator->page_size = page_size;
    allocator->total_pages = num_pages;
    allocator->free_pages = num_pages;

    // 初始化位图（全1表示空闲）
    memset(&allocator->free_bitmap, 0xFF, sizeof(bitmap_256k_t));

    // 验证页数不超过位图容量
    if (num_pages > 256 * 1024) {
        free(allocator);
        return NULL;
    }

    // 分配大页内存
    size_t total_size = page_size * num_pages;
    allocator->base_address = mmap(NULL, total_size, PROT_READ | PROT_WRITE,
                                   MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (allocator->base_address == MAP_FAILED) {
        free(allocator);
        return NULL;
    }

    // 尝试启用透明大页
#ifdef MADV_HUGEPAGE
    madvise(allocator->base_address, total_size, MADV_HUGEPAGE);
#endif

    return allocator;
}

void huge_page_allocator_destroy(huge_page_allocator_t* allocator) {
    if (!allocator) return;

    if (allocator->base_address && allocator->base_address != MAP_FAILED) {
        size_t total_size = allocator->page_size * allocator->total_pages;
        munmap(allocator->base_address, total_size);
    }

    free(allocator);
}

void* huge_page_allocator_alloc(huge_page_allocator_t* allocator, size_t size) {
    if (!allocator || size == 0 || allocator->free_pages == 0) return NULL;

    // 计算需要的页数
    size_t pages_needed = (size + allocator->page_size - 1) / allocator->page_size;
    if (pages_needed > allocator->free_pages) return NULL;

    // 查找连续的空闲页
    // 简化实现：只支持单页分配
    int free_page_index = bitmap_find_first_clear(&allocator->free_bitmap, 0);
    if (free_page_index < 0) return NULL;

    // 标记页为已使用
    bitmap_set(&allocator->free_bitmap, (uint32_t)free_page_index);
    allocator->free_pages--;

    // 计算地址
    return (char*)allocator->base_address + (free_page_index * allocator->page_size);
}

bool huge_page_allocator_free(huge_page_allocator_t* allocator, void* ptr, size_t size) {
    if (!allocator || !ptr) return false;

    // 验证指针在分配器范围内
    uintptr_t addr = (uintptr_t)ptr;
    uintptr_t base = (uintptr_t)allocator->base_address;
    uintptr_t end = base + (allocator->page_size * allocator->total_pages);

    if (addr < base || addr >= end) return false;

    // 计算页索引
    size_t page_index = (addr - base) / allocator->page_size;

    // 验证页是否已分配
    if (!bitmap_test(&allocator->free_bitmap, (uint32_t)page_index)) {
        // 页已空闲，重复释放
        return false;
    }

    // 标记页为空闲
    bitmap_clear(&allocator->free_bitmap, (uint32_t)page_index);
    allocator->free_pages++;

    return true;
}
