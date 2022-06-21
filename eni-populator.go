package enimore

import (
	"context"
	"fmt"
	"sync"

	"github.com/aereal/enimore/internal/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
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

func NewENIPopulator(client aws.EC2Client) *ENIPopulator {
	return &ENIPopulator{client: client, res: &Result{Results: map[string]ResultFragment{}}}
}

type ENIPopulator struct {
	client aws.EC2Client
	res    *Result
}

func (p *ENIPopulator) Result() *Result {
	return p.res
}

func (p *ENIPopulator) PopulateWithSecurityGroups(ctx context.Context, sgAssociation *securityGroupAssociation) error {
	client := p.client
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []types.Filter{
			{Name: awssdk.String("group-id"), Values: sgAssociation.securityGroupIDs()},
			{Name: awssdk.String("attachment.status"), Values: []string{"attached"}},
		},
	}
	out, err := client.DescribeNetworkInterfaces(ctx, input)
	if err != nil {
		return fmt.Errorf("ec2.DescribeNetworkInterfaces: %w", err)
	}
	for _, x := range out.NetworkInterfaces {
		for _, sg := range x.Groups {
			resourceARN, ok := sgAssociation.get(*sg.GroupId)
			if !ok {
				continue
			}
			p.res.add(resourceARN, x)
			break
		}
	}
	return nil
}

type securityGroupAssociation struct {
	sync.RWMutex
	sgID2Resource map[string]arn.ARN
}

func (a *securityGroupAssociation) hasAny() bool {
	a.RLock()
	defer a.RUnlock()
	return len(a.sgID2Resource) > 0
}

func (a *securityGroupAssociation) add(resource arn.ARN, securityGroupIDs ...string) {
	a.Lock()
	defer a.Unlock()
	if a.sgID2Resource == nil {
		a.sgID2Resource = map[string]arn.ARN{}
	}
	for _, sgID := range securityGroupIDs {
		a.sgID2Resource[sgID] = resource
	}
}

func (a *securityGroupAssociation) get(arnRef string) (arn.ARN, bool) {
	a.RLock()
	defer a.RUnlock()
	if arnRef == "" {
		return arn.ARN{}, false
	}
	x, ok := a.sgID2Resource[arnRef]
	return x, ok
}

func (a *securityGroupAssociation) securityGroupIDs() []string {
	a.RLock()
	defer a.RUnlock()
	ret := make([]string, len(a.sgID2Resource))
	var i int
	for x := range a.sgID2Resource {
		ret[i] = x
		i++
	}
	return ret
}
