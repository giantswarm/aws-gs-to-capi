package capi

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	capiawsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	kubeadmapiv1alpha3 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	kubeadmtypev1beta1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/types/v1beta1"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

func kubeAdmControlPlaneName(clusterID string) string {
	return fmt.Sprintf("%s-control-plane", clusterID)
}

func transformKubeAdmControlPlane(gsCRs *giantswarm.GSClusterCrs, k8sVersion string) *kubeadmv1alpha3.KubeadmControlPlane {
	replicas := int32(1)
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
								"cloud-provider":             "aws",
								"etcd-prefix":                "giantswarm.io",
								"encryption-provider-config": "/etc/kubernetes/encryption/k8s-encryption-config.yaml",
							},
							ExtraVolumes: []kubeadmtypev1beta1.HostPathMount{
								{
									Name:      "encryption",
									HostPath:  "/etc/kubernetes/encryption/",
									MountPath: "/etc/kubernetes/encryption/",
								},
							},
						},
						CertSANs: []string{
							apiEndpointFromDomain(gsCRs.AWSCluster.Spec.Cluster.DNS.Domain, clusterID),
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
								"initial-cluster-state":                          "existing",
								"initial-cluster":                                "$ETCD_INITIAL_CLUSTER",
								"experimental-peer-skip-client-san-verification": "true",
							},
							ImageMeta: kubeadmtypev1beta1.ImageMeta{
								ImageTag:        "v3.4.14",
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
					LocalAPIEndpoint: kubeadmtypev1beta1.APIEndpoint{
						BindPort: 443,
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
								Name: customFilesSecretName(clusterID),
								Key:  migrationScriptKey,
							},
						},
					},
					{
						Path:  "/etc/kubernetes/encryption/k8s-encryption-config.yaml",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: customFilesSecretName(clusterID),
								Key:  encryptionKeyKey,
							},
						},
					},
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
					{
						Path:  "/etc/kubernetes/pki/ca.crt",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: caCertsName(clusterID),
								Key:  "tls.crt",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/ca.key",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: caCertsName(clusterID),
								Key:  "tls.key",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/ca.key",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: caCertsName(clusterID),
								Key:  "tls.key",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/ca.crt",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: caCertsName(clusterID),
								Key:  "tls.crt",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/sa.pub",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: saCertsName(clusterID),
								Key:  "tls.crt",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/sa.key",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: saCertsName(clusterID),
								Key:  "tls.key",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/old.key",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: etcdCertsName(clusterID),
								Key:  "key",
							},
						},
					},
					{
						Path:  "/etc/kubernetes/pki/etcd/old.crt",
						Owner: "root:root",
						ContentFrom: &kubeadmapiv1alpha3.FileSource{
							Secret: kubeadmapiv1alpha3.SecretFileSource{
								Name: etcdCertsName(clusterID),
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
