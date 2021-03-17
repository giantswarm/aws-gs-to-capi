package capi

import (
	"bytes"
	"fmt"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"text/template"
)

const (
	migrationScriptKey     = "migration"
	encryptionKeyKey       = "encryption"
	kubeProxyKubeconfigKey = "kubeproxy-kubeconfig"
	kubeProxyConfigKey     = "kubeproxy-config"
	vaultCAPrivateKeyKey   = "vaultca-private-key"
)

type CustomFilesParams struct {
	APIEndpoint   string
	ETCDEndpoint  string
	EncryptionKey string
	ClusterID     string
	Namespace     string
	KubeProxyCA   string
	KubeProxyKey  string
	KubeProxyCrt  string
}

func customFilesSecret(params CustomFilesParams) (v1.Secret, error) {
	migrationScriptContent, err := renderTemplate(unitTmpl, params)
	if err != nil {
		return v1.Secret{}, microerror.Mask(err)
	}

	encryptionKeyContent, err := renderTemplate(encryptionConfigTmpl, params)
	if err != nil {
		return v1.Secret{}, microerror.Mask(err)
	}

	kubeProxyKubeconfigContent, err := renderTemplate(kubeConfigTmpl, params)
	if err != nil {
		return v1.Secret{}, microerror.Mask(err)
	}

	o := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      customFilesSecretName(params.ClusterID),
			Namespace: params.Namespace,
		},
		Data: map[string][]byte{
			migrationScriptKey:     []byte(migrationScriptContent),
			encryptionKeyKey:       []byte(encryptionKeyContent),
			kubeProxyKubeconfigKey: []byte(kubeProxyKubeconfigContent),
			kubeProxyConfigKey:     []byte(kubeProxyConfig),
		},
	}

	return o, nil
}

func renderTemplate(tmpl string, params interface{}) (string, error) {
	var buff bytes.Buffer
	t := template.Must(template.New("tmpl").Parse(tmpl))

	err := t.Execute(&buff, params)
	if err != nil {
		return "", microerror.Mask(err)
	}
	return buff.String(), nil
}

func customFilesSecretName(clusterID string) string {
	return fmt.Sprintf("%s-custom-files", clusterID)
}

func etcdCertsName(clusterID string) string {
	return fmt.Sprintf("%s-etcd", clusterID)
}

func saCertsName(clusterID string) string {
	return fmt.Sprintf("%s-sa", clusterID)
}

func caCertsName(clusterID string) string {
	return fmt.Sprintf("%s-ca", clusterID)
}
