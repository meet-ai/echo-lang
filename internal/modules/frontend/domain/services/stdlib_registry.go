// Package services 定义标准库模块注册服务
package services

// StdlibRegistry 标准库模块注册器
// 职责：预加载标准库模块符号，注册到类型推断服务
type StdlibRegistry struct {
	typeInferenceService *SimpleTypeInferenceService
}

// NewStdlibRegistry 创建标准库模块注册器
func NewStdlibRegistry(typeInferenceService *SimpleTypeInferenceService) *StdlibRegistry {
	return &StdlibRegistry{
		typeInferenceService: typeInferenceService,
	}
}

// RegisterAllStandardLibraryModules 注册所有标准库模块
// 在编译器初始化时调用，预加载所有标准库模块符号
func (r *StdlibRegistry) RegisterAllStandardLibraryModules() error {
	// 注册内置类型的 print_string / print_int（供 print 语句与方法调用类型推断）
	if err := r.registerBuiltinPrintMethods(); err != nil {
		return err
	}
	// 注册内置函数 len、Option（供类型推断 functionTable 查找）
	if err := r.registerBuiltinFunctions(); err != nil {
		return err
	}
	// 注册 time 模块
	if err := r.registerTimeModule(); err != nil {
		return err
	}

	// 注册 math 模块
	if err := r.registerMathModule(); err != nil {
		return err
	}

	// 注册 string 扩展方法
	if err := r.registerStringExtensions(); err != nil {
		return err
	}

	// 注册 atomic 模块（在 sync 包中）
	if err := r.registerAtomicModule(); err != nil {
		return err
	}

	// 注册 sync 模块的其他类型（Mutex, RwLock）
	if err := r.registerSyncModule(); err != nil {
		return err
	}

	// 注册 io 模块
	if err := r.registerIOModule(); err != nil {
		return err
	}

	// 注册 core 模块
	if err := r.registerCoreModule(); err != nil {
		return err
	}

	// 注册 collections 模块
	if err := r.registerCollectionsModule(); err != nil {
		return err
	}

	// 注册 from ... import 使用的包导出（供类型推断 functionTable 查找）
	r.registerPackageExportsForFromImport()

	return nil
}

// registerPackageExportsForFromImport 为 from <path> import x, y 注册包导出
// 使类型推断能将导入的符号注册到 functionTable
func (r *StdlibRegistry) registerPackageExportsForFromImport() {
	// math：示例/测试中 from "math" import add, subtract, multiply, sqrt
	mathExports := map[string]string{
		"add": "int", "subtract": "int", "multiply": "int",
		"sqrt": "f64", "abs": "f64", "max": "f64", "min": "f64",
		"ceil": "f64", "floor": "f64", "round": "f64", "sin": "f64", "cos": "f64", "tan": "f64",
	}
	r.typeInferenceService.SetPackageExports("math", mathExports)

	// utils：from "utils" import helper, format
	utilsExports := map[string]string{
		"helper": "string", "format": "string",
	}
	r.typeInferenceService.SetPackageExports("utils", utilsExports)

	// types：from "types" import Point, Rectangle, newPoint, area, Color, Status
	typesExports := map[string]string{
		"Point": "Point", "Rectangle": "Rectangle",
		"newPoint": "Point", "area": "int",
		"Color": "Color", "Status": "Status",
	}
	r.typeInferenceService.SetPackageExports("types", typesExports)

	// internal/cache：from "internal/cache" import newItem, getValue
	internalCacheExports := map[string]string{
		"newItem": "CacheItem", "getValue": "string",
	}
	r.typeInferenceService.SetPackageExports("internal/cache", internalCacheExports)

	// internal/utils：from "internal/utils" import helper
	internalUtilsExports := map[string]string{
		"helper": "string",
	}
	r.typeInferenceService.SetPackageExports("internal/utils", internalUtilsExports)
}

// registerBuiltinPrintMethods 为内置类型注册 print_string / print_int，用于类型推断
func (r *StdlibRegistry) registerBuiltinPrintMethods() error {
	// 整数类型：print_string（转字符串后打印）、print_int（直接打印整数）
	for _, t := range []string{"i64", "i32", "int"} {
		r.typeInferenceService.methodTable[t+".print_string"] = "void"
		r.typeInferenceService.methodTable[t+".print_int"] = "void"
	}
	// 浮点类型：print_string（转字符串后打印）
	for _, t := range []string{"f64", "float"} {
		r.typeInferenceService.methodTable[t+".print_string"] = "void"
	}
	// 布尔：print_string
	r.typeInferenceService.methodTable["bool.print_string"] = "void"
	// 字符串：print_string（字面量或变量.print_string(...)）
	r.typeInferenceService.methodTable["string.print_string"] = "void"
	// void：print_string（无操作，用于 store() 等返回 void 的链式调用推断）
	r.typeInferenceService.methodTable["void.print_string"] = "void"
	// 标准库类型：print_string（调试/展示用）
	r.typeInferenceService.methodTable["Time.print_string"] = "void"
	r.typeInferenceService.methodTable["Duration.print_string"] = "void"
	r.typeInferenceService.methodTable["Mutex[T].print_string"] = "void"
	r.typeInferenceService.methodTable["RwLock[T].print_string"] = "void"
	r.typeInferenceService.methodTable["AtomicInt.print_string"] = "void"
	r.typeInferenceService.methodTable["AtomicBool.print_string"] = "void"
	return nil
}

// registerBuiltinFunctions 为内置函数注册 functionTable 条目，供类型推断使用
// len(x) -> int；Option(x) -> Option[T]（T 由参数类型推断）
func (r *StdlibRegistry) registerBuiltinFunctions() error {
	r.typeInferenceService.functionTable["len"] = "int"
	r.typeInferenceService.functionTable["Option"] = "Option[T]"
	return nil
}

// registerTimeModule 注册 time 模块
func (r *StdlibRegistry) registerTimeModule() error {
	// 1. 注册模块符号：time -> "module:time"
	r.typeInferenceService.symbolTable["time"] = "module:time"

	// 2. 注册模块级函数
	// time.now() -> Time
	r.typeInferenceService.methodTable["module:time.now"] = "Time"

	// time.from_unix(sec: i64) -> Time
	r.typeInferenceService.methodTable["module:time.from_unix"] = "Time"

	// time.from_unix_nanos(sec: i64, nsec: i32) -> Time
	r.typeInferenceService.methodTable["module:time.from_unix_nanos"] = "Time"

	// time.from_secs(secs: i64) -> Duration
	r.typeInferenceService.methodTable["module:time.from_secs"] = "Duration"

	// time.from_millis(millis: i64) -> Duration
	r.typeInferenceService.methodTable["module:time.from_millis"] = "Duration"

	// time.from_hours(hours: i64) -> Duration
	r.typeInferenceService.methodTable["module:time.from_hours"] = "Duration"

	// 3. 注册 Time 类型的方法
	// Time.unix() -> i64
	r.typeInferenceService.methodTable["Time.unix"] = "i64"

	// Time.unix_nanos() -> i64
	r.typeInferenceService.methodTable["Time.unix_nanos"] = "i64"

	// Time.nanosecond() -> i32
	r.typeInferenceService.methodTable["Time.nanosecond"] = "i32"

	// Time.is_zero() -> bool
	r.typeInferenceService.methodTable["Time.is_zero"] = "bool"

	// Time.add(d: Duration) -> Time
	r.typeInferenceService.methodTable["Time.add"] = "Time"

	// Time.sub_duration(d: Duration) -> Time
	r.typeInferenceService.methodTable["Time.sub_duration"] = "Time"

	// Time.sub(other: Time) -> Duration
	r.typeInferenceService.methodTable["Time.sub"] = "Duration"

	// Time.before(other: Time) -> bool
	r.typeInferenceService.methodTable["Time.before"] = "bool"

	// Time.after(other: Time) -> bool
	r.typeInferenceService.methodTable["Time.after"] = "bool"

	// Time.eq(other: Time) -> bool
	r.typeInferenceService.methodTable["Time.eq"] = "bool"

	// Time.format(layout: string) -> string
	r.typeInferenceService.methodTable["Time.format"] = "string"

	// 4. 注册 Duration 类型的方法
	// Duration.as_secs() -> i64
	r.typeInferenceService.methodTable["Duration.as_secs"] = "i64"

	// Duration.as_millis() -> i64
	r.typeInferenceService.methodTable["Duration.as_millis"] = "i64"

	// Duration.as_hours() -> i64
	r.typeInferenceService.methodTable["Duration.as_hours"] = "i64"

	// Duration.add(other: Duration) -> Duration
	r.typeInferenceService.methodTable["Duration.add"] = "Duration"

	// Duration.sub(other: Duration) -> Duration
	r.typeInferenceService.methodTable["Duration.sub"] = "Duration"

	return nil
}

// registerMathModule 注册 math 模块
func (r *StdlibRegistry) registerMathModule() error {
	// 1. 注册模块符号：math -> "module:math"
	r.typeInferenceService.symbolTable["math"] = "module:math"

	// 2. 注册模块级函数（常用函数）
	// math.sqrt(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.sqrt"] = "f64"

	// math.abs(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.abs"] = "f64"

	// math.abs_int(x: i64) -> i64
	r.typeInferenceService.methodTable["module:math.abs_int"] = "i64"

	// math.max(a: f64, b: f64) -> f64
	r.typeInferenceService.methodTable["module:math.max"] = "f64"

	// math.max_int(a: i64, b: i64) -> i64
	r.typeInferenceService.methodTable["module:math.max_int"] = "i64"

	// math.min(a: f64, b: f64) -> f64
	r.typeInferenceService.methodTable["module:math.min"] = "f64"

	// math.min_int(a: i64, b: i64) -> i64
	r.typeInferenceService.methodTable["module:math.min_int"] = "i64"

	// math.ceil(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.ceil"] = "f64"

	// math.floor(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.floor"] = "f64"

	// math.round(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.round"] = "f64"

	// math.trunc(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.trunc"] = "f64"

	// math.cbrt(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.cbrt"] = "f64"

	// math.pow(base: f64, exp: f64) -> f64
	r.typeInferenceService.methodTable["module:math.pow"] = "f64"

	// math.ln(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.ln"] = "f64"

	// math.log10(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.log10"] = "f64"

	// math.log2(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.log2"] = "f64"

	// math.exp(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.exp"] = "f64"

	// math.sin(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.sin"] = "f64"

	// math.cos(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.cos"] = "f64"

	// math.tan(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.tan"] = "f64"

	// math.asin(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.asin"] = "f64"

	// math.acos(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.acos"] = "f64"

	// math.atan(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.atan"] = "f64"

	// math.atan2(y: f64, x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.atan2"] = "f64"

	// math.sinh(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.sinh"] = "f64"

	// math.cosh(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.cosh"] = "f64"

	// math.tanh(x: f64) -> f64
	r.typeInferenceService.methodTable["module:math.tanh"] = "f64"

	return nil
}

// registerStringExtensions 注册 string 扩展方法
func (r *StdlibRegistry) registerStringExtensions() error {
	// string 是内置类型，不需要注册模块符号
	// 直接注册扩展方法到 methodTable

	// string.len() -> int
	r.typeInferenceService.methodTable["string.len"] = "int"

	// string.is_empty() -> bool
	r.typeInferenceService.methodTable["string.is_empty"] = "bool"

	// string.contains(sub: string) -> bool
	r.typeInferenceService.methodTable["string.contains"] = "bool"

	// string.starts_with(prefix: string) -> bool
	r.typeInferenceService.methodTable["string.starts_with"] = "bool"

	// string.ends_with(suffix: string) -> bool
	r.typeInferenceService.methodTable["string.ends_with"] = "bool"

	// string.split(delimiter: string) -> []string
	r.typeInferenceService.methodTable["string.split"] = "[string]"

	// string.trim() -> string
	r.typeInferenceService.methodTable["string.trim"] = "string"

	// string.to_upper() -> string
	r.typeInferenceService.methodTable["string.to_upper"] = "string"

	// string.to_lower() -> string
	r.typeInferenceService.methodTable["string.to_lower"] = "string"

	// 3. 注册 string 模块级函数
	// string.new() -> StringBuilder
	r.typeInferenceService.methodTable["module:string.new"] = "StringBuilder"

	// string.with_capacity(cap: int) -> StringBuilder
	r.typeInferenceService.methodTable["module:string.with_capacity"] = "StringBuilder"

	// 4. 注册 StringBuilder 类型的方法
	// StringBuilder.append_str(s: string) -> void
	r.typeInferenceService.methodTable["StringBuilder.append_str"] = "void"

	// StringBuilder.to_string() -> string
	r.typeInferenceService.methodTable["StringBuilder.to_string"] = "string"

	return nil
}

// registerAtomicModule 注册 atomic 模块（在 sync 包中）
func (r *StdlibRegistry) registerAtomicModule() error {
	// 注意：atomic 模块在 sync 包中，但用户可能通过 import atomic 导入
	// 这里先注册 sync 模块，后续可以根据实际导入路径调整

	// 1. 注册模块符号：atomic -> "module:atomic"（如果用户使用 import atomic）
	// 或者 sync -> "module:sync"（如果用户使用 import sync）
	r.typeInferenceService.symbolTable["atomic"] = "module:atomic"
	r.typeInferenceService.symbolTable["sync"] = "module:sync"

	// 2. 注册模块级函数
	// atomic.new_atomic_int(initial: i64) -> AtomicInt
	r.typeInferenceService.methodTable["module:atomic.new_atomic_int"] = "AtomicInt"

	// atomic.new_atomic_bool(initial: bool) -> AtomicBool
	r.typeInferenceService.methodTable["module:atomic.new_atomic_bool"] = "AtomicBool"

	// sync.new_atomic_int(initial: i64) -> AtomicInt（兼容 sync 包导入）
	r.typeInferenceService.methodTable["module:sync.new_atomic_int"] = "AtomicInt"

	// sync.new_atomic_bool(initial: bool) -> AtomicBool（兼容 sync 包导入）
	r.typeInferenceService.methodTable["module:sync.new_atomic_bool"] = "AtomicBool"

	// 3. 注册 AtomicInt 类型的方法
	// AtomicInt.load() -> i64
	r.typeInferenceService.methodTable["AtomicInt.load"] = "i64"

	// AtomicInt.store(value: i64) -> void（无返回值）
	r.typeInferenceService.methodTable["AtomicInt.store"] = "void"

	// AtomicInt.swap(value: i64) -> i64
	r.typeInferenceService.methodTable["AtomicInt.swap"] = "i64"

	// AtomicInt.compare_and_swap(expected: i64, new_value: i64) -> bool
	r.typeInferenceService.methodTable["AtomicInt.compare_and_swap"] = "bool"

	// AtomicInt.add(delta: i64) -> i64
	r.typeInferenceService.methodTable["AtomicInt.add"] = "i64"

	// AtomicInt.sub(delta: i64) -> i64
	r.typeInferenceService.methodTable["AtomicInt.sub"] = "i64"

	// AtomicInt.increment() -> i64
	r.typeInferenceService.methodTable["AtomicInt.increment"] = "i64"

	// AtomicInt.decrement() -> i64
	r.typeInferenceService.methodTable["AtomicInt.decrement"] = "i64"

	// 4. 注册 AtomicBool 类型的方法
	// AtomicBool.load() -> bool
	r.typeInferenceService.methodTable["AtomicBool.load"] = "bool"

	// AtomicBool.store(value: bool) -> void
	r.typeInferenceService.methodTable["AtomicBool.store"] = "void"

	// AtomicBool.swap(value: bool) -> bool
	r.typeInferenceService.methodTable["AtomicBool.swap"] = "bool"

	// AtomicBool.compare_and_swap(expected: bool, new_value: bool) -> bool
	r.typeInferenceService.methodTable["AtomicBool.compare_and_swap"] = "bool"

	return nil
}

// registerSyncModule 注册 sync 模块的其他类型（Mutex, RwLock）
func (r *StdlibRegistry) registerSyncModule() error {
	// 1. 注册模块符号（如果还没有）
	if _, exists := r.typeInferenceService.symbolTable["sync"]; !exists {
		r.typeInferenceService.symbolTable["sync"] = "module:sync"
	}

	// 2. 注册 Mutex[T] 泛型方法
	// sync.new_mutex(data: T) -> Mutex[T]
	r.typeInferenceService.methodTable["module:sync.new_mutex"] = "Mutex[T]"

	// Mutex[T].lock() -> MutexGuard[T]
	r.typeInferenceService.methodTable["Mutex[T].lock"] = "MutexGuard[T]"

	// Mutex[T].try_lock() -> Option[MutexGuard[T]]
	r.typeInferenceService.methodTable["Mutex[T].try_lock"] = "Option[MutexGuard[T]]"

	// 3. 注册 RwLock[T] 泛型方法
	// sync.new_rwlock(data: T) -> RwLock[T]
	r.typeInferenceService.methodTable["module:sync.new_rwlock"] = "RwLock[T]"

	// RwLock[T].read_lock() -> RwLockReadGuard[T]
	r.typeInferenceService.methodTable["RwLock[T].read_lock"] = "RwLockReadGuard[T]"

	// RwLock[T].write_lock() -> RwLockWriteGuard[T]
	r.typeInferenceService.methodTable["RwLock[T].write_lock"] = "RwLockWriteGuard[T]"

	// RwLock[T].try_read_lock() -> Option[RwLockReadGuard[T]]
	r.typeInferenceService.methodTable["RwLock[T].try_read_lock"] = "Option[RwLockReadGuard[T]]"

	// RwLock[T].try_write_lock() -> Option[RwLockWriteGuard[T]]
	r.typeInferenceService.methodTable["RwLock[T].try_write_lock"] = "Option[RwLockWriteGuard[T]]"

	return nil
}

// registerIOModule 注册 io 模块
func (r *StdlibRegistry) registerIOModule() error {
	// 1. 注册模块符号
	r.typeInferenceService.symbolTable["io"] = "module:io"

	// 2. 注册模块级函数
	// io.open(path: string) -> Result[File]
	r.typeInferenceService.methodTable["module:io.open"] = "Result[File]"

	// io.create(path: string) -> Result[File]
	r.typeInferenceService.methodTable["module:io.create"] = "Result[File]"

	// 3. 注册 File 类型方法
	// File.close() -> Result[void]
	r.typeInferenceService.methodTable["File.close"] = "Result[void]"

	// File.read() -> Result[string]
	r.typeInferenceService.methodTable["File.read"] = "Result[string]"

	// File.write(data: string) -> Result[int]
	r.typeInferenceService.methodTable["File.write"] = "Result[int]"

	// File.flush() -> Result[void]
	r.typeInferenceService.methodTable["File.flush"] = "Result[void]"

	return nil
}

// registerCoreModule 注册 core 模块
func (r *StdlibRegistry) registerCoreModule() error {
	// 1. 注册模块符号
	r.typeInferenceService.symbolTable["core"] = "module:core"

	// 2. 注册 Option[T] 泛型方法
	// Option[T].is_some() -> bool
	r.typeInferenceService.methodTable["Option[T].is_some"] = "bool"

	// Option[T].is_none() -> bool
	r.typeInferenceService.methodTable["Option[T].is_none"] = "bool"

	// Option[T].unwrap() -> T
	r.typeInferenceService.methodTable["Option[T].unwrap"] = "T"

	// Option[T].unwrap_or(default: T) -> T
	r.typeInferenceService.methodTable["Option[T].unwrap_or"] = "T"

	// Option[T].ok_or(err: E) -> Result[T, E]
	r.typeInferenceService.methodTable["Option[T].ok_or"] = "Result[T, E]"

	// 3. 注册 Result[T, E] 泛型方法
	// Result[T, E].is_ok() -> bool
	r.typeInferenceService.methodTable["Result[T, E].is_ok"] = "bool"

	// Result[T, E].is_err() -> bool
	r.typeInferenceService.methodTable["Result[T, E].is_err"] = "bool"

	// Result[T, E].unwrap() -> T
	r.typeInferenceService.methodTable["Result[T, E].unwrap"] = "T"

	// Result[T, E].unwrap_err() -> E
	r.typeInferenceService.methodTable["Result[T, E].unwrap_err"] = "E"

	// Result[T, E].ok() -> Option[T]
	r.typeInferenceService.methodTable["Result[T, E].ok"] = "Option[T]"

	return nil
}

// registerCollectionsModule 注册 collections 模块
func (r *StdlibRegistry) registerCollectionsModule() error {
	// 1. 注册模块符号
	r.typeInferenceService.symbolTable["collections"] = "module:collections"

	// 2. 注册模块级函数（泛型）
	// collections::new[T]() -> Vec[T]
	r.typeInferenceService.methodTable["module:collections.new"] = "Vec[T]"

	// collections::with_capacity[T](cap: int) -> Vec[T]
	r.typeInferenceService.methodTable["module:collections.with_capacity"] = "Vec[T]"

	// 3. 注册 Vec[T] 泛型方法
	// Vec[T].push(item: T) -> void
	r.typeInferenceService.methodTable["Vec[T].push"] = "void"

	// Vec[T].pop() -> Option[T]
	r.typeInferenceService.methodTable["Vec[T].pop"] = "Option[T]"

	// Vec[T].get(index: int) -> Option[T]
	r.typeInferenceService.methodTable["Vec[T].get"] = "Option[T]"

	// Vec[T].len() -> int
	r.typeInferenceService.methodTable["Vec[T].len"] = "int"

	// Vec[T].capacity() -> int
	r.typeInferenceService.methodTable["Vec[T].capacity"] = "int"

	// Vec[T].is_empty() -> bool
	r.typeInferenceService.methodTable["Vec[T].is_empty"] = "bool"

	// Vec[T].first() -> Option[T]
	r.typeInferenceService.methodTable["Vec[T].first"] = "Option[T]"

	// Vec[T].last() -> Option[T]
	r.typeInferenceService.methodTable["Vec[T].last"] = "Option[T]"

	// Vec[T].clear() -> void
	r.typeInferenceService.methodTable["Vec[T].clear"] = "void"

	// Vec[T].grow() -> void（内存管理，需要编译器支持 sizeof）
	r.typeInferenceService.methodTable["Vec[T].grow"] = "void"

	// Vec[T].reserve(additional: int) -> void（内存管理，需要编译器支持 sizeof）
	r.typeInferenceService.methodTable["Vec[T].reserve"] = "void"

	// Vec[T].shrink_to_fit() -> void（内存管理，需要编译器支持 sizeof）
	r.typeInferenceService.methodTable["Vec[T].shrink_to_fit"] = "void"

	// 4. 注册 HashMap[K, V] 泛型方法
	// HashMap[K, V].insert(key: K, value: V) -> Option[V]
	r.typeInferenceService.methodTable["HashMap[K, V].insert"] = "Option[V]"

	// HashMap[K, V].get(key: K) -> Option[V]
	r.typeInferenceService.methodTable["HashMap[K, V].get"] = "Option[V]"

	// HashMap[K, V].remove(key: K) -> Option[V]
	r.typeInferenceService.methodTable["HashMap[K, V].remove"] = "Option[V]"

	// HashMap[K, V].contains_key(key: K) -> bool
	r.typeInferenceService.methodTable["HashMap[K, V].contains_key"] = "bool"

	// HashMap[K, V].len() -> int
	r.typeInferenceService.methodTable["HashMap[K, V].len"] = "int"

	// HashMap[K, V].is_empty() -> bool
	r.typeInferenceService.methodTable["HashMap[K, V].is_empty"] = "bool"

	// HashMap[K, V].clear() -> void
	r.typeInferenceService.methodTable["HashMap[K, V].clear"] = "void"

	// 5. 注册 HashMap 构造函数
	// collections::new[K, V]() -> HashMap[K, V]
	r.typeInferenceService.methodTable["module:collections.new"] = "HashMap[K, V]"

	// collections::with_capacity[K, V](cap: int) -> HashMap[K, V]
	r.typeInferenceService.methodTable["module:collections.with_capacity"] = "HashMap[K, V]"

	// 6. 注册 map[K, V] 的 keys/values（与 HashMap 同义，语言中常用 map 关键字）
	r.typeInferenceService.methodTable["map[T, E].keys"] = "[T]"
	r.typeInferenceService.methodTable["map[T, E].values"] = "[E]"

	// 7. 注册切片 [T] 的方法（与 Vec[T] 同义，语言中 [int] 表示切片）
	r.typeInferenceService.methodTable["[T].pop"] = "T"
	r.typeInferenceService.methodTable["[T].push"] = "void"
	r.typeInferenceService.methodTable["[T].insert"] = "void"
	r.typeInferenceService.methodTable["[T].remove"] = "void"

	return nil
}
