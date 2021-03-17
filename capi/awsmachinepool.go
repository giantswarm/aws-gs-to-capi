package capi

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	giantswarmawsalpha3 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	capiawsexpv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
)

func awsmachinepool(d *giantswarmawsalpha3.AWSMachineDeployment, region string, clusterID string) (*capiawsexpv1alpha3.AWSMachinePool, error) {
	sess, err := getAWSSession(region)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ec2Client := ec2.New(sess)

	i := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{fmt.Sprintf("%s-worker", clusterID)}),
			},
			{
				Name:   aws.String("tag:giantswarm.io/machine-deployment"),
				Values: aws.StringSlice([]string{d.Name}),
			},
		},
	}

	o, err := ec2Client.DescribeSecurityGroups(i)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(o.SecurityGroups) != 1 {
		return nil, microerror.Maskf(nil, "expected 1 master security group but found %d", len(o.SecurityGroups))
	}

	i2 := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:giantswarm.io/machine-deployment"),
				Values: aws.StringSlice([]string{d.Name}),
			},
		},
	}

	o2, err := ec2Client.DescribeSubnets(i2)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	awsmp := &capiawsexpv1alpha3.AWSMachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AWSMachinePool",
			APIVersion: capiawsexpv1alpha3.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
		},
		Spec: capiawsexpv1alpha3.AWSMachinePoolSpec{
			MinSize: int32(d.Spec.NodePool.Scaling.Min),
			MaxSize: int32(d.Spec.NodePool.Scaling.Max),
			AWSLaunchTemplate: capiawsexpv1alpha3.AWSLaunchTemplate{
				Name:               d.Name,
				InstanceType:       d.Spec.Provider.Worker.InstanceType,
				SSHKeyName:         aws.String("vaclav"),
				IamInstanceProfile: "node.cluster-api-provider-aws.sigs.k8s.io",
				AdditionalSecurityGroups: []capiawsv1alpha3.AWSResourceReference{
					{
						ID: o.SecurityGroups[0].GroupId,
					},
				},
			},
		},
	}

	for _, subnet := range o2.Subnets {
		awsmp.Spec.Subnets = append(awsmp.Spec.Subnets, capiawsv1alpha3.AWSResourceReference{ID: subnet.SubnetId})
	}

	return awsmp, nil
}
