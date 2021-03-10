package main

import (
	"fmt"

	"os"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"

	"github.com/giantswarm/aws-gs-to-capi/capi"
	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

type Flag struct {
	AWSRegion string
	ClusterID string

	K8sVersion string
}

func main() {
	err := mainError()
	if err != nil {
		panic(fmt.Sprintf("%#v", err))
	}
}

func mainError() error {
	var err error

	var f Flag
	flag.StringVar(&f.AWSRegion, "aws-region", "eu-west-1", "AWS Region.")
	flag.StringVar(&f.ClusterID, "cluster-id", "", "GS cluster ID.")
	flag.StringVar(&f.K8sVersion, "k8s-version", "v1.19.4", "Kubernetes version fot the new CAPI cluster")

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		flag.Usage()
		return nil
	}
	flag.Parse()

	gsCrs, err := giantswarm.FetchCrs(f.ClusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	capiCRs, err := capi.TransformGsToCAPICrs(gsCrs, f.K8sVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	err = capi.PrintOutCrs(capiCRs)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
