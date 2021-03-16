package capi

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	awsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	apiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	kubeadmv1alpha3 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1alpha3"

	ctrlv1 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/aws-gs-to-capi/ctrlclient"
	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

type Crs struct {
	UnitSecret *v1.Secret
	EtcdCerts  *v1.Secret

	Cluster                     *apiv1alpha3.Cluster
	AWSCluster                  *awsv1alpha3.AWSCluster
	ControlPlane                *kubeadmv1alpha3.KubeadmControlPlane
	ControlPlaneMachineTemplate *awsv1alpha3.AWSMachineTemplate
	OldControlPlaneMachines     []apiv1alpha3.Machine

	MachineDeployments         []*apiv1alpha3.MachineDeployment
	MachineDeploymentTemplates []*awsv1alpha3.AWSMachineTemplate
}

func TransformGsToCAPICrs(gsCRs *giantswarm.GSClusterCrs, k8sVersion string) (*Crs, error) {
	var err error
	clusterID := gsCRs.AWSCluster.Name
	namespace := gsCRs.AWSCluster.Namespace

	p := CustomFilesParams{
		APIEndpoint:   apiEndpointFromDomain(gsCRs.AWSCluster.Spec.Cluster.DNS.Domain, clusterID),
		ClusterID:     clusterID,
		ETCDEndpoint:  etcdEndpointFromDomain(gsCRs.AWSCluster.Spec.Cluster.DNS.Domain, clusterID),
		EncryptionKey: string(gsCRs.EtcdEncryptionKey.Data["encryption"]),
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

	gsCRs.EtcdCerts.Name = oldEtcdCertsSecretName(clusterID)
	gsCRs.EtcdCerts.APIVersion = secret.APIVersion
	gsCRs.EtcdCerts.Kind = secret.Kind
	gsCRs.EtcdCerts.ResourceVersion = ""
	gsCRs.EtcdCerts.UID = ""

	oldCpMachines := oldControlPlaneMachines(clusterID, namespace, gsCRs.G8sControlPlane.Spec.Replicas)

	crs := &Crs{
		UnitSecret: &secret,
		EtcdCerts:  gsCRs.EtcdCerts,

		Cluster:                     cluster,
		AWSCluster:                  awsCluster,
		ControlPlane:                kubeadmCP,
		ControlPlaneMachineTemplate: cpMachineTemplate,
		OldControlPlaneMachines:     oldCpMachines,
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

	certs, err := yaml.Marshal(crs.EtcdCerts)
	if err != nil {
		return microerror.Mask(err)
	}
	out += string(certs)
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

	for _, old := range crs.OldControlPlaneMachines {
		m, err := yaml.Marshal(old)
		if err != nil {
			return microerror.Mask(err)
		}
		out += string(m)
		out += "\n---\n"
	}

	fmt.Printf("%s\n", out)

	return nil
}

func CreateResourcesInTargetK8s(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Create(ctx, crs.UnitSecret)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Create(ctx, crs.EtcdCerts)
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

	for i, old := range crs.OldControlPlaneMachines {
		old.OwnerReferences[0].UID = crs.ControlPlane.UID
		err = ctrl.Create(ctx, &old)
		if err != nil {
			return microerror.Mask(err)
		}

		time.Sleep(time.Second * 2)
		err = ctrl.Get(ctx,
			ctrlv1.ObjectKey{
				Name:      old.Name,
				Namespace: old.Namespace,
			},
			&old)
		if err != nil {
			return microerror.Mask(err)
		}
		FillMachineStatus(&old, i+1)
		err = ctrl.Status().Update(ctx, &old)
		if err != nil {
			return microerror.Mask(err)
		}

		awsMachine := fakeAWSMachine(old.Name, old.Namespace, i)
		err = ctrl.Create(ctx, awsMachine)
		if err != nil {
			return microerror.Mask(err)
		}

		kubeadmConfig := fakeKubeAdmConfig(old.Name, old.Namespace, i)
		err = ctrl.Create(ctx, kubeadmConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func DeleteResourcesInTargetK8s(crs *Crs, k8sContext string) error {
	ctx := context.Background()
	ctrl, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	err = ctrl.Delete(ctx, crs.UnitSecret)
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

	for i, old := range crs.OldControlPlaneMachines {
		old.OwnerReferences[0].UID = crs.ControlPlane.UID
		err = ctrl.Delete(ctx, &old)
		if err != nil {
			return microerror.Mask(err)
		}

		awsMachine := fakeAWSMachine(old.Name, old.Namespace, i)
		err = ctrl.Delete(ctx, awsMachine)
		if err != nil {
			return microerror.Mask(err)
		}

		kubeadmConfig := fakeKubeAdmConfig(old.Name, old.Namespace, i)
		err = ctrl.Delete(ctx, kubeadmConfig)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func etcdEndpointFromDomain(domain string, clusterID string) string {
	return fmt.Sprintf("etcd.%s.k8s.%s", clusterID, domain)
}
func apiEndpointFromDomain(domain string, clusterID string) string {
	return fmt.Sprintf("api.%s.k8s.%s", clusterID, domain)
}
