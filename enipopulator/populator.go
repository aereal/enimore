package enipopulator

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ResultFragment struct {
	NetworkInterfaces []string
}

type Result struct {
	sync.Mutex

	Results map[string]ResultFragment
}

func (r *Result) Add(key string, enis []string) {
	r.Lock()
	defer r.Unlock()
	if r.Results == nil {
		r.Results = map[string]ResultFragment{}
	}
	f := ResultFragment{}
	f.NetworkInterfaces = append(r.Results[key].NetworkInterfaces, enis...)
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

func (p *ENIPopulator) PopulateWithSecurityGroups(ctx context.Context, securityGroupIds []string, sg2resource map[string]string) error {
	client := p.client
	out, err := client.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{Filters: []types.Filter{{Name: aws.String("group-id"), Values: securityGroupIds}}})
	if err != nil {
		return fmt.Errorf("ec2.DescribeNetworkInterfaces: %w", err)
	}
	for _, x := range out.NetworkInterfaces {
		for _, sg := range x.Groups {
			if sg.GroupId == nil {
				continue
			}
			resource := sg2resource[*sg.GroupId]
			if resource == "" {
				continue
			}
			p.res.Add(resource, []string{*x.NetworkInterfaceId})
		}
	}
	return nil
}
