package capi

import (
	"fmt"
	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	kubeadmapiv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kubeadmtypev1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/v1beta1"

	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"
)

func kubeAdmControlPlaneName(clusterID string) string {
	return fmt.Sprintf("%s-control-plane", clusterID)
}

func transformKubeAdmControlPlane(gsCRs *giantswarm.GSClusterCrs, k8sVersion string) *kubeadmv1alpha3.KubeadmControlPlane {
	replicas := int32(gsCRs.G8sControlPlane.Spec.Replicas)
	clusterID := gsCRs.AWSCluster.Name

	cp := &kubeadmv1alpha3.KubeadmControlPlane{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kubeadmv1alpha3.GroupVersion.String(),
			Kind:       "KubeadmControlPlane",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kubeAdmControlPlaneName(gsCRs.AWSCluster.Name),
			Namespace: gsCRs.AWSCluster.Namespace,
		},
		Spec: kubeadmv1alpha3.KubeadmControlPlaneSpec{
			InfrastructureTemplate: v1.ObjectReference{
				APIVersion: capiawsv1alpha3.GroupVersion.String(),
				Name:       awsMachineTemplateCPName(clusterID),
				Kind:       "AWSMachineTemplate",
			},
			KubeadmConfigSpec: kubeadmapiv1alpha3.KubeadmConfigSpec{

				ClusterConfiguration: &kubeadmtypev1beta1.ClusterConfiguration{
					APIServer: kubeadmtypev1beta1.APIServer{
						ControlPlaneComponent: kubeadmtypev1beta1.ControlPlaneComponent{
							ExtraArgs: map[string]string{
								"cloud-provider": "aws",
							},
						},
						CertSANs: []string{
							gsCRs.Cluster.Status.APIEndpoints[0].Host,
						},
					},
					ControllerManager: kubeadmtypev1beta1.ControlPlaneComponent{
						ExtraArgs: map[string]string{
							"cloud-provider": "aws",
						},
					},
					Etcd: kubeadmtypev1beta1.Etcd{
						Local: &kubeadmtypev1beta1.LocalEtcd{
							DataDir: "/var/lib/etcd/data",
							ExtraArgs: map[string]string{
								"listen-client-urls": "https://0.0.0.0:2379",
							},
							ImageMeta: kubeadmtypev1beta1.ImageMeta{
								ImageTag:        "v3.4.13",
								ImageRepository: "quay.io/giantswarm",
							},
						},
					},
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
						},
						Name: "{{ ds.meta_data.local_hostname }}",
					},
				},
				Files: []kubeadmapiv1alpha3.File{
					{
						Path:  "/etc/systemd/system/etcd3-attach-deps.service",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: unitSecretName(clusterID),
								Key:  unitSecretKey,
							},
						},
					},
				},
				PreKubeadmCommands: []string{
					"systemctl enable etcd3-attach-deps.service",
					"systemctl start etcd3-attach-deps.service",
				},
			},
			Replicas: &replicas,
			Version:  k8sVersion,
		},
	}
	return cp
}
