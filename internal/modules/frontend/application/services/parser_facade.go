package services

import "echo/internal/modules/frontend/domain/entities"

// ParserFacade 应用层使用的解析器门面接口
// 作为 Application 层对解析能力的抽象，内部由 domain 层实现提供。
type ParserFacade interface {
	Parse(source string) (*entities.Program, error)
}

