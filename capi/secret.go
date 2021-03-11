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
	unitTmpl = `#!/bin/sh

# create etcd ca bundle
cat /etc/kubernetes/pki/etcd/ca.crt  /etc/kubernetes/pki/etcd/old-etcd-ca.pem> /etc/kubernetes/pki/etcd/ca-bundle.pem

# get ETCDCTL
DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download
ETCD_VER=v3.4.13
rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
rm -rf /tmp/etcd-download-test && mkdir -p /tmp/etcd-download-test
curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp/etcd-download-test --strip-components=1
rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
/tmp/etcd-download-test/etcd --version
/tmp/etcd-download-test/etcdctl version

# get machine IP
IP=$(ip route | grep default | awk '{print $9}')

# add new member to the old etcd cluster
new_cluster=$(/tmp/etcd-download-test/etcdctl \
	--cacert=/etc/kubernetes/pki/etcd/old-etcd-ca.pem \
	--key=/etc/kubernetes/pki/etcd/old-etcd-key.pem \
	--cert=/etc/kubernetes/pki/etcd/old-etcd-cert.pem \
	--endpoints=https://{{.ETCDEndpoint}}:2379 \
	--peer-urls="https://${IP}:2380" \
	member \
	add \
	$(hostname) | grep 'ETCD_INITIAL_CLUSTER=')

# export ETCD_INITIAL_CLUSTER env for later envsubst command
export ${new_cluster}

# copy tmpl
cp /tmp/kubeadm.yaml /tmp/kubeadm.yaml.tmpl

# fill the initial cluster variable into kubeadm config
envsubst < /tmp/kubeadm.yaml.tmpl > /tmp/kubeadm.yaml

`
)

const (
	unitSecretKey = "unit"
)

func renderTemplate(etcdEndpoint string) (string, error) {
	p := struct {
		ETCDEndpoint string
	}{
		ETCDEndpoint: etcdEndpoint,
	}

	var buff bytes.Buffer
	t := template.Must(template.New("unit").Parse(unitTmpl))

	err := t.Execute(&buff, p)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return buff.String(), nil
}

func unitSecret(clusterID string, namespace string, etcdEndpoint string) (v1.Secret, error) {
	content, err := renderTemplate(etcdEndpoint)
	if err != nil {
		return v1.Secret{}, microerror.Mask(err)
	}

	o := v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      unitSecretName(clusterID),
			Namespace: namespace,
		},
		Data: map[string][]byte{
			unitSecretKey: []byte(content),
		},
	}

	return o, nil
}

func unitSecretName(clusterID string) string {
	return fmt.Sprintf("%s-custom-init-script", clusterID)
}

func certsSecretName(clusterID string) string {
	return fmt.Sprintf("%s-etcd-certs", clusterID)
}
