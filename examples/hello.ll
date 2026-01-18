; ModuleID = 'examples/hello.eo'
source_filename = "examples/hello.eo"

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
declare void @print_int(i32 %value)
declare void @print_string(i8* %str)

define i32 @main() {
entry:
  call void @run_scheduler()
  ret i32 0
}
