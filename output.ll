@.str.1 = global [2 x i8] c"ok"
@.str.2 = global [12 x i8] c"Before await"
@.str.3 = global [11 x i8] c"After await"

declare i8* @coroutine_spawn(void (i8*)* %entry, i32 %arg_count, i8* %args, i8* %future)

declare i8* @coroutine_await(i8* %future)

declare i8* @future_new()

declare void @future_resolve(i8* %future, i8* %value)

declare void @scheduler_yield()

declare i8* @channel_create()

declare void @channel_send(i8* %channel, i8* %value)

declare i8* @channel_receive(i8* %channel)

declare i32 @channel_select(i8* %channels, i8* %operations, i32 %count)

declare void @run_scheduler()

declare void @future_reject(i8* %future, i8* %error)

declare void @print_int(i32 %value)

declare void @print_string(i8* %str)

define i8* @simpleAsync() {
entry:
	%0 = call i8* @future_new()
	ret i8* %0
}

define void @simpleAsync_executor(i8* %param0) {
entry:
	%0 = bitcast [2 x i8]* @.str.1 to i8*
	call void @future_resolve(i8* %param0, i8* %0)
	ret void
}

define i32 @main() {
entry:
	%0 = bitcast [12 x i8]* @.str.2 to i8*
	call void @print_string(i8* %0)
	%result = alloca i8*
	%1 = call i8* @simpleAsync()
	%2 = call i8* @coroutine_await(i8* %1)
	store i8* %2, i8** %result
	%3 = bitcast [11 x i8]* @.str.3 to i8*
	call void @print_string(i8* %3)
	call void @run_scheduler()
	ret i32 0
}
