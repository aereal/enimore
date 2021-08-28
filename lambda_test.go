package enimore

import (
	"context"
	"fmt"
	"testing"

	"github.com/aereal/enimore/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/golang/mock/gomock"
)

func TestLambdaFunctionAccumulate_ok(t *testing.T) {
	fnARN := "arn:aws:lambda:us-east-1:123456789012:function/my-fn"
	securityGroupIDs := []string{"sg-1234567890", "sg-987654321"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := mocks.NewMockLambdaClient(ctrl)
	mlc.EXPECT().ListFunctions(gomock.Any(), gomock.Any()).Times(1).Return(&lambda.ListFunctionsOutput{
		Functions: []lambdatypes.FunctionConfiguration{
			{
				FunctionArn: &fnARN,
				VpcConfig: &lambdatypes.VpcConfigResponse{
					SecurityGroupIds: securityGroupIDs,
				},
			},
		},
	}, nil)
	mec := mocks.NewMockEC2Client(ctrl)
	mec.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []ec2types.NetworkInterface{
			{
				NetworkInterfaceId: aws.String("eni-12345"),
				Groups:             []ec2types.GroupIdentifier{{GroupId: &securityGroupIDs[0]}, {GroupId: &securityGroupIDs[1]}},
				AvailabilityZone:   aws.String("us-east-1a"),
			},
		},
	}, nil)
	p := NewENIPopulator(mec)
	a := NewLambdaFunctionAccumulator(mlc, []arn.ARN{mustParseARN(fnARN)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLambdaFunctionAccumulate_notVPC(t *testing.T) {
	fnARN := "arn:aws:lambda:us-east-1:123456789012:function/no-vpc"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := mocks.NewMockLambdaClient(ctrl)
	mlc.EXPECT().ListFunctions(gomock.Any(), gomock.Any()).Times(1).Return(&lambda.ListFunctionsOutput{
		Functions: []lambdatypes.FunctionConfiguration{{FunctionArn: &fnARN}},
	}, nil)
	mec := mocks.NewMockEC2Client(ctrl)
	p := NewENIPopulator(mec)
	a := NewLambdaFunctionAccumulator(mlc, []arn.ARN{mustParseARN(fnARN)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLambdaFunctionAccumulate_noARNs(t *testing.T) {
	xs := [][]arn.ARN{
		{},
		{
			mustParseARN("arn:aws:iam::123456789012:role/my-role"),
		},
	}
	for _, targetARNs := range xs {
		t.Run(fmt.Sprintf("targetARNs=%#v", targetARNs), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mlc := mocks.NewMockLambdaClient(ctrl)
			mec := mocks.NewMockEC2Client(ctrl)
			p := NewENIPopulator(mec)
			a := NewLambdaFunctionAccumulator(mlc, targetARNs)
			ctx := context.Background()
			err := a.Accumulate(ctx, p)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func mustParseARN(s string) arn.ARN {
	parsed, err := arn.Parse(s)
	if err != nil {
		panic(err)
	}
	return parsed
}
