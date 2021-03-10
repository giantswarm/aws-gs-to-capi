package capi

import (
	"fmt"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

func clusterName(clusterID string) string {
	return fmt.Sprintf("%s", clusterID)
}
func transformCluster(gsCRs *giantswarm.GSClusterCrs) *apiv1alpha3.Cluster {
	clusterID := gsCRs.AWSCluster.Name

	cluster := &apiv1alpha3.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: apiv1alpha3.GroupVersion.String(),
		},

		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName(clusterID),
			Namespace: gsCRs.AWSCluster.Namespace,
		},
		Spec: apiv1alpha3.ClusterSpec{
			ClusterNetwork: &apiv1alpha3.ClusterNetwork{
				Pods: &apiv1alpha3.NetworkRanges{
					CIDRBlocks: []string{gsCRs.AWSCluster.Spec.Provider.Pods.CIDRBlock},
				},
				//Services: &apiv1alpha3.NetworkRanges{
					//			CIDRBlocks: []string{gsCRs.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0]},
				//},
			},
			ControlPlaneRef: &v1.ObjectReference{
				APIVersion: kubeadmv1alpha3.GroupVersion.String(),
				Name:       kubeAdmControlPlaneName(clusterID),
				Kind:       "KubeadmControlPlane",
			},
			InfrastructureRef: &v1.ObjectReference{
				APIVersion: capiawsv1alpha3.GroupVersion.String(),
				Name:       clusterID,
				Kind:       "AWSCluster",
			},
		},
	}

	return cluster

}
