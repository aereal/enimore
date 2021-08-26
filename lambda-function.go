package enimore

import (
	"context"

	"github.com/aereal/enimore/enipopulator"
	arnparser "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

var serviceLambda = "lambda"

type lambdaClient interface {
	ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error)
}

func NewLambdaFunctionAccumulator(client lambdaClient, arns []arnparser.ARN) *LambdaFunctionAccumulator {
	accum := &LambdaFunctionAccumulator{client: client}
	for _, arn := range arns {
		if arn.Service == serviceLambda {
			accum.arns = append(accum.arns, arn)
		}
	}
	return accum
}

type LambdaFunctionAccumulator struct {
	arns   []arnparser.ARN
	client lambdaClient
}

var _ Accumulator = &LambdaFunctionAccumulator{}

func (a *LambdaFunctionAccumulator) Accumulate(ctx context.Context, populator *enipopulator.ENIPopulator) error {
	// fnARN -> isUnseen
	unseen := map[string]bool{}
	for _, fn := range a.arns {
		unseen[fn.String()] = true
	}
	var securityGroupIds []string
	sg2fn := map[string]string{}
	input := &lambda.ListFunctionsInput{}
	for {
		out, err := a.client.ListFunctions(ctx, input)
		if err != nil {
			return err
		}
		for _, fn := range out.Functions {
			if fn.VpcConfig == nil {
				continue
			}
			if !unseen[*fn.FunctionArn] {
				continue
			}
			securityGroupIds = append(securityGroupIds, fn.VpcConfig.SecurityGroupIds...)
			for _, sg := range fn.VpcConfig.SecurityGroupIds {
				sg2fn[sg] = *fn.FunctionArn
			}
			delete(unseen, *fn.FunctionArn)
		}
		if len(unseen) == 0 || out.NextMarker == nil {
			break
		}
		input.Marker = out.NextMarker
	}
	if err := populator.PopulateWithSecurityGroups(ctx, securityGroupIds, sg2fn); err != nil {
		return err
	}
	return nil
}
