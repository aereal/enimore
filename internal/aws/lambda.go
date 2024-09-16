//go:generate go run github.com/golang/mock/mockgen -package aws -destination ./mock_lambda.go github.com/aereal/enimore/internal/aws LambdaClient

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type LambdaClient interface {
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
}
