package capi

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	giantswarmawsalpha3 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
)

func awsMachineTemplateCPName(clusterID string) string {
	return fmt.Sprintf("%s-control-plane", clusterID)
}
func transformAWSMachineTemplateCP(cp *giantswarmawsalpha3.AWSControlPlane, clusterID string, region string) (*capiawsv1alpha3.AWSMachineTemplate, error) {
	sshKeyName := "vaclav"

	sess, err := getAWSSession(region)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ec2Client := ec2.New(sess)

	i := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: aws.StringSlice([]string{fmt.Sprintf("%s-master", clusterID)}),
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

	machineTemplate := &capiawsv1alpha3.AWSMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: capiawsv1alpha3.GroupVersion.String(),
			Kind:       "AWSMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      awsMachineTemplateCPName(clusterID),
			Namespace: cp.Namespace,
		},
		Spec: capiawsv1alpha3.AWSMachineTemplateSpec{
			Template: capiawsv1alpha3.AWSMachineTemplateResource{
				Spec: capiawsv1alpha3.AWSMachineSpec{
					IAMInstanceProfile: "control-plane.cluster-api-provider-aws.sigs.k8s.io",
					InstanceType:       cp.Spec.InstanceType,
					SSHKeyName:         &sshKeyName,
					AdditionalSecurityGroups: []capiawsv1alpha3.AWSResourceReference{
						{
							ID: o.SecurityGroups[0].GroupId,
						},
					},
				},
			},
		},
	}

	return machineTemplate, nil
}
