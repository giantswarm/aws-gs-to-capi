package capi

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
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
					DataSecretName: aws.String("test"),
				},
			},
			Status: apiv1alpha3.MachineStatus{
				NodeRef: &v1.ObjectReference{
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       fmt.Sprintf("etcd%d", i),
					Kind:       "Node",
				},
			},
		}

		o = append(o, m)
	}

	return o
}
