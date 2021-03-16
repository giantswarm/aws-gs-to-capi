package capi

const (
	unitTmpl = `#!/bin/sh
# set proper hostname
hostnamectl set-hostname $(curl http://169.254.169.254/latest/meta-data/local-hostname)

# create etcd ca bundle
cat /etc/kubernetes/pki/etcd/ca.crt /etc/kubernetes/pki/etcd/old-etcd-ca.pem > /etc/kubernetes/pki/etcd/ca-bundle.pem

# get ETCDCTL
DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download
ETCD_VER=v3.4.13
rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
rm -rf /tmp/etcd && mkdir -p /tmp/etcd
curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
tar xzvf /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz -C /tmp/etcd --strip-components=1
rm -f /tmp/etcd-${ETCD_VER}-linux-amd64.tar.gz
/tmp/etcd/etcdctl version

# get machine IP
IP=$(ip route | grep default | awk '{print $9}')

# add new member to the old etcd cluster
while ! new_cluster=$(/tmp/etcd/etcdctl \
	--cacert=/etc/kubernetes/pki/etcd/old-etcd-ca.pem \
	--key=/etc/kubernetes/pki/etcd/old-etcd-key.pem \
	--cert=/etc/kubernetes/pki/etcd/old-etcd-cert.pem \
	--endpoints=https://{{.ETCDEndpoint}}:2379 \
	--peer-urls="https://${IP}:2380" \
	member \
	add \
	$(hostname -A) | grep 'ETCD_INITIAL_CLUSTER=')
do
	echo "retrying in 2s"
	sleep 2s
done

echo "successfully added a new member to the old etcd cluster"

# export ETCD_INITIAL_CLUSTER env for later envsubst command
export ${new_cluster}

# copy tmpl
cp /tmp/kubeadm.yaml /tmp/kubeadm.yaml.tmpl

sed -e '/external/,+4d' /tmp/kubeadm.yaml.tmpl

# fill the initial cluster variable into kubeadm config
envsubst < /tmp/kubeadm.yaml.tmpl > /tmp/kubeadm.yaml

`
)

const encryptionConfigTmpl = `kind: EncryptionConfig
apiVersion: v1
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: {{.EncryptionKey}}
    - identity: {}`

const kubeConfigTmpl = `apiVersion: v1
kind: Config
users:
- name: proxy
  user:
    client-certificate-data: {{.KubeProxyCrt}}
    client-key-data: {{.KubeProxyKey}}
clusters:
- name: local
  cluster:
    certificate-authority-data: {{.KubeProxyCA}}
    server: https://{{.APIEndpoint}}
contexts:
- context:
    cluster: local
    user: proxy
  name: service-account-context
current-context: service-account-context`

const kubeProxyConfig = `apiVersion: kubeproxy.config.k8s.io/v1alpha1
clientConnection:
  kubeconfig: /etc/kubernetes/kubeconfig/kube-proxy.yaml
kind: KubeProxyConfiguration
mode: iptables
metricsBindAddress: 0.0.0.0:10249`
