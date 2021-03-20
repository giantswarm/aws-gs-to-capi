package capi

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	giantswarmawsalpha3 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
)

const (
	awsTagSubnetType     = "giantswarm.io/subnet-type"
	awsSubnetTypePrivate = "private"
	awsSubnetTypePublic  = "public"
	awsSubnetTypeCNI     = "aws-cni"

	vpcIDFilter = "vpc-id"
)

func awsClusterName(clusterID string) string {
	return fmt.Sprintf("%s", clusterID)
}

func transformAWSCluster(awsCluster *giantswarmawsalpha3.AWSCluster) (*capiawsv1alpha3.AWSCluster, error) {
	var err error

	var igw *string
	{
		igw, err = fetchClusterIGW(awsCluster.Spec.Provider.Region, awsCluster.Status.Provider.Network.VPCID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var subnets capiawsv1alpha3.Subnets
	{
		subnets, err = fetchClusterSubnets(awsCluster.Spec.Provider.Region, awsCluster.Status.Provider.Network.VPCID)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	cl := &capiawsv1alpha3.AWSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capiawsv1alpha3.GroupVersion.String(),
			Kind:       "AWSCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      awsClusterName(awsCluster.Name),
			Namespace: awsCluster.Namespace,
		},

		Spec: capiawsv1alpha3.AWSClusterSpec{
			ControlPlaneEndpoint: apiv1alpha3.APIEndpoint{
				Port: 443,
			},
			NetworkSpec: capiawsv1alpha3.NetworkSpec{
				VPC: capiawsv1alpha3.VPCSpec{
					ID:                awsCluster.Status.Provider.Network.VPCID,
					CidrBlock:         awsCluster.Status.Provider.Network.CIDR,
					InternetGatewayID: igw,
				},
				Subnets: subnets,
			},
			Region: awsCluster.Spec.Provider.Region,
			Bastion: capiawsv1alpha3.Bastion{
				Enabled: false,
			},
			ControlPlaneLoadBalancer: &capiawsv1alpha3.AWSLoadBalancerSpec{},
		},
	}

	return cl, nil
}

func fetchClusterSubnets(region string, vpcID string) (capiawsv1alpha3.Subnets, error) {
	sess, err := getAWSSession(region)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ec2Client := ec2.New(sess)

	i := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(vpcIDFilter),
				Values: aws.StringSlice([]string{vpcID}),
			},
			{
				Name:   aws.String("tag:giantswarm.io/stack"),
				Values: aws.StringSlice([]string{"tccp"}),
			},
		},
	}

	allVPCSubnets, err := ec2Client.DescribeSubnets(i)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var filteredSubnets capiawsv1alpha3.Subnets
	for _, subnet := range allVPCSubnets.Subnets {
		// ignore CNI subnets, we only add private and public subnets to the CRs
		if !isCNISubnet(subnet.Tags) {
			s := &capiawsv1alpha3.SubnetSpec{
				ID:               *subnet.SubnetId,
				CidrBlock:        *subnet.CidrBlock,
				AvailabilityZone: *subnet.AvailabilityZone,
				IsPublic:         isPublicNetwork(subnet.Tags),
			}
			filteredSubnets = append(filteredSubnets, s)
		}
	}

	return filteredSubnets, nil
}

func fetchClusterIGW(region string, vpcID string) (*string, error) {
	sess, err := getAWSSession(region)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ec2Client := ec2.New(sess)

	i := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.vpc-id"),
				Values: aws.StringSlice([]string{vpcID}),
			},
		},
	}

	o, err := ec2Client.DescribeInternetGateways(i)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(o.InternetGateways) != 1 {
		return nil, microerror.Maskf(nil, "found %d internet gateways but expected 1", len(o.InternetGateways))
	}

	return o.InternetGateways[0].InternetGatewayId, nil
}

func getAWSSession(region string) (*session.Session, error) {
	// recreate aws session, this time with proper region
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return awsSession, nil
}

func isCNISubnet(tags []*ec2.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == awsTagSubnetType && *tag.Value == awsSubnetTypeCNI {
			return true
		}
	}

	return false
}

func isPublicNetwork(tags []*ec2.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == awsTagSubnetType && *tag.Value == awsSubnetTypePublic {
			return true
		}
	}

	return false
}
