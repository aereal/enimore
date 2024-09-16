//go:generate go run github.com/golang/mock/mockgen -package aws -destination ./mock_ecs.go github.com/aereal/enimore/internal/aws ECSClient

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ECSClient interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
}
