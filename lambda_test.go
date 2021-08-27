package enimore

import (
	"context"
	"fmt"
	"testing"

	"github.com/aereal/enimore/enipopulator"
	"github.com/aereal/enimore/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/golang/mock/gomock"
)

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
