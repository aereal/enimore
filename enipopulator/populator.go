package enipopulator

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type NetworkInterface struct {
	NetworkInterfaceID string
	AvailabilityZone   string `json:",omitempty"`
}

type ResultFragment struct {
	NetworkInterfaces []NetworkInterface
}

type Result struct {
	sync.Mutex

	Results map[string]ResultFragment
}

func (r *Result) add(resourceARN arn.ARN, eni types.NetworkInterface) {
	r.Lock()
	defer r.Unlock()
	if r.Results == nil {
		r.Results = map[string]ResultFragment{}
	}
	f := ResultFragment{}
	key := resourceARN.String()
	f.NetworkInterfaces = append(r.Results[key].NetworkInterfaces, NetworkInterface{NetworkInterfaceID: *eni.NetworkInterfaceId, AvailabilityZone: *eni.AvailabilityZone})
	r.Results[key] = f
}

type ec2Client interface {
	DescribeNetworkInterfaces(ctx context.Context, params *ec2.DescribeNetworkInterfacesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error)
}

func New(client ec2Client) *ENIPopulator {
	return &ENIPopulator{client: client, res: &Result{Results: map[string]ResultFragment{}}}
}

type ENIPopulator struct {
	client ec2Client
	res    *Result
}

func (p *ENIPopulator) Result() *Result {
	return p.res
}

func (p *ENIPopulator) PopulateWithSecurityGroups(ctx context.Context, securityGroupIds []string, sg2resource map[string]arn.ARN) error {
	client := p.client
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{Name: aws.String("group-id"), Values: securityGroupIds},
			{Name: aws.String("attachment.status"), Values: []string{"attached"}},
		},
	}
	out, err := client.DescribeNetworkInterfaces(ctx, input)
	if err != nil {
		return fmt.Errorf("ec2.DescribeNetworkInterfaces: %w", err)
	}
	for _, x := range out.NetworkInterfaces {
		for _, sg := range x.Groups {
			if sg.GroupId == nil {
				continue
			}
			resourceARN, ok := sg2resource[*sg.GroupId]
			if !ok {
				continue
			}
			p.res.add(resourceARN, x)
		}
	}
	return nil
}
