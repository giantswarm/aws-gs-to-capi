package capi

import (
	"context"
	"encoding/base64"
	"fmt"
	v1alpha32 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1alpha3"
	"sigs.k8s.io/cluster-api/exp/api/v1alpha3"

	"github.com/giantswarm/aws-gs-to-capi/vault"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	awsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	capiawsexpv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/exp/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	"github.com/giantswarm/aws-gs-to-capi/ctrlclient"
	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

type Crs struct {
	CustomFiles *v1.Secret
	EtcdCerts   *v1.Secret
	SACerts     *v1.Secret
	CACerts     *v1.Secret

	Cluster                     *apiv1alpha3.Cluster
	AWSCluster                  *awsv1alpha3.AWSCluster
	ControlPlane                *kubeadmv1alpha3.KubeadmControlPlane
	ControlPlaneMachineTemplate *awsv1alpha3.AWSMachineTemplate
	OldControlPlaneMachines     []apiv1alpha3.Machine

	MachinePools []*MachinePoolSpec
}

type MachinePoolSpec struct {
	AWSMachinePool *capiawsexpv1alpha3.AWSMachinePool
	MachinePool    *v1alpha3.MachinePool
	KubeadmConfig  *v1alpha32.KubeadmConfig
}

func TransformGsToCAPICrs(gsCRs *giantswarm.GSClusterCrs, k8sVersion string) (*Crs, error) {
	var err error
	clusterID := gsCRs.AWSCluster.Name
	namespace := gsCRs.AWSCluster.Namespace

	p := CustomFilesParams{
		APIEndpoint:   apiEndpointFromDomain(gsCRs.AWSCluster.Spec.Cluster.DNS.Domain, clusterID),
		ClusterID:     clusterID,
		ETCDEndpoint:  etcdEndpointFromDomain(gsCRs.AWSCluster.Spec.Cluster.DNS.Domain, clusterID),
		EncryptionKey: string(gsCRs.EncryptionKey.Data["encryption"]),
		Namespace:     namespace,
		KubeProxyCA:   base64.StdEncoding.EncodeToString(gsCRs.KubeproxyCerts.Data["ca"]),
		KubeProxyKey:  base64.StdEncoding.EncodeToString(gsCRs.KubeproxyCerts.Data["key"]),
		KubeProxyCrt:  base64.StdEncoding.EncodeToString(gsCRs.KubeproxyCerts.Data["crt"]),
	}

	secret, err := customFilesSecret(p)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cluster := transformCluster(gsCRs)

	awsCluster, err := transformAWSCluster(gsCRs.AWSCluster)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	kubeadmCP := transformKubeAdmControlPlane(gsCRs, k8sVersion)

	cpMachineTemplate, err := transformAWSMachineTemplateCP(gsCRs.AWSControlPlane, clusterID, gsCRs.AWSCluster.Spec.Provider.Region)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	gsCRs.EtcdCerts.Name = etcdCertsName(clusterID)
	gsCRs.EtcdCerts.APIVersion = secret.APIVersion
	gsCRs.EtcdCerts.Kind = secret.Kind
	gsCRs.EtcdCerts.ResourceVersion = ""
	gsCRs.EtcdCerts.UID = ""
	gsCRs.EtcdCerts.Data["tls.crt"] = gsCRs.EtcdCerts.Data["crt"]
	gsCRs.EtcdCerts.Data["tls.key"] = gsCRs.EtcdCerts.Data["key"]

	gsCRs.SACerts.Name = saCertsName(clusterID)
	gsCRs.SACerts.APIVersion = secret.APIVersion
	gsCRs.SACerts.Kind = secret.Kind
	gsCRs.SACerts.ResourceVersion = ""
	gsCRs.SACerts.UID = ""
	gsCRs.SACerts.Data["tls.crt"] = gsCRs.SACerts.Data["crt"]
	gsCRs.SACerts.Data["tls.key"] = gsCRs.SACerts.Data["key"]

	caPrivKey, err := vault.GetVaultCAKey(clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	caCerts := gsCRs.EtcdCerts.DeepCopy()
	caCerts.Name = caCertsName(clusterID)
	caCerts.Data["tls.crt"] = caCerts.Data["ca"]
	caCerts.Data["tls.key"] = []byte(caPrivKey)

	crs := &Crs{
		CustomFiles: &secret,
		EtcdCerts:   gsCRs.EtcdCerts,
		SACerts:     gsCRs.SACerts,
		CACerts:     caCerts,

		Cluster:                     cluster,
		AWSCluster:                  awsCluster,
		ControlPlane:                kubeadmCP,
		ControlPlaneMachineTemplate: cpMachineTemplate,
	}

	for _, md := range gsCRs.AWSMachineDeployments {
		awsmp, err := awsmachinepool(md, gsCRs.AWSCluster.Spec.Provider.Region, clusterID)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		mp := machinePool(md, clusterID, k8sVersion)
		kubeadmConfig := machinePoolKubeAdmConfig(md, clusterID)

		crs.MachinePools = append(crs.MachinePools, &MachinePoolSpec{
			AWSMachinePool: awsmp,
			MachinePool:    mp,
			KubeadmConfig:  kubeadmConfig,
		})
	}

	return crs, nil
}

func CreateControlPlaneResources(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Create(ctx, crs.CustomFiles)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Create(ctx, crs.EtcdCerts)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Create(ctx, crs.SACerts)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Create(ctx, crs.CACerts)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Create(ctx, crs.Cluster)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Create(ctx, crs.AWSCluster)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Create(ctx, crs.ControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Create(ctx, crs.ControlPlaneMachineTemplate)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func CreateNodePoolResources(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, mp := range crs.MachinePools {
		err = ctrl.Create(ctx, mp.AWSMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
		err = ctrl.Create(ctx, mp.KubeadmConfig)
		if err != nil {
			return microerror.Mask(err)
		}
		err = ctrl.Create(ctx, mp.MachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func DeleteNPResources(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, mp := range crs.MachinePools {
		err = ctrl.Delete(ctx, mp.AWSMachinePool)
		err = ctrl.Delete(ctx, mp.KubeadmConfig)
		err = ctrl.Delete(ctx, mp.MachinePool)
	}

	return nil
}

func DeleteCPResources(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Delete(ctx, crs.CustomFiles)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Delete(ctx, crs.EtcdCerts)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Delete(ctx, crs.Cluster)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Delete(ctx, crs.AWSCluster)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Delete(ctx, crs.ControlPlane)
	if err != nil {
		return microerror.Mask(err)
	}
	err = ctrl.Delete(ctx, crs.ControlPlaneMachineTemplate)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}

func etcdEndpointFromDomain(domain string, clusterID string) string {
	return fmt.Sprintf("etcd.%s.k8s.%s", clusterID, domain)
}
func apiEndpointFromDomain(domain string, clusterID string) string {
	return fmt.Sprintf("api.%s.k8s.%s", clusterID, domain)
}
