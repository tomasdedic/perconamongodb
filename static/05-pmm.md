# Instalace PMM server
zdroj:
[Instalace SERVER](https://www.percona.com/blog/2020/07/23/using-percona-kubernetes-operators-with-percona-monitoring-and-management)  
```
k create namespace percona-pmm
k config set-context $(k config current-context) --namespace=percona-pmm
k apply -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.11.0/deploy/bundle.yaml --namespace=percona-pmm
```
## Přidání helm repa
```
helm repo add percona https://percona-charts.storage.googleapis.com
helm repo update
```

## Instalace helm
```
helm install monitoring percona/pmm-server --set platform=kubernetes --version 2.7.0 --set "credentials.password=admin"
```

## vytvoření secret
Zkontrolovat zda jsou nastaveny User a Password
 ```
 PMM_SERVER_USER: admin
 PMM_SERVER_PASSWORD: admin
 ```
apply secret 
```
k apply -f tasks/deploy/secrets.yaml
```


# Instalace PMM client
zdroj: [Instalace CLIENT](https://www.percona.com/doc/kubernetes-operator-for-psmongodb/monitoring.html#installing-pmm-server)  
instalace klienta se nastavuje v `tasks/pmm/cr.yaml` 
- pmm.enabled=true

## Apply custom resource
CR  
```
k apply -f tasks/pmm/cr.yaml
```

## kontrola logů
```
k logs mongo1-rs0-0 -c pmm-client
```
## přihlášení do konzole
 zjistit ip adresu service  

 ```
 k get svc 
 ```
 result 
 
 ```
NAME                 TYPE           CLUSTER-IP     EXTERNAL-IP      PORT(S)           AGE
mongo1-cfg           ClusterIP      None           <none>           27017/TCP         81m
mongo1-mongos        LoadBalancer   10.0.253.127   20.101.170.206   27017:31414/TCP   78m
mongo1-rs0           ClusterIP      None           <none>           27017/TCP         81m
monitoring-service   LoadBalancer   10.0.5.20      20.101.169.202   443:31051/TCP     81m
```

Adresa pro přihlášení  
https://20.101.169.202

# Instalace PMM na již existujícim prostředí

## Přidání helm repa
```
helm repo add percona https://percona-charts.storage.googleapis.com
helm repo update
```

## Instalace helm
```
helm install monitoring percona/pmm-server --set platform=kubernetes --version 2.7.0 --set "credentials.password=admin123456" -n percona
```

## vytvoření secret
Zkontrolovat zda jsou nastaveny User a Password secret `my-cluster-name-secrets` zadané při instalaci helm `--set "credentials.password=admin123456"` 
 ```
 PMM_SERVER_USER: admin
 PMM_SERVER_PASSWORD: admin123456
 ```

# Povolení PMM clienta
povolit pmm clienta v `tasks/deploy/cr.yaml` 
nebo přímo editovat CR `PerconaServerMongoDB`  
```   
   pmm:
    enabled: true
```
## Přihlášení do konzole
 zjistit ip adresu service  
```
k get svc -n percona
NAME                 TYPE           CLUSTER-IP    EXTERNAL-IP     PORT(S)           AGE
minio                ClusterIP      10.0.110.84   <none>          9000/TCP          24d
minio-console        ClusterIP      10.0.70.142   <none>          9001/TCP          24d
mongo2-cfg           ClusterIP      None          <none>          27017/TCP         44h
mongo2-mongos        LoadBalancer   10.0.93.168   20.71.82.140    27017:32645/TCP   43h
mongo2-rs0           ClusterIP      None          <none>          27017/TCP         44h
monitoring-service   LoadBalancer   10.0.91.123   20.103.245.65   443:31030/TCP     12m
```
Adresa pro přihlášení  
https://20.103.245.65  
nebo interně   
https://monitoring-service.percona.svc.cluster.local - (netestováno)