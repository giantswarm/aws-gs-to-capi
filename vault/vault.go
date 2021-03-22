package vault

import (
	"fmt"

	"github.com/giantswarm/microerror"
	vaultclient "github.com/hashicorp/vault/api"
)

func GetVaultCAKey(clusterID string) (string, error) {
	config := vaultclient.DefaultConfig()
	c, err := vaultclient.NewClient(config)
	if err != nil {
		return "", microerror.Mask(err)
	}

	secret, err := c.Logical().Read(fmt.Sprintf("pki-%s/gimmeallyourlovin", clusterID))
	if err != nil {
		return "", microerror.Mask(err)
	}

	key, ok := secret.Data["private_key"].(string)

	if !ok {
		return "", microerror.Maskf(nil, "failed to convert vault data into string")
	}

	return key, nil
}
