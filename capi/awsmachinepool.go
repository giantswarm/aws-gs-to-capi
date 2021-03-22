package capi

import (
	"fmt"
	v1 "k8s.io/api/core/v1"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	giantswarmawsalpha3 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	capiawsexpv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmapiv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kubeadmtypev1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/v1beta1"
	expapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
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
			Name:      machinePoolName(clusterID, d.Name),
			Namespace: d.Namespace,
		},
		Spec: capiawsexpv1alpha3.AWSMachinePoolSpec{
			MinSize: int32(d.Spec.NodePool.Scaling.Min),
			MaxSize: int32(d.Spec.NodePool.Scaling.Max),
			AWSLaunchTemplate: capiawsexpv1alpha3.AWSLaunchTemplate{
				Name:               d.Name,
				InstanceType:       d.Spec.Provider.Worker.InstanceType,
				SSHKeyName:         aws.String("vaclav"),
				IamInstanceProfile: "nodes.cluster-api-provider-aws.sigs.k8s.io",
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
		awsmp.Spec.AvailabilityZones = append(awsmp.Spec.AvailabilityZones, *subnet.AvailabilityZone)
	}

	return awsmp, nil
}

func machinePool(d *giantswarmawsalpha3.AWSMachineDeployment, clusterID string, k8sVersion string) *expapiv1alpha3.MachinePool {
	replicas := int32(d.Spec.NodePool.Scaling.Min)
	mp := &expapiv1alpha3.MachinePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeadmConfig",
			APIVersion: capiawsexpv1alpha3.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machinePoolName(clusterID, d.Name),
			Namespace: d.Namespace,
		},
		Spec: expapiv1alpha3.MachinePoolSpec{
			ClusterName: clusterID,
			Replicas:    &replicas,
			Template: apiv1alpha3.MachineTemplateSpec{
				Spec: apiv1alpha3.MachineSpec{
					ClusterName: clusterID,
					Version:     &k8sVersion,
					InfrastructureRef: v1.ObjectReference{
						Name:       machinePoolName(clusterID, d.Name),
						Namespace:  d.Namespace,
						Kind:       "AWSMachinePool",
						APIVersion: capiawsexpv1alpha3.GroupVersion.String(),
					},
					Bootstrap: apiv1alpha3.Bootstrap{
						ConfigRef: &v1.ObjectReference{
							Name:       machinePoolName(clusterID, d.Name),
							Namespace:  d.Namespace,
							Kind:       "KubeadmConfig",
							APIVersion: kubeadmapiv1alpha3.GroupVersion.String(),
						},
					},
				},
			},
		},
	}
	return mp
}

func machinePoolKubeAdmConfig(d *giantswarmawsalpha3.AWSMachineDeployment, clusterID string) *kubeadmapiv1alpha3.KubeadmConfig {
	c := &kubeadmapiv1alpha3.KubeadmConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KubeadmConfig",
			APIVersion: capiawsexpv1alpha3.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      machinePoolName(clusterID, d.Name),
			Namespace: d.Namespace,
		},
		Spec: kubeadmapiv1alpha3.KubeadmConfigSpec{
			PreKubeadmCommands: []string{
				"hostnamectl set-hostname $(curl http://169.254.169.254/latest/meta-data/local-hostname)",
			},
			InitConfiguration: &kubeadmtypev1beta1.InitConfiguration{
				NodeRegistration: kubeadmtypev1beta1.NodeRegistrationOptions{
					KubeletExtraArgs: map[string]string{
						"cloud-provider": "aws",
					},
					Name: "{{ ds.meta_data.local_hostname }}",
				},
			},
			JoinConfiguration: &kubeadmtypev1beta1.JoinConfiguration{
				NodeRegistration: kubeadmtypev1beta1.NodeRegistrationOptions{
					KubeletExtraArgs: map[string]string{
						"cloud-provider": "aws",
						"node-labels":    "node.kubernetes.io/worker,role=worker",
					},
					Name: "{{ ds.meta_data.local_hostname }}",
				},
			},
			Files: []kubeadmapiv1alpha3.File{
				{
					Path:  "/etc/kubernetes/config/kube-proxy.yaml",
					Owner: "root:root",
					ContentFrom: &kubeadmapiv1alpha3.FileSource{
						Secret: kubeadmapiv1alpha3.SecretFileSource{
							Name: customFilesSecretName(clusterID),
							Key:  kubeProxyKubeconfigKey,
						},
					},
				},
				{
					Path:  "/etc/kubernetes/config/proxy-config.yml",
					Owner: "root:root",
					ContentFrom: &kubeadmapiv1alpha3.FileSource{
						Secret: kubeadmapiv1alpha3.SecretFileSource{
							Name: customFilesSecretName(clusterID),
							Key:  kubeProxyConfigKey,
						},
					},
				},
			},
		},
	}

	return c
}

func machinePoolName(clusterID string, machinePool string) string {
	return fmt.Sprintf("%s-worker-%s", clusterID, machinePool)
}
