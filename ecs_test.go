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
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/golang/mock/gomock"
)

func TestECSServiceAccumulate_ok(t *testing.T) {
	serviceARN1 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-1/my-service"
	serviceARN2 := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster-2/my-service-2"
	securityGroupIDs1 := []string{"sg-1234567890", "sg-987654321"}
	securityGroupIDs2 := []string{"sg-8765432109", "sg-7654321098"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := mocks.NewMockECSClient(ctrl)
	mlc.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Times(1).Return(&ecs.DescribeServicesOutput{
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
	}, nil)
	mlc.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Times(1).Return(&ecs.DescribeServicesOutput{
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
	}, nil)
	mec := mocks.NewMockEC2Client(ctrl)
	mec.EXPECT().DescribeNetworkInterfaces(gomock.Any(), gomock.Any()).Times(1).Return(&ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []ec2types.NetworkInterface{
			{
				NetworkInterfaceId: aws.String("eni-12345"),
				Groups:             []ec2types.GroupIdentifier{{GroupId: &securityGroupIDs1[0]}, {GroupId: &securityGroupIDs1[1]}},
				AvailabilityZone:   aws.String("us-east-1a"),
			},
		},
	}, nil)
	p := enipopulator.New(mec)
	a := NewECSServiceAccumulator(mlc, []arn.ARN{mustParseARN(serviceARN1), mustParseARN(serviceARN2)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
}

func TestECSServiceAccumulate_notVPC(t *testing.T) {
	serviceARN := "arn:aws:ecs:us-east-1:123456789012:service/my-cluster/no-vpc"
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mlc := mocks.NewMockECSClient(ctrl)
	mlc.EXPECT().DescribeServices(gomock.Any(), gomock.Any()).Times(1).Return(&ecs.DescribeServicesOutput{
		Services: []ecstypes.Service{{ServiceArn: &serviceARN}},
	}, nil)
	mec := mocks.NewMockEC2Client(ctrl)
	p := enipopulator.New(mec)
	a := NewECSServiceAccumulator(mlc, []arn.ARN{mustParseARN(serviceARN)})
	ctx := context.Background()
	err := a.Accumulate(ctx, p)
	if err != nil {
		t.Fatal(err)
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
			mcc := mocks.NewMockECSClient(ctrl)
			mec := mocks.NewMockEC2Client(ctrl)
			p := enipopulator.New(mec)
			a := NewECSServiceAccumulator(mcc, targetARNs)
			ctx := context.Background()
			err := a.Accumulate(ctx, p)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
