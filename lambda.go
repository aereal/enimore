package enimore

import (
	"context"
	"fmt"

	"github.com/aereal/enimore/internal/aws"
	arnparser "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

var serviceLambda = "lambda"

func NewLambdaFunctionAccumulator(client aws.LambdaClient, arns []arnparser.ARN) *LambdaFunctionAccumulator {
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
	client aws.LambdaClient
}

var _ Accumulator = &LambdaFunctionAccumulator{}

func (a *LambdaFunctionAccumulator) Accumulate(ctx context.Context, populator *ENIPopulator) error {
	if len(a.arns) == 0 {
		return nil
	}
	// fnARN -> isUnseen
	unseen := map[string]bool{}
	for _, fn := range a.arns {
		unseen[fn.String()] = true
	}
	association := &securityGroupAssociation{}
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
			fnARN, err := arnparser.Parse(*fn.FunctionArn)
			if err != nil {
				return fmt.Errorf("[BUG] invalid ARN: %w", err)
			}
			association.add(fnARN, fn.VpcConfig.SecurityGroupIds...)
			delete(unseen, *fn.FunctionArn)
		}
		if len(unseen) == 0 || out.NextMarker == nil {
			break
		}
		input.Marker = out.NextMarker
	}
	if association.hasAny() {
		if err := populator.PopulateWithSecurityGroups(ctx, association); err != nil {
			return err
		}
	}
	return nil
}
