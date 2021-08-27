package enimore

import (
	"context"
	"fmt"
	"testing"

	"github.com/aereal/enimore/enipopulator"
	"github.com/aereal/enimore/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/golang/mock/gomock"
)

func TestLambdaFunctionAccumulate_notVPC(t *testing.T) {
	fnARN := "arn:aws:lambda:us-east-1:123456789012:function/no-vpc"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := mocks.NewMockLambdaClient(ctrl)
	mlc.EXPECT().ListFunctions(gomock.Any(), gomock.Any()).Times(1).Return(&lambda.ListFunctionsOutput{
		Functions: []lambdatypes.FunctionConfiguration{{FunctionArn: &fnARN}},
	}, nil)
	mec := mocks.NewMockEC2Client(ctrl)
	p := enipopulator.New(mec)
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
			p := enipopulator.New(mec)
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
