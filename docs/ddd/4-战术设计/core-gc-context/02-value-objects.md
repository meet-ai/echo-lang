# 核心GC上下文 - 值对象定义

## 1. GCConfiguration (GC配置)

### 1.1 业务含义
GC的行为配置参数，通过值对象保证配置的不可变性和完整性。

### 1.2 构成属性
```go
type GCConfiguration struct {
    // 触发配置
    triggerRatio float64       // GC触发阈值 (0.0-1.0)
    
    // 性能目标
    targetPauseTime time.Duration  // 目标暂停时间
    maxPauseTime    time.Duration  // 最大暂停时间
    
    // 堆配置
    initialHeapSize int64       // 初始堆大小
    maxHeapSize     int64       // 最大堆大小
    heapGrowthRatio float64     // 堆增长比率
    
    // 并发配置
    maxConcurrentWorkers int     // 最大并发工作线程数
    
    // 调试配置
    enableTracing     bool      // 启用追踪
    enableProfiling   bool      // 启用性能剖析
}
```

### 1.3 业务行为

#### 创建配置
```go
func NewGCConfiguration(opts ...ConfigurationOption) (GCConfiguration, error) {
    config := GCConfiguration{
        triggerRatio:       0.75,  // 默认75%
        targetPauseTime:    10 * time.Millisecond,
        maxPauseTime:       100 * time.Millisecond,
        initialHeapSize:    16 * 1024 * 1024,  // 16MB
        maxHeapSize:        1024 * 1024 * 1024, // 1GB
        heapGrowthRatio:    1.5,
        maxConcurrentWorkers: runtime.NumCPU(),
        enableTracing:      false,
        enableProfiling:    false,
    }
    
    // 应用选项
    for _, opt := range opts {
        opt(&config)
    }
    
    // 验证配置
    if err := config.Validate(); err != nil {
        return GCConfiguration{}, err
    }
    
    return config, nil
}
```

#### 配置验证
```go
func (c GCConfiguration) Validate() error {
    if c.triggerRatio <= 0 || c.triggerRatio >= 1 {
        return fmt.Errorf("trigger ratio must be between 0 and 1")
    }
    
    if c.targetPauseTime <= 0 {
        return fmt.Errorf("target pause time must be positive")
    }
    
    if c.maxPauseTime < c.targetPauseTime {
        return fmt.Errorf("max pause time must be >= target pause time")
    }
    
    if c.initialHeapSize <= 0 || c.maxHeapSize <= c.initialHeapSize {
        return fmt.Errorf("invalid heap size configuration")
    }
    
    return nil
}
```

#### 配置比较
```go
func (c GCConfiguration) Equals(other GCConfiguration) bool {
    return c.triggerRatio == other.triggerRatio &&
           c.targetPauseTime == other.targetPauseTime &&
           c.maxPauseTime == other.maxPauseTime &&
           c.initialHeapSize == other.initialHeapSize &&
           c.maxHeapSize == other.maxHeapSize &&
           c.heapGrowthRatio == other.heapGrowthRatio &&
           c.maxConcurrentWorkers == other.maxConcurrentWorkers &&
           c.enableTracing == other.enableTracing &&
           c.enableProfiling == other.enableProfiling
}
```

## 2. PerformanceMetrics (性能指标)

### 2.1 业务含义
GC执行的性能度量结果，用于监控和自适应决策。

### 2.2 构成属性
```go
type PerformanceMetrics struct {
    // 时间指标
    pauseTimeP50    time.Duration  // P50暂停时间
    pauseTimeP95    time.Duration  // P95暂停时间
    pauseTimeP99    time.Duration  // P99暂停时间
    totalPauseTime  time.Duration  // 总暂停时间
    gcDuration      time.Duration  // GC总持续时间
    
    // 内存指标
    heapUsageRatio     float64      // 堆使用率
    memoryReclaimed    int64        // 回收内存量
    allocationRate     int64        // 分配速率 (bytes/sec)
    fragmentationRatio float64      // 碎片率
    
    // 对象统计
    totalObjects       int64        // 总对象数
    liveObjects        int64        // 存活对象数
    markedObjects      int64        // 已标记对象数
    promotedObjects    int64        // 晋升对象数
    
    // CPU指标
    gcCPUPercentage    float64      // GC CPU使用率
    mutatorUtilization float64      // 应用线程利用率
    
    // 收集时间戳
    collectedAt        time.Time    // 收集时间
}
```

### 2.3 业务行为

#### 创建指标
```go
func NewPerformanceMetrics() *PerformanceMetricsBuilder {
    return &PerformanceMetricsBuilder{
        collectedAt: time.Now(),
    }
}

type PerformanceMetricsBuilder struct {
    metrics PerformanceMetrics
}

func (b *PerformanceMetricsBuilder) WithPauseTimes(p50, p95, p99, total time.Duration) *PerformanceMetricsBuilder {
    b.metrics.pauseTimeP50 = p50
    b.metrics.pauseTimeP95 = p95
    b.metrics.pauseTimeP99 = p99
    b.metrics.totalPauseTime = total
    return b
}

func (b *PerformanceMetricsBuilder) WithMemoryStats(usageRatio float64, reclaimed, allocRate int64, fragmentation float64) *PerformanceMetricsBuilder {
    b.metrics.heapUsageRatio = usageRatio
    b.metrics.memoryReclaimed = reclaimed
    b.metrics.allocationRate = allocRate
    b.metrics.fragmentationRatio = fragmentation
    return b
}

func (b *PerformanceMetricsBuilder) Build() PerformanceMetrics {
    return b.metrics
}
```

#### 计算衍生指标
```go
func (m PerformanceMetrics) SurvivalRate() float64 {
    if m.totalObjects == 0 {
        return 0
    }
    return float64(m.liveObjects) / float64(m.totalObjects)
}

func (m PerformanceMetrics) ThroughputLoss() float64 {
    if m.gcDuration == 0 {
        return 0
    }
    return float64(m.totalPauseTime) / float64(m.gcDuration)
}

func (m PerformanceMetrics) MemoryEfficiency() float64 {
    if m.memoryReclaimed == 0 {
        return 0
    }
    // 计算内存回收效率
    return float64(m.memoryReclaimed) / float64(m.liveObjects)
}
```

#### 指标比较
```go
func (m PerformanceMetrics) IsBetterThan(other PerformanceMetrics) bool {
    // 多维度比较：暂停时间、吞吐量、内存效率
    pauseScore := m.pauseScore() - other.pauseScore()
    throughputScore := m.throughputScore() - other.throughputScore()
    memoryScore := m.memoryScore() - other.memoryScore()
    
    // 加权平均
    totalScore := pauseScore*0.5 + throughputScore*0.3 + memoryScore*0.2
    return totalScore > 0
}

func (m PerformanceMetrics) pauseScore() float64 {
    // 暂停时间评分：时间越短分数越高
    return 1.0 / (1.0 + float64(m.pauseTimeP95.Milliseconds()))
}
```

## 3. GCExecutionStats (GC执行统计)

### 3.1 业务含义
GC执行的累积统计信息，反映GC系统的长期运行状况。

### 3.2 构成属性
```go
type GCExecutionStats struct {
    // 基本统计
    totalCycles      int64         // 总GC周期数
    successfulCycles int64         // 成功GC周期数
    failedCycles     int64         // 失败GC周期数
    
    // 时间统计
    totalGCTime      time.Duration  // 总GC时间
    totalPauseTime   time.Duration  // 总暂停时间
    averageGCDuration time.Duration // 平均GC持续时间
    maxGCDuration    time.Duration  // 最大GC持续时间
    minGCDuration    time.Duration  // 最小GC持续时间
    
    // 内存统计
    totalMemoryAllocated   int64   // 总分配内存
    totalMemoryReclaimed   int64   // 总回收内存
    peakHeapUsage         int64   // 峰值堆使用
    averageHeapUsage      int64   // 平均堆使用
    
    // 对象统计
    totalObjectsAllocated int64   // 总分配对象数
    totalObjectsCollected int64   // 总回收对象数
    averageObjectSize     int64   // 平均对象大小
    
    // 时间戳
    lastTriggeredAt  *time.Time  // 最后触发时间
    lastCompletedAt  *time.Time  // 最后完成时间
    
    // 触发原因统计
    triggerReasons   map[string]int64  // 触发原因计数
}
```

### 3.3 业务行为

#### 合并统计
```go
func (s GCExecutionStats) Merge(other GCExecutionStats) GCExecutionStats {
    merged := GCExecutionStats{
        totalCycles:           s.totalCycles + other.totalCycles,
        successfulCycles:      s.successfulCycles + other.successfulCycles,
        failedCycles:          s.failedCycles + other.failedCycles,
        totalGCTime:           s.totalGCTime + other.totalGCTime,
        totalPauseTime:        s.totalPauseTime + other.totalPauseTime,
        totalMemoryAllocated:  s.totalMemoryAllocated + other.totalMemoryAllocated,
        totalMemoryReclaimed:  s.totalMemoryReclaimed + other.totalMemoryReclaimed,
        totalObjectsAllocated: s.totalObjectsAllocated + other.totalObjectsAllocated,
        totalObjectsCollected: s.totalObjectsCollected + other.totalObjectsCollected,
    }
    
    // 重新计算平均值
    if merged.totalCycles > 0 {
        merged.averageGCDuration = merged.totalGCTime / time.Duration(merged.totalCycles)
    }
    
    // 更新峰值
    if other.maxGCDuration > merged.maxGCDuration {
        merged.maxGCDuration = other.maxGCDuration
    }
    
    // 更新时间戳
    if other.lastCompletedAt != nil {
        merged.lastCompletedAt = other.lastCompletedAt
    }
    
    // 合并触发原因
    merged.triggerReasons = make(map[string]int64)
    for reason, count := range s.triggerReasons {
        merged.triggerReasons[reason] = count
    }
    for reason, count := range other.triggerReasons {
        merged.triggerReasons[reason] += count
    }
    
    return merged
}
```

#### 计算效率指标
```go
func (s GCExecutionStats) OverallEfficiency() float64 {
    if s.totalGCTime == 0 {
        return 0
    }
    
    // 综合效率：回收效率 * 时间效率
    reclamationEfficiency := float64(s.totalMemoryReclaimed) / float64(s.totalMemoryAllocated)
    timeEfficiency := float64(s.totalPauseTime) / float64(s.totalGCTime)
    
    return reclamationEfficiency * (1 - timeEfficiency)
}

func (s GCExecutionStats) SuccessRate() float64 {
    if s.totalCycles == 0 {
        return 0
    }
    return float64(s.successfulCycles) / float64(s.totalCycles)
}
```

## 4. ObjectType (对象类型)

### 4.1 业务含义
对象的类型信息，用于GC时的类型特定处理。

### 4.2 定义
```go
type ObjectType struct {
    name       string        // 类型名称
    size       int64         // 类型大小
    hasPointers bool         // 是否包含指针
    alignment  int           // 对齐要求
    category   TypeCategory  // 类型类别
}

type TypeCategory int

const (
    TypeCategoryPrimitive TypeCategory = iota  // 基本类型
    TypeCategoryArray                          // 数组
    TypeCategoryStruct                         // 结构体
    TypeCategoryInterface                      // 接口
    TypeCategoryChannel                        // 通道
    TypeCategorySlice                          // 切片
    TypeCategoryMap                            // 映射
    TypeCategoryString                         // 字符串
)
```

### 4.3 业务行为

#### 类型注册
```go
var typeRegistry = make(map[string]ObjectType)

func RegisterObjectType(name string, size int64, hasPointers bool, alignment int, category TypeCategory) ObjectType {
    objType := ObjectType{
        name:        name,
        size:        size,
        hasPointers: hasPointers,
        alignment:   alignment,
        category:    category,
    }
    
    typeRegistry[name] = objType
    return objType
}
```

#### 类型查询
```go
func GetObjectType(name string) (ObjectType, bool) {
    objType, exists := typeRegistry[name]
    return objType, exists
}

func (t ObjectType) IsReferenceType() bool {
    return t.category == TypeCategoryInterface ||
           t.category == TypeCategoryChannel ||
           t.category == TypeCategorySlice ||
           t.category == TypeCategoryMap
}

func (t ObjectType) RequiresSpecialHandling() bool {
    return t.IsReferenceType() || t.hasPointers
}
```

## 5. MemoryAddress (内存地址)

### 5.1 业务含义
堆中对象的内存地址值对象，保证地址的类型安全。

### 5.2 定义
```go
type MemoryAddress struct {
    value uintptr
}
```

### 5.3 业务行为

#### 地址运算
```go
func (addr MemoryAddress) Add(offset uintptr) MemoryAddress {
    return MemoryAddress{value: addr.value + offset}
}

func (addr MemoryAddress) Subtract(other MemoryAddress) uintptr {
    return addr.value - other.value
}

func (addr MemoryAddress) IsValid() bool {
    return addr.value != 0
}

func (addr MemoryAddress) IsAligned(alignment uintptr) bool {
    return addr.value%alignment == 0
}
```

#### 地址比较
```go
func (addr MemoryAddress) Equals(other MemoryAddress) bool {
    return addr.value == other.value
}

func (addr MemoryAddress) LessThan(other MemoryAddress) bool {
    return addr.value < other.value
}

func (addr MemoryAddress) InRange(start, end MemoryAddress) bool {
    return addr.value >= start.value && addr.value < end.value
}
```

## 6. Size (大小)

### 6.1 业务含义
内存大小的值对象，提供类型安全的大小计算。

### 6.2 定义
```go
type Size struct {
    bytes int64
}
```

### 6.3 业务行为

#### 预定义大小
```go
var (
    SizeZero    = Size{bytes: 0}
    SizeByte    = Size{bytes: 1}
    SizeKB      = Size{bytes: 1024}
    SizeMB      = Size{bytes: 1024 * 1024}
    SizeGB      = Size{bytes: 1024 * 1024 * 1024}
)
```

#### 大小运算
```go
func (s Size) Add(other Size) Size {
    return Size{bytes: s.bytes + other.bytes}
}

func (s Size) Multiply(factor int64) Size {
    return Size{bytes: s.bytes * factor}
}

func (s Size) Divide(divisor int64) Size {
    return Size{bytes: s.bytes / divisor}
}

func (s Size) ToString() string {
    if s.bytes >= SizeGB.bytes {
        return fmt.Sprintf("%.2f GB", float64(s.bytes)/float64(SizeGB.bytes))
    } else if s.bytes >= SizeMB.bytes {
        return fmt.Sprintf("%.2f MB", float64(s.bytes)/float64(SizeMB.bytes))
    } else if s.bytes >= SizeKB.bytes {
        return fmt.Sprintf("%.2f KB", float64(s.bytes)/float64(SizeKB.bytes))
    }
    return fmt.Sprintf("%d bytes", s.bytes)
}
```

## 7. 值对象不变性保证

### 7.1 构造函数模式
```go
// 保证创建时就验证完整性
func NewGCConfiguration(triggerRatio float64, targetPause time.Duration) (GCConfiguration, error) {
    if triggerRatio <= 0 || triggerRatio >= 1 {
        return GCConfiguration{}, fmt.Errorf("invalid trigger ratio")
    }
    
    return GCConfiguration{
        triggerRatio:   triggerRatio,
        targetPauseTime: targetPause,
        // 其他字段使用默认值或参数
    }, nil
}
```

### 7.2 私有字段
```go
type immutableValue struct {
    value int  // 私有字段，只能通过构造函数设置
}

// 没有setter方法，保证不变性
func (v immutableValue) Value() int {
    return v.value
}
```

### 7.3 防御性复制
```go
type valueWithSlice struct {
    items []int
}

func NewValueWithSlice(items []int) valueWithSlice {
    // 防御性复制，防止外部修改
    copied := make([]int, len(items))
    copy(copied, items)
    
    return valueWithSlice{items: copied}
}

func (v valueWithSlice) Items() []int {
    // 返回副本，防止外部修改
    result := make([]int, len(v.items))
    copy(result, v.items)
    return result
}
```

## 8. 值对象测试模式

### 8.1 等价性测试
```go
func TestGCConfiguration_Equality(t *testing.T) {
    config1, _ := NewGCConfiguration(0.75, 10*time.Millisecond)
    config2, _ := NewGCConfiguration(0.75, 10*time.Millisecond)
    
    // 值对象通过值比较相等
    assert.True(t, config1.Equals(config2))
    
    // 修改一个属性后不相等
    config3, _ := NewGCConfiguration(0.8, 10*time.Millisecond)
    assert.False(t, config1.Equals(config3))
}
```

### 8.2 不变性测试
```go
func TestPerformanceMetrics_Immutability(t *testing.T) {
    metrics := NewPerformanceMetrics().
        WithPauseTimes(1*time.Millisecond, 5*time.Millisecond, 10*time.Millisecond, 100*time.Millisecond).
        Build()
    
    originalP95 := metrics.pauseTimeP95
    
    // 尝试修改（如果有setter，应该会编译错误）
    // metrics.pauseTimeP95 = 20*time.Millisecond  // 这行代码应该无法编译
    
    // 验证值未改变
    assert.Equal(t, originalP95, metrics.pauseTimeP95)
}
```
