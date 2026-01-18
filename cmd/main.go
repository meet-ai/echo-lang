package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/meetai/echo-lang/internal/modules/backend/domain/services"
	"github.com/meetai/echo-lang/internal/modules/backend/infrastructure/codegen"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/entities"
	frontendServices "github.com/meetai/echo-lang/internal/modules/frontend/domain/services"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "build":
		handleBuildCommand(os.Args[2:])
	case "run":
		handleRunCommand(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		// 兼容旧的直接编译模式
		handleLegacyCommand(os.Args[1:])
	}
}

func printUsage() {
	fmt.Println("Echo Language Compiler")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  echoc build <file.eo> [options]    - Build/compile an Echo source file")
	fmt.Println("  echoc run <file.eo> [options]      - Compile and run an Echo source file")
	fmt.Println("  echoc help                         - Show this help message")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -backend string    Backend to use: llvm-ir (default), ocaml")
	fmt.Println("  -target string     Target output: ir, executable (default)")
	fmt.Println("  -runtime string    Runtime implementation: ucontext (default), asm")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  echoc build examples/hello.eo")
	fmt.Println("  echoc build examples/hello.eo -backend=llvm-ir -target=executable")
	fmt.Println("  echoc run examples/hello.eo")
	fmt.Println("  echoc run examples/hello.eo -backend=llvm-ir")
}

func handleBuildCommand(args []string) {
	var backend = "llvm-ir"
	var target = "ir"
	var runtime = "ucontext" // 默认使用ucontext实现
	var inputFile string

	// 解析命令行参数
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-backend" && i+1 < len(args) {
			backend = args[i+1]
			i++
		} else if arg == "-target" && i+1 < len(args) {
			target = args[i+1]
			i++
		} else if arg == "-runtime" && i+1 < len(args) {
			runtime = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "-backend=") {
			backend = strings.TrimPrefix(arg, "-backend=")
		} else if strings.HasPrefix(arg, "-target=") {
			target = strings.TrimPrefix(arg, "-target=")
		} else if strings.HasPrefix(arg, "-runtime=") {
			runtime = strings.TrimPrefix(arg, "-runtime=")
		} else if !strings.HasPrefix(arg, "-") {
			// 非选项参数，应该是文件名
			if inputFile != "" {
				fmt.Fprintf(os.Stderr, "Usage: echoc build <file.eo> [options]\n")
				os.Exit(1)
			}
			inputFile = arg
		}
	}

	// 检查是否提供了文件名
	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: echoc build <file.eo> [options]\n")
		os.Exit(1)
	}

	fmt.Printf("Building %s with backend=%s, target=%s, runtime=%s\n", inputFile, backend, target, runtime)

	// 读取文件内容
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	// 创建解析器
	parser := frontendServices.NewSimpleParser()

	// 解析文件
	program, err := parser.Parse(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// 创建代码生成器
	var codeGenerator services.CodeGenerator

	switch backend {
	case "ocaml":
		codeGenerator = codegen.NewCodeGenerator("ocaml")
	case "llvm-ir":
		codeGenerator = codegen.NewCodeGenerator("llvm")
	default:
		fmt.Fprintf(os.Stderr, "Unknown backend: %s\n", backend)
		os.Exit(1)
	}

	// 生成代码
	if target == "executable" {
		output, err := generateExecutable(program, codeGenerator, backend, runtime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating executable: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(output)
	} else {
		// 生成代码并输出
		output := codeGenerator.GenerateCode(program)
		fmt.Print(output)
	}
}

func handleRunCommand(args []string) {
	var backend = "llvm-ir"
	var runtime = "ucontext" // 默认使用ucontext实现
	var inputFile string

	// 解析命令行参数
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-backend" && i+1 < len(args) {
			backend = args[i+1]
			i++
		} else if arg == "-runtime" && i+1 < len(args) {
			runtime = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "-backend=") {
			backend = strings.TrimPrefix(arg, "-backend=")
		} else if strings.HasPrefix(arg, "-runtime=") {
			runtime = strings.TrimPrefix(arg, "-runtime=")
		} else if !strings.HasPrefix(arg, "-") {
			// 非选项参数，应该是文件名
			if inputFile != "" {
				fmt.Fprintf(os.Stderr, "Usage: echoc run <file.eo> [options]\n")
				os.Exit(1)
			}
			inputFile = arg
		}
	}

	// 检查是否提供了文件名
	if inputFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: echoc run <file.eo> [options]\n")
		os.Exit(1)
	}

	fmt.Printf("Running %s with backend=%s, runtime=%s\n", inputFile, backend, runtime)

	// 读取文件内容
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	// 创建解析器
	parser := frontendServices.NewSimpleParser()

	// 解析文件
	program, err := parser.Parse(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// 创建代码生成器
	var codeGenerator services.CodeGenerator

	switch backend {
	case "ocaml":
		codeGenerator = codegen.NewCodeGenerator("ocaml")
		// OCaml 后端：生成可执行文件然后运行
		fmt.Println("OCaml backend not yet implemented for run command")
		os.Exit(1)
	case "llvm-ir":
		codeGenerator = codegen.NewCodeGenerator("llvm")
	default:
		fmt.Fprintf(os.Stderr, "Unknown backend: %s\n", backend)
		os.Exit(1)
	}

	// 生成可执行文件
	_, err = generateExecutable(program, codeGenerator, backend, runtime)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating executable: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated executable successfully")

	// 运行可执行文件
	execPath := "program" // 这是 generateLLVMExecutable 生成的默认文件名
	fmt.Printf("Executing: %s\n", execPath)

	cmd := exec.Command("./" + execPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running executable: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Execution completed")
}

func handleLegacyCommand(args []string) {
	// 兼容旧的直接编译模式（不带子命令）
	var backend = flag.String("backend", "llvm-ir", "Backend to use: ocaml, llvm-ir")
	var target = flag.String("target", "code", "Target output: code, executable")
	var runtime = flag.String("runtime", "ucontext", "Runtime implementation: ucontext, asm")
	flag.CommandLine.Parse(args)

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <input.eo>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Try 'echoc help' for more information.\n")
		os.Exit(1)
	}

	inputFile := flag.Arg(0)

	// 读取文件内容
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", inputFile, err)
		os.Exit(1)
	}

	// 创建解析器
	parser := frontendServices.NewSimpleParser()

	// 解析文件
	program, err := parser.Parse(string(content))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// 创建代码生成器
	var codeGenerator services.CodeGenerator
	var outputFormat string

	switch *backend {
	case "ocaml":
		codeGenerator = codegen.NewCodeGenerator("ocaml")
		outputFormat = "OCaml Code"
	case "llvm-ir":
		codeGenerator = codegen.NewCodeGenerator("llvm")
		outputFormat = "LLVM IR"
	default:
		fmt.Fprintf(os.Stderr, "Unknown backend: %s\n", *backend)
		os.Exit(1)
	}

	// 生成代码
	var output string
	if *target == "executable" {
		output, err = generateExecutable(program, codeGenerator, *backend, *runtime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating executable: %v\n", err)
			os.Exit(1)
		}
		outputFormat = "Executable " + outputFormat
	} else {
		output = codeGenerator.GenerateCode(program)
	}

	// 输出结果
	if *backend == "ocaml" {
		// OCaml后端：输出完整信息（向后兼容）
		fmt.Println("=== Parsed AST ===")
		fmt.Println(program)
		fmt.Printf("\n=== Generated %s ===\n", outputFormat)
		fmt.Println(output)
	} else {
		// 其他后端：只输出代码（用于管道处理）
		fmt.Println(output)
	}
}

// generateExecutable 生成可执行文件
func generateExecutable(program *entities.Program, generator services.CodeGenerator, backend string, runtime string) (string, error) {
	switch backend {
	case "llvm-ir":
		return generateLLVMExecutable(program, generator, runtime)
	default:
		return generator.GenerateCode(program), nil
	}
}

// generateLLVMExecutable 生成LLVM可执行文件
func generateLLVMExecutable(program *entities.Program, generator services.CodeGenerator, runtime string) (string, error) {
	// 1. 生成IR代码
	irCode := generator.GenerateCode(program)

	// 2. 写入IR文件
	irFile := "output.ll"
	if err := os.WriteFile(irFile, []byte(irCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write IR file: %w", err)
	}

	// 3. 使用clang直接编译IR为可执行文件
	executable := "program"

	// 根据runtime选择正确的库文件
	var runtimeLib string
	switch runtime {
	case "asm":
		runtimeLib = filepath.Join("build", "libcoroutine_asm.a")
		fmt.Printf("Using assembly-based runtime: %s\n", runtimeLib)
	default:
		runtimeLib = filepath.Join("build", "libcoroutine.a")
		fmt.Printf("Using ucontext-based runtime: %s\n", runtimeLib)
	}

	cmd := exec.Command("clang", irFile, runtimeLib, "-o", executable, "-lpthread")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Clang compilation error:\n%s\n", string(output))
		return "", fmt.Errorf("failed to compile and link executable: %w", err)
	}

	return fmt.Sprintf("Generated executable: %s", executable), nil
}
