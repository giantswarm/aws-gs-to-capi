package capi

import (
	"bytes"
	"fmt"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"text/template"
)

const (
	unitTmpl = `[Unit]
Description=Attach etcd dependencies
Requires=network.target
After=network.target

[Service]
Environment="URL=https://github.com/giantswarm/aws-attach-etcd-dep/releases/download/v0.1.0-capi/aws-attach-etcd-dep"
Environment="NAME=%p.service"
Environment="DEVICE=/dev/xvdh"
Environment="FSTYPE=ext4"
Environment="MOUNT_PATH=/var/lib/etcd"
Environment="CLUSTER_ID={{.ClusterID}}"
Type=oneshot
RemainAfterExit=yes
ExecStartPre=/bin/rm -rf /tmp/aws-attach-etcd-dep
ExecStartPre=/usr/bin/wget https://github.com/giantswarm/aws-attach-etcd-dep/releases/download/v0.1.0-capi/aws-attach-etcd-dep -O /tmp/aws-attach-etcd-dep
ExecStartPre=/bin/chmod +x /tmp/aws-attach-etcd-dep
ExecStartPre=/bin/bash -c "/bin/mkdir -p ${MOUNT_PATH}"
{{- if .NVMEDisk }}
ExecStartPre=/bin/bash -c "/bin/ln -s /dev/nvme1n1 ${DEVICE}"
{{- end }}
ExecStart=/bin/bash -c "/tmp/aws-attach-etcd-dep \
      --eni-device-index=1 \
      --eni-device-name=ens4 \
      --eni-tag-key=giantswarm.io/cluster \
      --eni-tag-value=${CLUSTER_ID} \
      --mount-disk=true \
      --mount-disk-path=${MOUNT_PATH} \
      --volume-device-name=${DEVICE} \
      --volume-device-filesystem-type=${FSTYPE} \
      --volume-device-label=etcd \
      --volume-tag-key=giantswarm.io/cluster \
      --volume-tag-value=${CLUSTER_ID}"
ExecStartPost=-/bin/bash -c "/bin/mkdir -p ${MOUNT_PATH}/data && /bin/mv ${MOUNT_PATH}/member ${MOUNT_PATH}/data/"
ExecStartPost=/bin/systemctl daemon-reload
ExecStartPost=/bin/systemctl restart systemd-networkd
ExecStartPost=/bin/bash -c "while [ ! -f /tmp/kubeadm.yaml ]; do sleep 2; done && \
							while [ ! -f  /usr/bin/kubeadm ]; do sleep 2; done && \
							sleep 20s && \
							/usr/bin/kubeadm init --ignore-preflight-errors='DirAvailable--var-lib-etcd-data' --config /tmp/kubeadm.yaml"

[Install]
WantedBy=multi-user.target
`
)

const (
	unitSecretKey      = "unit"
	deviceNameNVME     = "nvme1n1"
)

func renderTemplate(clusterID string, nvmeDisk bool) (string, error) {
	p := struct {
		ClusterID  string
		NVMEDisk bool
	}{
		ClusterID:  clusterID,
		NVMEDisk: nvmeDisk,
	}

	var buff bytes.Buffer
	t := template.Must(template.New("unit").Parse(unitTmpl))

	err := t.Execute(&buff, p)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return buff.String(), nil
}

func unitSecret(clusterID string, namespace string, instanceType string) (v1.Secret, error) {
	content, err := renderTemplate(clusterID, isNVMEDisk(instanceType))
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
	return fmt.Sprintf("%s-etcd3-attach-deps-unit", clusterID)
}

func isNVMEDisk(instanceType string) bool {
	if strings.HasPrefix(instanceType, "m5") {
		return true
	} else {
		return false
	}

}
