package capi

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	giantswarmawsalpha3 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
)

func awsMachineTemplateCPName(clusterID string) string {
	return fmt.Sprintf("%s-control-plane", clusterID)
}
func transformAWSMachineTemplateCP(cp *giantswarmawsalpha3.AWSControlPlane, clusterID string) *capiawsv1alpha3.AWSMachineTemplate {
	sshKeyName := "vaclav"
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
					InstanceType: cp.Spec.InstanceType,
					SSHKeyName: &sshKeyName,

				},
			},
		},
	}

	return machineTemplate
}
