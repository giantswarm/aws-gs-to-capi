package dns

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/service/elb"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/giantswarm/microerror"
	awsv1alpha3 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/aws-gs-to-capi/ctrlclient"
)

func UpdateAPIDNSToNewELB(clusterID string, dnsDomain string, region string, k8sContext string) error {
	lbDNSName, lbName, err := waitForAPIELBName(clusterID, k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	err = updateAPIDNS(dnsDomain, lbName, lbDNSName, region)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func waitForAPIELBName(clusterID string, k8sContext string) (string, string, error) {
	ctrlClient, err := ctrlclient.GetCtrlClient(k8sContext)
	if err != nil {
		return "", "", microerror.Mask(err)
	}
	var awsCluster awsv1alpha3.AWSCluster

	waitCounter := 0

	ctx := context.Background()
	for {
		err := ctrlClient.Get(ctx,
			ctrl.ObjectKey{
				Name:      clusterID,
				Namespace: "default",
			},
			&awsCluster,
		)
		if err != nil {
			return "", "", microerror.Mask(err)
		}

		if awsCluster.Status.Network.APIServerELB.DNSName != "" {
			fmt.Printf("Fetched new API ELB DNS '%s'\n", awsCluster.Status.Network.APIServerELB.DNSName)
			return awsCluster.Status.Network.APIServerELB.DNSName, awsCluster.Status.Network.APIServerELB.Name, nil
		} else {
			fmt.Printf("API  DNS name is not ready yet, sleeping for 10s ...\n")
			time.Sleep(time.Second * 10)
		}
		waitCounter += 1
		if waitCounter >= 60 {
			fmt.Printf("API ELB DNS name is not ready after 600s, failure, exiting\n")
			return "", "", microerror.Maskf(nil, "API ELB DNS name is not ready after 600s")
		}
	}
}

func updateAPIDNS(dnsDomain string, lbName string, lbDNS string, lbRegion string) error {
	awsSession, err := getAWSSession(lbRegion)
	if err != nil {
		return microerror.Mask(err)
	}

	elbCLient := elb.New(awsSession)
	{
		/*
			i := &elb.DeleteLoadBalancerListenersInput{
				LoadBalancerName:  aws.String(lbName),
				LoadBalancerPorts: []*int64{aws.Int64(443)}}
			_, err := elbCLient.DeleteLoadBalancerListeners(i)
			if err != nil {
				return microerror.Mask(err)
			}
			i2 := &elb.CreateLoadBalancerListenersInput{
				LoadBalancerName: aws.String(lbName),
				Listeners: []*elb.Listener{
					{
						LoadBalancerPort: aws.Int64(443),
						InstancePort:     aws.Int64(443),
						InstanceProtocol: aws.String("TCP"),
						Protocol:         aws.String("TCP"),
					},
				}}
			_, err = elbCLient.CreateLoadBalancerListeners(i2)
			if err != nil {
				return microerror.Mask(err)
			}

			i3 := &elb.ConfigureHealthCheckInput{
				LoadBalancerName: aws.String(lbName),
				HealthCheck: &elb.HealthCheck{
					HealthyThreshold:   aws.Int64(2),
					Interval:           aws.Int64(5),
					Timeout:            aws.Int64(3),
					UnhealthyThreshold: aws.Int64(1),
					Target:             aws.String("SSL:443"),
				}}

			_, err = elbCLient.ConfigureHealthCheck(i3)
			if err != nil {
				return microerror.Mask(err)
			}

			fmt.Printf("Reconfigured ELB to forward traffic to port 443\n")
		*/
	}

	for {
		i := &elb.DescribeInstanceHealthInput{LoadBalancerName: aws.String(lbName)}

		o, err := elbCLient.DescribeInstanceHealth(i)
		if err != nil {
			return microerror.Mask(err)
		}

		if len(o.InstanceStates) > 0 && *o.InstanceStates[0].State == "InService" {
			fmt.Printf("New ELB has healthy instances\n")
			break
		} else {
			fmt.Printf("Waiting for healthy intances for the new ELB \n")
			time.Sleep(time.Second * 10)
		}
	}

	olb, err := elbCLient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{LoadBalancerNames: aws.StringSlice([]string{lbName})})
	if err != nil {
		return microerror.Mask(err)
	}

	r53Client := route53.New(awsSession)

	i := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(fmt.Sprintf("%s.", dnsDomain)),
	}

	o, err := r53Client.ListHostedZonesByName(i)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, hz := range o.HostedZones {
		if *hz.Name != fmt.Sprintf("%s.", dnsDomain) {
			continue
		}
		i2 := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: hz.Id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action: aws.String(route53.ChangeActionUpsert),
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name: aws.String(fmt.Sprintf("api.%s", dnsDomain)),
							Type: aws.String("A"),
							AliasTarget: &route53.AliasTarget{
								DNSName:              aws.String(lbDNS),
								EvaluateTargetHealth: aws.Bool(true),
								HostedZoneId:         olb.LoadBalancerDescriptions[0].CanonicalHostedZoneNameID,
							},
						},
					},
				},
			},
		}

		fmt.Printf("updating DNS record '%s' to '%s' in zone  '%s' \n", fmt.Sprintf("api.%s", dnsDomain), lbDNS, *hz.Id)
		_, err := r53Client.ChangeResourceRecordSets(i2)
		if err != nil {
			return microerror.Mask(err)
		}

	}
	return nil
}

func DeleteDNSRecords(clusterID string, dnsDomain string, lbRegion string, k8sContext string) error {
	lbDNS, lbName, err := waitForAPIELBName(clusterID, k8sContext)
	if err != nil {
		return microerror.Mask(err)
	}

	awsSession, err := getAWSSession(lbRegion)
	if err != nil {
		return microerror.Mask(err)
	}
	elbCLient := elb.New(awsSession)
	olb, err := elbCLient.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{LoadBalancerNames: aws.StringSlice([]string{lbName})})
	if err != nil {
		return microerror.Mask(err)
	}

	r53Client := route53.New(awsSession)

	i := &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(fmt.Sprintf("%s.", dnsDomain)),
	}

	o, err := r53Client.ListHostedZonesByName(i)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, hz := range o.HostedZones {
		if *hz.Name != fmt.Sprintf("%s.", dnsDomain) {
			continue
		}
		i2 := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: hz.Id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: []*route53.Change{
					{
						Action: aws.String(route53.ChangeActionDelete),
						ResourceRecordSet: &route53.ResourceRecordSet{
							Name: aws.String(fmt.Sprintf("api.%s", dnsDomain)),
							Type: aws.String("A"),
							AliasTarget: &route53.AliasTarget{
								DNSName:              aws.String(lbDNS),
								EvaluateTargetHealth: aws.Bool(true),
								HostedZoneId:         olb.LoadBalancerDescriptions[0].CanonicalHostedZoneNameID,
							},
						},
					},
				},
			},
		}

		o, err := r53Client.ChangeResourceRecordSets(i2)
		if err != nil {
			return microerror.Mask(err)
		}
		fmt.Printf("deleted record %s \n", o.ChangeInfo.String())

	}
	return nil
}

func getAWSSession(region string) (*session.Session, error) {
	awsSession, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	return awsSession, nil
}
