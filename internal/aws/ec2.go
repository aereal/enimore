//go:generate go run github.com/golang/mock/mockgen -package aws -destination ./mock_ec2.go github.com/aereal/enimore/internal/aws EC2Client

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type EC2Client interface {
	DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}
