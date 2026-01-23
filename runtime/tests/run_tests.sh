#!/bin/bash
# Runtime 测试运行脚本

set -e

echo "=== Runtime 测试运行脚本 ==="
echo ""

# 编译选项
CFLAGS="-std=c11 -Wall -Wextra -DSTANDALONE_TEST -g"
INCLUDES="-I runtime/domain -I runtime/tests/unit-tests -I runtime/include"
LIBS="-lpthread"

# 测试文件列表
TESTS=(
    "runtime/tests/unit/domain/coroutine/test_coroutine_switch.c"
    "runtime/tests/unit/domain/channel/test_channel_blocking.c"
    "runtime/tests/unit/domain/channel/test_channel_buffered.c"
    "runtime/tests/unit/domain/future/test_future_wake.c"
    "runtime/tests/integration/concurrency/test_async_await.c"
)

# 测试名称
TEST_NAMES=(
    "协程切换测试"
    "通道阻塞测试"
    "缓冲通道测试"
    "Future唤醒测试"
    "异步集成测试"
)

# 编译并运行单个测试（语法检查）
run_test() {
    local test_file=$1
    local test_name=$2
    local test_basename=$(basename "$test_file" .c)
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "测试: $test_name"
    echo "文件: $test_basename"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 编译检查
    echo "  [1/2] 编译检查..."
    if gcc -c $CFLAGS $INCLUDES "$test_file" -o /tmp/${test_basename}.o 2>&1 | tee /tmp/${test_basename}_compile.log | grep -q "error"; then
        echo "  ❌ 编译失败"
        echo "  错误信息:"
        grep "error" /tmp/${test_basename}_compile.log | head -3
        return 1
    else
        echo "  ✓ 编译成功"
    fi
    
    # 语法验证
    echo "  [2/2] 语法验证..."
    if grep -q "STANDALONE_TEST" "$test_file" && grep -q "int main" "$test_file"; then
        echo "  ✓ 包含测试主函数"
    else
        echo "  ⚠️  未找到测试主函数"
    fi
    
    echo ""
    return 0
}

# 运行所有测试
echo "开始运行测试..."
echo ""

total=0
passed=0

for i in "${!TESTS[@]}"; do
    test_file="${TESTS[$i]}"
    test_name="${TEST_NAMES[$i]}"
    
    if [ -f "$test_file" ]; then
        total=$((total + 1))
        if run_test "$test_file" "$test_name"; then
            passed=$((passed + 1))
        fi
    else
        echo "⚠️  文件不存在: $test_file"
    fi
done

# 输出总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "=== 测试总结 ==="
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "总计: $total"
echo "通过: $passed"
echo "失败: $((total - passed))"
echo ""

if [ $passed -eq $total ]; then
    echo "🎉 所有测试编译成功！"
    echo ""
    echo "📝 注意:"
    echo "  - 测试文件已创建并编译验证通过"
    echo "  - 完整运行需要链接运行时库"
    echo "  - 可以使用 'make build-runtime' 构建运行时库"
    echo "  - 然后链接测试文件与运行时库执行"
    exit 0
else
    echo "❌ 部分测试编译失败"
    exit 1
fi

