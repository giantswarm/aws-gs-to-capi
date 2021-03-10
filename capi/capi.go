package capi

import (
	"fmt"
	v1 "k8s.io/api/core/v1"

	"github.com/giantswarm/microerror"
	awsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

type Crs struct {
	UnitSecret *v1.Secret

	Cluster                     *apiv1alpha3.Cluster
	AWSCluster                  *awsv1alpha3.AWSCluster
	ControlPlane                *kubeadmv1alpha3.KubeadmControlPlane
	ControlPlaneMachineTemplate *awsv1alpha3.AWSMachineTemplate

	MachineDeployments         []*apiv1alpha3.MachineDeployment
	MachineDeploymentTemplates []*awsv1alpha3.AWSMachineTemplate
}

func TransformGsToCAPICrs(gsCRs *giantswarm.GSClusterCrs, k8sVersion string) (*Crs, error) {
	var err error
	clusterID := gsCRs.AWSCluster.Name
	namespace := gsCRs.AWSCluster.Namespace

	secret, err := unitSecret(clusterID, namespace, gsCRs.AWSControlPlane.Spec.InstanceType)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	cluster := transformCluster(gsCRs)

	awsCluster, err := transformAWSCluster(gsCRs.AWSCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	kubeadmCP := transformKubeAdmControlPlane(gsCRs, k8sVersion)

	cpMachineTemplate := transformAWSMachineTemplateCP(gsCRs.AWSControlPlane, clusterID)

	crs := &Crs{
		UnitSecret: &secret,

		Cluster:                     cluster,
		AWSCluster:                  awsCluster,
		ControlPlane:                kubeadmCP,
		ControlPlaneMachineTemplate: cpMachineTemplate,
	}

	return crs, nil
}

func PrintOutCrs(crs *Crs) error {
	out := ""

	secret, err := yaml.Marshal(crs.UnitSecret)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(secret)
	out += "\n---\n"

	cluster, err := yaml.Marshal(crs.Cluster)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(cluster)
	out += "\n---\n"

	awsCluster, err := yaml.Marshal(crs.AWSCluster)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(awsCluster)
	out += "\n---\n"

	controlPlane, err := yaml.Marshal(crs.ControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(controlPlane)
	out += "\n---\n"

	awsControlPlaneMachine, err := yaml.Marshal(crs.ControlPlaneMachineTemplate)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(awsControlPlaneMachine)
	out += "\n---\n"

	fmt.Printf("%s\n", out)

	return nil
}
