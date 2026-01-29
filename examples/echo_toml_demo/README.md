# echo.toml 使用演示

这是一个完整的示例项目，演示如何使用 `echo.toml` 配置文件管理依赖包。

## 项目结构

```
echo_toml_demo/
├── echo.toml              # 项目配置文件
├── README.md              # 本文件
├── src/
│   └── main.eo            # 主程序
└── vendor/                # 依赖包目录
    ├── mathlib/           # 依赖包1（默认路径）
    │   └── mathlib.eo
    ├── http/              # 依赖包2（自定义路径）
    │   └── http.eo
    └── utils/             # 依赖包3（默认路径）
        └── utils.eo
```

## 配置文件说明

### echo.toml

```toml
name = "echo-toml-demo"
version = "1.0.0"

[package]
entry = "src/main.eo"
module = "demo"

[dependencies]
"mathlib" = "2.1.0"                                    # 默认路径：vendor/mathlib
"network/http" = { version = "1.0.0", path = "vendor/http" }  # 自定义路径
"utils" = { version = "0.5.0" }                       # 默认路径：vendor/utils
```

### 依赖包配置说明

1. **mathlib**：使用简单格式，默认路径 `vendor/mathlib`
2. **network/http**：使用详细格式，自定义路径 `vendor/http`
3. **utils**：使用详细格式，默认路径 `vendor/utils`

## 代码示例

### src/main.eo

```eo
package main

import mathlib
import "network/http" as HttpLib
from "utils" import helper, format

func main() {
    // 使用 mathlib
    let sum = mathlib.add(10, 20)
    
    // 使用 network/http
    HttpLib.get("https://example.com")
    
    // 使用 utils
    let helperValue = helper()
    let formatted = format("测试消息")
}
```

## 包查找流程

当代码中使用 `import mathlib` 时：

1. 首先查找项目源码：`src/mathlib/` 或 `src/mathlib.eo`
2. 如果找不到，从 `echo.toml` 读取依赖配置
3. 检查 `dependencies` 中是否有 `"mathlib"`
4. 根据配置解析路径：
   - 如果配置了 `path`：使用配置的路径
   - 如果没有配置 `path`：使用默认路径 `vendor/mathlib`
5. 查找包文件：
   - 优先查找 `vendor/mathlib.eo`（单文件包）
   - 如果不存在，查找 `vendor/mathlib/`（目录包）
   - 如果目录存在，查找 `vendor/mathlib/mathlib.eo`

## 运行方式

```bash
# 编译项目
echoc build src/main.eo

# 包管理器会自动：
# 1. 读取 echo.toml 配置
# 2. 解析依赖包路径
# 3. 加载依赖包
```

## 注意事项

1. **配置文件位置**：`echo.toml` 必须放在项目根目录
2. **依赖包路径**：依赖包必须放在配置的路径下（或默认路径 `vendor/`）
3. **包文件命名**：
   - 单文件包：`{包名}.eo`
   - 目录包：`{包名}/{包名}.eo`
4. **版本号**：目前仅用于记录，不进行版本检查

## 相关文档

- **快速使用指南**：`docs/echo.toml-快速使用指南.md`
- **完整文档**：`docs/依赖包管理使用指南.md`
- **配置示例**：`examples/echo.toml.example`
