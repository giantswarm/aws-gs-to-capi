package capi

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	awsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmbootstrapv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
)

func oldControlPlaneMachines(clusterID string, namespace string, masters int) []apiv1alpha3.Machine {
	var o []apiv1alpha3.Machine

	for i := 1; i <= masters; i++ {
		m := apiv1alpha3.Machine{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apiv1alpha3.GroupVersion.String(),
				Kind:       "Machine",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-control-plane-old-%d", clusterID, i),
				Namespace: namespace,
				Labels: map[string]string{
					"cluster.x-k8s.io/cluster-name":  clusterID,
					"cluster.x-k8s.io/control-plane": "",
				},
				Annotations: map[string]string{
					"cluster.x-k8s.io/paused": "true",
				},
				OwnerReferences: []metav1.OwnerReference{
					{
						Name:       kubeAdmControlPlaneName(clusterID),
						Kind:       "KubeadmControlPlane",
						APIVersion: kubeadmv1alpha3.GroupVersion.String(),
					},
				},
			},
			Spec: apiv1alpha3.MachineSpec{
				ClusterName: clusterID,
				Bootstrap: apiv1alpha3.Bootstrap{
					ConfigRef: &v1.ObjectReference{
						APIVersion: kubeadmbootstrapv1alpha3.GroupVersion.String(),
						Kind:       "KubeadmConfig",
						Name:       fmt.Sprintf("%s-control-plane-old-%d", clusterID, i),
						UID:        "745fec2c-33e3-4989-bf9c-11c9309c9c99",
					},
				},
				InfrastructureRef: v1.ObjectReference{
					Name:       "fake-ref",
					APIVersion: awsv1alpha3.GroupVersion.String(),
					Kind:       "AWSMachine",
					Namespace:  namespace,
				},
				ProviderID: aws.String("aws:///eu-west-1c/i-fakeid097jnw1"),
			},
		}

		o = append(o, m)
	}

	return o
}

func fakeKubeAdmConfig(clusterID string, namespace string, masterID int) *kubeadmbootstrapv1alpha3.KubeadmConfig {
	return &kubeadmbootstrapv1alpha3.KubeadmConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kubeadmbootstrapv1alpha3.GroupVersion.String(),
			Kind:       "KubeadmConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-control-plane-old-%d", clusterID, masterID),
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name":  clusterID,
				"cluster.x-k8s.io/control-plane": "",
			},
			Annotations: map[string]string{
				"cluster.x-k8s.io/paused": "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       kubeAdmControlPlaneName(clusterID),
					Kind:       "KubeadmControlPlane",
					APIVersion: kubeadmv1alpha3.GroupVersion.String(),
				},
				{
					Name:       fmt.Sprintf("%s-control-plane-old-%d", clusterID, masterID),
					Kind:       "AWSMachine",
					APIVersion: apiv1alpha3.GroupVersion.String(),
				},
			},
		},
	}
}

func fakeAWSMachine(clusterID string, namespace string, masterID int) *awsv1alpha3.AWSMachine {
	return &awsv1alpha3.AWSMachine{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiv1alpha3.GroupVersion.String(),
			Kind:       "AWSMachine",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-control-plane-old-%d", clusterID, masterID),
			Namespace: namespace,
			Labels: map[string]string{
				"cluster.x-k8s.io/cluster-name":  clusterID,
				"cluster.x-k8s.io/control-plane": "",
			},
			Annotations: map[string]string{
				"cluster.x-k8s.io/paused": "true",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       kubeAdmControlPlaneName(clusterID),
					Kind:       "KubeadmControlPlane",
					APIVersion: kubeadmv1alpha3.GroupVersion.String(),
				},
			},
		},
	}
}

func FillMachineStatus(m *apiv1alpha3.Machine, i int) {
	m.Status = apiv1alpha3.MachineStatus{
		NodeRef: &v1.ObjectReference{
			APIVersion:      v1.SchemeGroupVersion.String(),
			Name:            fmt.Sprintf("etcd%d", i),
			Kind:            "Node",
			UID:             "745fec2c-33e3-4989-bf9c-11c9309c9c99",
			ResourceVersion: "12321313213",
			FieldPath:       "/unknown",
		},
		Phase:               "Running",
		BootstrapReady:      true,
		InfrastructureReady: true,
		Version:             aws.String("1.19.4"),
	}

}
