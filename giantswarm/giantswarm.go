package giantswarm

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"

	awsv1alpha2 "github.com/giantswarm/apiextensions/pkg/apis/infrastructure/v1alpha2"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	corev1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"
)

const (
	defaultNamespace = "default"
)

type GSClusterCrs struct {
	Cluster               *corev1alpha2.Cluster
	AWSCluster            *awsv1alpha2.AWSCluster
	AWSControlPlane       *awsv1alpha2.AWSControlPlane
	AWSMachineDeployments []*awsv1alpha2.AWSMachineDeployment
	G8sControlPlane       *awsv1alpha2.G8sControlPlane

	EtcdCerts         *v1.Secret
	EtcdEncryptionKey *v1.Secret
	KubeproxyCerts    *v1.Secret
}

func FetchCrs(clusterID string) (*GSClusterCrs, error) {
	gsClient, err := ApiClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	awsCluster, err := gsClient.InfrastructureV1alpha2().AWSClusters(defaultNamespace).Get(clusterID, metav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	awsCPs, err := gsClient.InfrastructureV1alpha2().AWSControlPlanes(defaultNamespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("giantswarm.io/cluster=%s", clusterID),
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(awsCPs.Items) != 1 {
		return nil, microerror.Maskf(nil, "expected 1 AWSControlPlane but got %d for cluster id %s", len(awsCPs.Items), clusterID)
	}

	g8scp, err := gsClient.InfrastructureV1alpha2().G8sControlPlanes(defaultNamespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("giantswarm.io/cluster=%s", clusterID),
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if len(g8scp.Items) != 1 {
		return nil, microerror.Maskf(nil, "expected 1 G8sControlPlane but got %d for cluster id %s", len(g8scp.Items), clusterID)
	}

	awsMDs, err := gsClient.InfrastructureV1alpha2().AWSMachineDeployments(defaultNamespace).List(metav1.ListOptions{
		LabelSelector: fmt.Sprintf("giantswarm.io/cluster=%s", clusterID),
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	crs := &GSClusterCrs{
		AWSCluster:      awsCluster,
		AWSControlPlane: &awsCPs.Items[0],
		G8sControlPlane: &g8scp.Items[0],
	}

	for _, md := range awsMDs.Items {
		crs.AWSMachineDeployments = append(crs.AWSMachineDeployments, &md)
	}

	c, err := K8sClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	s, err := c.CoreV1().Secrets(defaultNamespace).Get(fmt.Sprintf("%s-etcd1", clusterID), metav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	crs.EtcdCerts = s

	e, err := c.CoreV1().Secrets(defaultNamespace).Get(fmt.Sprintf("%s-encryption", clusterID), metav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	crs.EtcdEncryptionKey = e

	k, err := c.CoreV1().Secrets(defaultNamespace).Get(fmt.Sprintf("%s-worker", clusterID), metav1.GetOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	crs.KubeproxyCerts = k

	return crs, nil
}

func ApiClient() (*versioned.Clientset, error) {
	home, exists := os.LookupEnv("HOME")
	if !exists {
		home = "/root"
	}

	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := versioned.NewForConfig(config)

	return client, nil
}

func K8sClient() (*kubernetes.Clientset, error) {
	home, exists := os.LookupEnv("HOME")
	if !exists {
		home = "/root"
	}

	configPath := filepath.Join(home, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	client, err := kubernetes.NewForConfig(config)

	return client, nil
}
