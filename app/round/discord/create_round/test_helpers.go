package createround

import "context"

// testOperationWrapper is a shared test helper that bypasses operation wrapping for testing
var testOperationWrapper = func(ctx context.Context, operationName string, operationFunc func(ctx context.Context) (CreateRoundOperationResult, error)) (CreateRoundOperationResult, error) {
	return operationFunc(ctx)
}
