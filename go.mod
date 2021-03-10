module github.com/giantswarm/aws-gs-to-capi

go 1.15

require (
	github.com/aws/aws-sdk-go v1.37.25
	github.com/giantswarm/apiextensions v0.4.20
	github.com/giantswarm/microerror v0.3.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/tools v0.0.0-20191029190741-b9c20aec41a5
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.9
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	sigs.k8s.io/cluster-api v0.3.12
	sigs.k8s.io/cluster-api-provider-aws v0.6.4
	sigs.k8s.io/yaml v1.2.0
)
