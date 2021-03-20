package main

import (
	"fmt"
	"github.com/giantswarm/aws-gs-to-capi/dns"
	"os"

	"github.com/giantswarm/microerror"
	flag "github.com/spf13/pflag"

	"github.com/giantswarm/aws-gs-to-capi/capi"
	"github.com/giantswarm/aws-gs-to-capi/giantswarm"
)

type Flag struct {
	AWSRegion  string
	ClusterID  string
	K8sVersion string
	Context    string
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
	flag.StringVar(&f.Context, "context", "", "define in which k8s context the resources should be created")

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		flag.Usage()
		return nil
	}
	flag.Parse()

	if f.Context == "" {
		fmt.Printf("ERROR: target context cannot be empty")
		return nil
	}
	fmt.Printf("\n\n")

	gsCrs, err := giantswarm.FetchCrs(f.ClusterID)
	if err != nil {
		return microerror.Mask(err)
	}

	capiCRs, err := capi.TransformGsToCAPICrs(gsCrs, f.K8sVersion)
	if err != nil {
		return microerror.Mask(err)
	}

	if isCreateAll() {
		err = capi.CreateControlPlaneResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
		err = dns.UpdateAPIDNSToNewELB(capiCRs.Cluster.Name, fmt.Sprintf("%s.k8s.%s", capiCRs.Cluster.Name, gsCrs.AWSCluster.Spec.Cluster.DNS.Domain), f.AWSRegion, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
		err = capi.CreateNodePoolResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isCreateCP() {
		err = capi.CreateControlPlaneResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isCreateNP() {
		err = capi.CreateNodePoolResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isDeleteAll() {
		err = capi.DeleteNPResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
		err = capi.DeleteCPResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
		err = dns.DeleteDNSRecords(capiCRs.Cluster.Name, f.AWSRegion)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isDeleteCP() {
		err = capi.DeleteCPResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isDeleteNP() {
		err = capi.DeleteNPResources(capiCRs, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isUpdateDNS() {
		err = dns.UpdateAPIDNSToNewELB(capiCRs.Cluster.Name, fmt.Sprintf("%s.k8s.%s", capiCRs.Cluster.Name, gsCrs.AWSCluster.Spec.Cluster.DNS.Domain), f.AWSRegion, f.Context)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if isDeleteDNS() {
		err = dns.DeleteDNSRecords(capiCRs.Cluster.Name, f.AWSRegion)
		if err != nil {
			return microerror.Mask(err)
		}
	} else {
		fmt.Printf("Error: Did not found requested command, exiting.\n")
	}
	fmt.Printf("\n\n")

	return nil
}

func isCreateAll() bool {
	return len(os.Args) > 2 && os.Args[1] == "create" && os.Args[2] == "all"
}

func isCreateCP() bool {
	return len(os.Args) > 2 && os.Args[1] == "create" && os.Args[2] == "cp"
}

func isCreateNP() bool {
	return len(os.Args) > 2 && os.Args[1] == "create" && os.Args[2] == "np"
}

func isDeleteAll() bool {
	return len(os.Args) > 2 && os.Args[1] == "delete" && os.Args[2] == "all"
}
func isDeleteCP() bool {
	return len(os.Args) > 2 && os.Args[1] == "delete" && os.Args[2] == "cp"
}
func isDeleteNP() bool {
	return len(os.Args) > 2 && os.Args[1] == "delete" && os.Args[2] == "np"
}

func isUpdateDNS() bool {
	return len(os.Args) > 2 && os.Args[1] == "update" && os.Args[2] == "dns"
}
func isDeleteDNS() bool {
	return len(os.Args) > 2 && os.Args[1] == "delete" && os.Args[2] == "dns"
}
