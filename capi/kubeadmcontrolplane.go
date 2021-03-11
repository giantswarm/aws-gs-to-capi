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
						//CertSANs: []string{
						//	gsCRs.Cluster.Status.APIEndpoints[0].Host,
						//},
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
								"peer-cert-file":        "/etc/kubernetes/pki/etcd/old-etcd-cert.pem",
								"peer-key-file":         "/etc/kubernetes/pki/etcd/old-etcd-key.pem",
								"peer-trusted-ca-file":  "/etc/kubernetes/pki/etcd/ca-bundle.pem",
								"initial-cluster-state": "existing",
								"initial-cluster":       "$ETCD_INITIAL_CLUSTER",
								"trusted-ca-file":       "/etc/kubernetes/pki/etcd/ca-bundle.pem",
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
						Path:  "/migration/join-existing-cluster.sh",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: unitSecretName(clusterID),
								Key:  unitSecretKey,
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/old-etcd-ca.pem",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: certsSecretName(clusterID),
								Key:  "ca",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/old-etcd-key.pem",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: certsSecretName(clusterID),
								Key:  "key",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/old-etcd-cert.pem",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: certsSecretName(clusterID),
								Key:  "crt",
							},
						},
					},
				},
				PreKubeadmCommands: []string{
					"/bin/sh /migration/join-existing-cluster.sh",
				},
			},
			Replicas: &replicas,
			Version:  k8sVersion,
		},
	}
	return cp
}
