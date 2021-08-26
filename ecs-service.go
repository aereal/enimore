package enimore

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aereal/enimore/enipopulator"
	arnparser "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"golang.org/x/sync/errgroup"
)

var serviceECS = "ecs"

type ecsClient interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
}

func NewECSServiceAccumulator(client ecsClient, arns []arnparser.ARN) *ECSServiceAccumulator {
	accum := &ECSServiceAccumulator{client: client}
	for _, arn := range arns {
		if arn.Service == serviceECS {
			accum.arns = append(accum.arns, arn)
		}
	}
	return accum
}

type ECSServiceAccumulator struct {
	arns   []arnparser.ARN
	client ecsClient
}

var _ Accumulator = &ECSServiceAccumulator{}

func clusterArnFromServiceArn(serviceARN arnparser.ARN) (arnparser.ARN, error) {
	if serviceARN.Service != serviceECS {
		return arnparser.ARN{}, fmt.Errorf("service must be ecs")
	}
	resource := strings.Split(serviceARN.Resource, "/")
	if len(resource) < 2 {
		return arnparser.ARN{}, fmt.Errorf("ARN resource is malformed")
	}
	return arnparser.ARN{
		AccountID: serviceARN.AccountID,
		Partition: serviceARN.Partition,
		Region:    serviceARN.Region,
		Service:   serviceARN.Service,
		Resource:  fmt.Sprintf("cluster/%s", resource[1]),
	}, nil
}

func (a *ECSServiceAccumulator) Accumulate(ctx context.Context, populator *enipopulator.ENIPopulator) error {
	// cluster => *ecs.DescribeServicesInput
	inputs := map[string]*ecs.DescribeServicesInput{}
	for _, serviceARN := range a.arns {
		clusterARN, err := clusterArnFromServiceArn(serviceARN)
		if err != nil {
			log.Printf("cannot extract cluster ARN from %s: %s", serviceARN, err)
			continue
		}
		key := clusterARN.String()
		if inputs[key] == nil {
			inputs[key] = &ecs.DescribeServicesInput{
				Cluster: &key,
			}
		}
		inputs[key].Services = append(inputs[key].Services, serviceARN.String())
	}
	eg, ctx := errgroup.WithContext(ctx)
	for _, i := range inputs {
		input := i
		eg.Go(func() error {
			out, err := a.client.DescribeServices(ctx, input)
			if err != nil {
				return fmt.Errorf("failed to describe service: %w", err)
			}
			var sgs []string
			sg2Service := map[string]string{}
			for _, svc := range out.Services {
				if svc.NetworkConfiguration == nil {
					continue
				}
				sgs = append(sgs, svc.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups...)
				for _, sg := range sgs {
					sg2Service[sg] = *svc.ServiceArn
				}
			}
			if err := populator.PopulateWithSecurityGroups(ctx, sgs, sg2Service); err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
