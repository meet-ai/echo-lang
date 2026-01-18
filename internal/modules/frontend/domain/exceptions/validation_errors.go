package exceptions

import "errors"

// Validation errors for frontend operations
var (
	ErrSourceFileIDRequired   = errors.New("source file ID is required")
	ErrSourceCodeRequired     = errors.New("source code is required")
	ErrSourceFilePathRequired = errors.New("source file path is required")
	ErrInvalidSourceFileID    = errors.New("invalid source file ID")
	ErrInvalidFilePath        = errors.New("invalid file path")
	ErrAnalysisNotComplete    = errors.New("required analysis not completed")
	ErrInvalidAnalysisType    = errors.New("invalid analysis type")
)
