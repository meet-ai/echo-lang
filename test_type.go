package main

import (
    "fmt"
    "github.com/llir/llvm/ir/types"
)

func main() {
    ptrType := types.NewPointer(types.I8)
    fmt.Println("Pointer type string:", ptrType.String())
}
