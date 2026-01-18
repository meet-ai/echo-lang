define i32 @main() {
entry:
	%name = alloca i32
	%age = alloca i32
	store i32 1, i32* %age
	%numbers = alloca i32
	%sum = alloca i32
	store i32 10, i32* %sum
	%greeting = alloca i32
	%greeting_load = load i32, i32* %greeting
	%status = alloca i32
	%status_load = load i32, i32* %status
	%sum = alloca i32
	%sum_load = load i32, i32* %sum
	%binop = sub i32 %sum_load, 1
	store i32 %binop, i32* %sum
	%age = alloca i32
	%age_load = load i32, i32* %age
	%binop = add i32 %age_load, 1
	store i32 %binop, i32* %age
	%age = alloca i32
	%age_load = load i32, i32* %age
	%binop = add i32 %age_load, 1
	store i32 %binop, i32* %age
	ret i32 0
}

