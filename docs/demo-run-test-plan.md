# Demo 运行测试计划

用于跟踪 `examples/*.eo` 在 `echoc run` 下的通过情况与修复进度。**进度按 demo 文件列表（demo_results.txt 的 FAIL 顺序）逐项跟踪。**

## 如何运行

```bash
# 需先构建
go build -o build/echoc ./cmd

# 全量跑 demo（超时 12s/个，跳过 *_error_test.eo）
bash run_demos_check.sh
```

- 结果写入 **demo_results.txt**，格式：`file,status,exit`
- 失败列表：`grep FAIL demo_results.txt`
- 查看某条失败原因：`./build/echoc run examples/<文件名>.eo 2>&1`

## 结果汇总（每次跑完脚本后更新）

| 项目      | 数值 |
| --------- | ---- |
| 运行日期  | 2026-02-11 |
| 总数      | 160（不含 _error_test） |
| OK        | 115 |
| FAIL      | 45 |
| 超时(124) | 2 |

## 失败项修复跟踪（按 demo 文件列表，每行一个 FAIL 文件）

下表行顺序与 `grep FAIL demo_results.txt` 一致，修一个更新一行。

| # | 文件 | 错误摘要 | 修复状态 | 备注 |
|---|------|----------|----------|------|
| 1 | examples/break_continue_test.eo | unexpected token in expression: while | 待修 | RDP 块内未解析 while |
| 2 | examples/comparison_test.eo | unexpected token in expression: if | 待修 | RDP 块内未解析 if |
| 3 | examples/complete_ast_test.eo | 超时 124 | 待修 | |
| 4 | examples/comprehensive_else_test.eo | 待补充 | 待修 | |
| 5 | examples/comprehensive_language_test_simple.eo | 待补充 | 待修 | |
| 6 | examples/control_flow_if_test.eo | 待补充 | 待修 | |
| 7 | examples/demo_generics.eo | 待补充 | 待修 | |
| 8 | examples/else_if_newline_separate_test.eo | 待补充 | 待修 | |
| 9 | examples/else_if_newline_test.eo | 待补充 | 待修 | |
| 10 | examples/error_handling_result_basic_test.eo | 待补充 | 待修 | |
| 11 | examples/generics_functions_basic_test.eo | 待补充 | 待修 | |
| 12 | examples/hello.eo | 超时 124 | 待修 | |
| 13 | examples/map_delete_test.eo | 待补充 | 待修 | |
| 14 | examples/map_iteration_test.eo | 待补充 | 待修 | |
| 15 | examples/math.eo | expected function body or ';' for abstract function | 待修 | |
| 16 | examples/methods_value_receiver_test.eo | 待补充 | 待修 | |
| 17 | examples/package_basic_test.eo | 待补充 | 待修 | |
| 18 | examples/package_cross_package_test.eo | 待补充 | 待修 | |
| 19 | examples/package_test.eo | 待补充 | 待修 | |
| 20 | examples/package_visibility_test.eo | 待补充 | 待修 | |
| 21 | examples/pattern_match_enum_test.eo | 待补充 | 待修 | |
| 22 | examples/test_associated_types_minimal.eo | 待补充 | 待修 | |
| 23 | examples/test_len_in_struct.eo | 待补充 | 待修 | |
| 24 | examples/trait_associated_types_full_test.eo | 待补充 | 待修 | |
| 25 | examples/trait_associated_types_minimal_test.eo | 待补充 | 待修 | |
| 26 | examples/trait_associated_types_simple_test.eo | 待补充 | 待修 | |
| 27 | examples/trait_associated_types_test.eo | 待补充 | 待修 | |
| 28 | examples/trait_basic_definition_test.eo | 待补充 | 待修 | |
| 29 | examples/trait_generic_definition_test.eo | 待补充 | 待修 | |
| 30 | examples/type_conversion_simple_runtime_test.eo | 待补充 | 待修 | |
| 31 | examples/type_conversion_simple_test.eo | 待补充 | 待修 | |
| 32 | examples/type_conversion_split_test.eo | 待补充 | 待修 | |
| 33 | examples/type_inference_array_method_test.eo | 待补充 | 待修 | |
| 34 | examples/type_inference_async_test.eo | 待补充 | 待修 | |
| 35 | examples/type_inference_binary_float_test.eo | 待补充 | 待修 | |
| 36 | examples/type_inference_binary_string_test.eo | 待补充 | 待修 | |
| 37 | examples/type_inference_field_access_test.eo | 待补充 | 待修 | |
| 38 | examples/type_inference_identifier_test.eo | 待补充 | 待修 | |
| 39 | examples/type_inference_index_test.eo | 待补充 | 待修 | |
| 40 | examples/type_inference_len_test.eo | 待补充 | 待修 | |
| 41 | examples/type_inference_method_call_test.eo | 待补充 | 待修 | |
| 42 | examples/type_inference_slice_test.eo | 待补充 | 待修 | |
| 43 | examples/type_inference_struct_test.eo | 待补充 | 待修 | |
| 44 | examples/types.eo | 待补充 | 待修 | |
| 45 | examples/utils.eo | 待补充 | 待修 | |

**错误摘要**：`./build/echoc run <文件> 2>&1` 首行或关键一句。

## 常见错误类型与对应动作

| 错误类型 | 可能原因 | 参考 |
| -------- | -------- | ---- |
| lexical: unterminated string | 词法字符串未闭合 | LexerService |
| syntax: expected '}' | print/return 等吞掉 `}` | SyntaxAnalyzer |
| syntax: unexpected token: while/if | 块内控制流未解析 | RDP parseBlockStatementInner 缺 if/while/for |
| syntax: unexpected token: spawn/await | 表达式未支持 | T-PARSE-001 |
| syntax: expected type name / variable type | 类型注解未支持（如 [T]、chan） | 已修 [T]，见 T-PARSE-001 |
| expected function body or ';' for abstract | trait/impl 抽象方法 | 语法/语义 |
| 超时(124) | 死循环或运行过久 | 看 demo 逻辑或调大 timeout |

## 更新流程

1. 执行 `bash run_demos_check.sh`，得到 **demo_results.txt**。
2. 用 `grep FAIL demo_results.txt` 得到失败列表；**失败项修复跟踪表按该顺序一行一个文件**，与 demo 文件列表一致。
3. 更新「结果汇总」：运行日期、OK/FAIL/超时数量（从 demo_results.txt 统计）。
4. 对表中每一行：跑 `./build/echoc run <文件> 2>&1`，把首行错误填到「错误摘要」；修好后「修复状态」改为「已修」并填「备注」。
5. 全部修完再跑一遍 `bash run_demos_check.sh`，用新 demo_results.txt 刷新本表（删除已过项、补充新 FAIL 行）。
