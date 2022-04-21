# Instalace PMM server

## Vytvoření secret pro propojení agenta s percona serverem
Zkontrolovat zda jsou nastaveny User a Password

```sh
#check password
kb get secrets my-cluster-name-secrets -o yaml|yq e '.data.PMM_SERVER_USER' -|base64 -d &&echo
#PMM_SERVER_USER: admin

kb get secrets my-cluster-name-users -o yaml|yq e '.data.PMM_SERVER_PASSWORD' -|base64 -d &&echo
#PMM_SERVER_PASSWORD: admin123456
```

```sh
#zmena nebo priprava
kb patch secret/my-cluster-name-secrets -p '{"data":{"PMM_SERVER_PASSWORD": '\"$(echo -n admin123456 | base64)\"'}}'
#pokud cluster uz bezi je potreba patchnout internal secrets jelikoz si je
#vytvari pri instalaci a pody mongo je pak referencuji
kb patch secret/internal-mongo1-users -p '{"data":{"PMM_SERVER_PASSWORD": '\"$(echo -n admin123456 | base64)\"'}}'
#nebo prepripravene
kb apply -f tasks/deploy/secrets.yaml
```
## Install PMM SERVER

```sh
kb apply -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.11.0/deploy/bundle.yaml 
helm repo add percona https://percona-charts.storage.googleapis.com
helm repo update
helm install monitoring percona/pmm-server --set "platform=kubernetes" --version 2.26.1 --set "persistence.size=100Gi"
```
```sh
#pripojeni k serveru
MONGO_CONN=$(kb get svc monitoring-service -o yaml|yq e '.status.loadBalancer.ingress[0].ip' -)
echo "https://$MONGO_CONN"
# prihlasime se admin/admin a heslo zmenime
# nebo lze default heslo zmenit prez cli grafany
kb exec -it monitoring-0 -- bash -c 'grafana-cli --homepath /usr/share/grafana --configOverrides cfg:default.paths.data=/srv/grafana admin reset-admin-password admin123456'
```

## Instalace PMM client
[Instalace CLIENT](https://www.percona.com/doc/kubernetes-operator-for-psmongodb/monitoring.html#installing-pmm-server)  
```sh
#patch CR
kb apply -f tasks/pmm/cr.yaml
```
Dojde k injektnuti containeru s PMM clientem do jednotlivych podu mongodb

## Profiling
Pro exporter se vyuziva uzivatel **clusterMonitor** 
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
