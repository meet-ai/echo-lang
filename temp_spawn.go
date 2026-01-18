func (ee *ExpressionEvaluatorImpl) EvaluateSpawnExpr(irManager generation.IRModuleManager, expr *entities.SpawnExpr) (interface{}, error) {
	// 检查是否是async函数的spawn
	if ident, ok := expr.Function.(*entities.Identifier); ok {
		// 查找executor函数
		executorFuncName := ident.Name + "_executor"
		executorFunc, exists := irManager.GetExternalFunction(executorFuncName)
		if exists {
			// 创建Future
			futureValue, err := irManager.CreateCall(irManager.GetExternalFunction("future_new"))
			if err != nil {
				return nil, fmt.Errorf("failed to create future: %w", err)
			}

			// spawn协程
			spawnFunc, spawnExists := irManager.GetExternalFunction("coroutine_spawn")
			if spawnExists {
				spawnArgs := []interface{}{
					executorFunc,                            // executor function
					constant.NewInt(types.I32, 1),         // arg_count
					constant.NewNull(types.NewPointer(types.I8)), // args (null for now)
					futureValue,                            // future
				}
				_, err = irManager.CreateCall(spawnFunc, spawnArgs...)
				if err != nil {
					return nil, fmt.Errorf("failed to create spawn call: %w", err)
				}
			}

			return futureValue, nil
		}
	}

	// 非async函数的spawn暂时不支持
	return nil, fmt.Errorf("spawn only supports async functions currently")
}
