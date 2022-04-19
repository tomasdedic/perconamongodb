# Instalace PMM server
[Instalace SERVER](https://www.percona.com/blog/2020/07/23/using-percona-kubernetes-operators-with-percona-monitoring-and-management)  

## Vytvoření secret pro propojení agenta s percona serverem
Zkontrolovat zda jsou nastaveny User a Password

```sh
kb get secrets my-cluster-name-secrets -o yaml|yq e '.data.PMM_SERVER_USER' -|base64 -d &&echo
#PMM_SERVER_USER: admin

kb get secrets my-clister-name-users -o yaml|yq e '.data.PMM_SERVER_PASSWORD' -|base64 -d &&echo
#PMM_SERVER_PASSWORD: admin123456
```
pokud nejsou muzeme pouzit zde
```sh
#bud patch
kubectl patch secret/my-cluster-name-secrets -p '{"data":{"PMM_SERVER_PASSWORD": '$(echo -n admin12346 | base64)'}}'
#nebo prepripravene
kb apply -f tasks/deploy/secrets.yaml
```
## Install PMM SERVER

```sh
kb apply -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.11.0/deploy/bundle.yaml 
helm repo add percona https://percona-charts.storage.googleapis.com
helm repo update
helm install monitoring percona/pmm-server --set "platform=kubernetes" --version 2.7.0 --set "credentials.password=admin123456" --set "persistence.size=100Gi"
```

## Instalace PMM client
[Instalace CLIENT](https://www.percona.com/doc/kubernetes-operator-for-psmongodb/monitoring.html#installing-pmm-server)  
```sh
#patch CR
kb apply -f tasks/pmm/cr.yaml
```
Dojde k injektnuti containeru s PMM clientem

## Connection
```sh
MONGO_CONN=$(kb get svc monitoring-service -o yaml|yq e '.status.loadBalancer.ingress[0].ip' -)
echo "https://$MONGO_CONN"
```

## Profiling
```sh
operationProfiling:
  mode: all
  slowOpThresholdMs: 200
  rateLimit: 100 
```
```sh
MONGO_EP=$(kb get svc mongo1-mongos -o yaml|yq e '.status.loadBalancer.ingress[0].ip' -)
USER_ADMIN_USER=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_USER_ADMIN_USER' -|base64 -d)
USER_ADMIN_PASS=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_USER_ADMIN_PASSWORD' -|base64 -d)
mongosh mongodb://$MONGO_EP -u $USER_ADMIN_USER -p $USER_ADMIN_PASS

# vytvorime roli
db.getSiblingDB("admin").createRole({
    role: "explainRole",
    privileges: [{
        resource: {
            db: "",
            collection: "system.profile"
            },
        actions: [
            "listIndexes",
            "listCollections",
            "dbStats",
            "dbHash",
            "collStats",
            "find"
            ]
        }],
    roles:[]
})

# tenhle uzivatel tam jiz je tak ho jen updatujeme s novyma rolema
db.getSiblingDB("admin").updateUser("clusterMonitor",
{
   roles: [
      { role: "explainRole", db: "admin" },
      { role: "clusterMonitor", db: "admin" },
      { role: "read", db: "local" }
   ]
})
```
Dale je nutne servisu pripojit k PMM grafane serveru  a to udelame prez
management
a nebo
```sh
MongoDB Instances OverviewPMM ---> InventoryPMM ---> Add Instance ---> Add a Remote MongoDB Instance

USER=$(kb get secrets my-cluster-name-secrets -o yaml|yq e '.data.MONGODB_CLUSTER_MONITOR_USER' -|base64 -d) 
PASSWORD=$(kb get secrets my-cluster-name-secrets -o yaml|yq e '.data.MONGODB_CLUSTER_MONITOR_PASSWORD' -|base64 -d) 
HOST=$(kb get svc mongo1-mongos -o yaml|yq '.metadata.name')
PORT=$(kb get svc mongo1-mongos -o yaml|yq '.spec.ports[0].port')
kb get svc mongo1-mongos -o yaml|yq --unwrapScalar=false '.metadata.name + ":" + .spec.ports[0].port'

#nebo prez pmm-admin tool lokalne 
kb exec monitoring-0 -- bash -c "pmm-admin add mongodb \
--username=$USER --password=$PASSWORD \
--service-name=mymongosvc --host=$HOST --port=$PORT"
```
