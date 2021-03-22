# how to use

Guide how migrate cluster `${CLUSTER_ID}` from GS MC `${OLD_MC}` to CAPA MC `${CAPI_MC}`
## prereq
- create kubeconfig and vaultconfig for the old MC (where GS cluster is located)
  - `opsctl create kubeconfig -i ${OLD_MC}`
  - `opsctl create vaultconfig -i ${OLD_MC}`
  - export the variables from the vaultconfig
- create kubeconfig for the new CAPI MC  - `${CAPI_MC}`
- export AWS credentials for the AWS account where the cluster is located
```
export AWS_ACCESS_KEY_ID=AKIARHXXXXXXX
export AWS_SECRET_ACCESS_KEY=2hgUZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ
export AWS_REGION=eu-west-1
```

## run the commands in the folowing order:
before running be sure to set current kubernetes contex to `${OLD_MC}`
```
./aws-gs-to-capi create cp --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
./aws-gs-to-capi update dns --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
#  now you need to remove manifests from old masters (specialy `api server` and `controller manager`), atm this is not automated
./aws-gs-to-capi create np --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
```


## how clean:
clean CAPA components first(you need MC CAPI kubeconfig) and than delete cluster via GS api
```
./aws-gs-to-capi delete np --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
./aws-gs-to-capi delete dns --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
./aws-gs-to-capi delete cp --context=${CAPI_MC} --cluster-id=${CLUSTER_ID}
gsctl delete cluster ${CLUSTER_ID}
```

