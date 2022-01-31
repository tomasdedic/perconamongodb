## Instalace AKS
```sh
  # deploy AKS cluster
  az account set --subscription 19f261f0-11a0-4dca-bd1b-f57397845a5f
  az aks get-credentials --resource-group Percona-rg --name percona-poc
  #disable autoscale and scale up to 3
  az aks update --resource-group Percona-rg --name percona-poc --disable-cluster-autoscaler
  az aks show --resource-group Percona-rg --name percona-poc --query agentPoolProfiles
  az aks scale --resource-group Percona-rg --name percona-poc --node-count 3 --nodepool-name agentpool
```
```sh
#start and stop cluster to save money
az aks stop --name percona-poc --resource-group Percona-rg
az aks start --name percona-poc --resource-group Percona-rg
```

