package enimore

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/aereal/enimore/internal/aws"
	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestECSServiceAccumulate_ok(t *testing.T) {
	serviceARN1 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-1/my-service"
	serviceARN2 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-2/my-service-2"
	securityGroupIDs1 := []string{"sg-1234567890", "sg-987654321"}
	securityGroupIDs2 := []string{"sg-8765432109", "sg-7654321098"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := aws.NewMockECSClient(ctrl)
	mlc.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(func(ctx context.Context, input *ecs.DescribeServicesInput, opts ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
		k := strings.Join(input.Services, "")
		switch k {
		case serviceARN1:
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceArn: &serviceARN1,
						NetworkConfiguration: &ecstypes.NetworkConfiguration{
							AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
								SecurityGroups: securityGroupIDs1,
							},
						},
					},
				},
			}, nil
		case serviceARN2:
			return &ecs.DescribeServicesOutput{
				Services: []ecstypes.Service{
					{
						ServiceArn: &serviceARN2,
						NetworkConfiguration: &ecstypes.NetworkConfiguration{
							AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
								SecurityGroups: securityGroupIDs2,
							},
						},
					},
				},
			}, nil
		default:
			t.Fatalf("unknown DescribeServices.Services: %#v", input.Services)
			return nil, nil
		}
	})
	mec := aws.NewMockEC2Client(ctrl)
	mec.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []ec2types.NetworkInterface{
			{
				NetworkInterfaceId: awssdk.String("eni-12345"),
				Groups:             []ec2types.GroupIdentifier{{GroupId: &securityGroupIDs1[0]}, {GroupId: &securityGroupIDs1[1]}, {GroupId: &securityGroupIDs2[0]}, {GroupId: &securityGroupIDs2[1]}},
				AvailabilityZone:   awssdk.String("us-east-1a"),
			},
			{
				NetworkInterfaceId: awssdk.String("eni-67890"),
				Groups:             []ec2types.GroupIdentifier{{GroupId: &securityGroupIDs2[0]}, {GroupId: &securityGroupIDs2[1]}},
				AvailabilityZone:   awssdk.String("us-east-1b"),
			},
		},
	}, nil)
	p := NewENIPopulator(mec)
	a := NewECSServiceAccumulator(mlc, []arn.ARN{mustParseARN(serviceARN1), mustParseARN(serviceARN2)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	got := p.Result()
	want := &Result{Results: map[string]ResultFragment{
		serviceARN1: {
			NetworkInterfaces: []NetworkInterface{
				{NetworkInterfaceID: "eni-12345", AvailabilityZone: "us-east-1a"},
			},
		},
		serviceARN2: {
			NetworkInterfaces: []NetworkInterface{
				{NetworkInterfaceID: "eni-67890", AvailabilityZone: "us-east-1b"},
			},
		},
	}}
	if diff := diffResult(t, got, want); diff != "" {
		t.Errorf("Result (-got, +want):\n%s", diff)
	}
}

func TestECSServiceAccumulate_notVPC(t *testing.T) {
	serviceARN := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/no-vpc"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := aws.NewMockECSClient(ctrl)
	mlc.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Times(1).Return(&ecs.DescribeServicesOutput{
		Services: []ecstypes.Service{{ServiceArn: &serviceARN}},
	}, nil)
	mec := aws.NewMockEC2Client(ctrl)
	p := NewENIPopulator(mec)
	a := NewECSServiceAccumulator(mlc, []arn.ARN{mustParseARN(serviceARN)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	got := p.Result()
	if diff := diffResult(t, got, &Result{Results: map[string]ResultFragment{}}); diff != "" {
		t.Errorf("Result (-got, +want):\n%s", diff)
	}
}

func TestECSServiceAccumulate_noARNs(t *testing.T) {
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
			mcc := aws.NewMockECSClient(ctrl)
			mec := aws.NewMockEC2Client(ctrl)
			p := NewENIPopulator(mec)
			a := NewECSServiceAccumulator(mcc, targetARNs)
			ctx := context.Background()
			err := a.Accumulate(ctx, p)
			if err != nil {
				t.Fatal(err)
			}
			got := p.Result()
			if diff := diffResult(t, got, &Result{Results: map[string]ResultFragment{}}); diff != "" {
				t.Errorf("Result (-got, +want):\n%s", diff)
			}
		})
	}
}

func diffResult(t *testing.T, got, want *Result) string {
	t.Helper()
	return cmp.Diff(got, want, cmpopts.IgnoreUnexported(sync.Mutex{}))
}
