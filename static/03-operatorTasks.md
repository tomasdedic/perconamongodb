## PerconaMongo operátor použití

### NGINX ingress controller
```yaml
helm install \
  nginx-ingress-external ingress-nginx/ingress-nginx \
  --create-namespace
  --namespace ingress-external \
  --set controller.ingressClassResource.name=nginx-external 
#test ingressClass
test 1 = \
$(kb get ingressclass|grep nginx-external|wc -l);\
echo $?
```
### CertManager
Percona nepodporuje cert-manager.io/v1 ale pouze stare API cert-manager.io/v1alpha2.
Je na to PR [cert-manager API version update](https://github.com/percona/percona-server-mongodb-operator/pull/863) 

#### Install
```yaml
kb create namespace cert-manager
helm repo add jetstack https://charts.jetstack.io
# Update your local Helm chart repository cache
helm repo update
helm search repo jetstack
# Install the cert-manager Helm chart
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v0.16.1 \
  --set installCRDs=true
```
#### Configure 
```yaml
cat <<EOF| kb apply -f -
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: pippo@gdu.org
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx-external
          podTemplate:
            spec:
              nodeSelector:
                "kubernetes.io/os": linux
EOF

test 1 = \
$(kb get clusterissuers|grep True|wc -l);\
echo $?
```

### Instalace Percona MongDB Op

```sh
git clone -b v1.11.0 https://github.com/percona/percona-server-mongodb-operator
kb create namespace percona
kb config set-context $(kb config current-context) --namespace=percona
kb apply -f https://raw.githubusercontent.com/percona/percona-server-mongodb-operator/v1.11.0/deploy/bundle.yaml 
#customresourcedefinition.apiextensions.k8s.io/perconaservermongodbs.psmdb.percona.com created
#customresourcedefinition.apiextensions.k8s.io/perconaservermongodbbackups.psmdb.percona.com created
#customresourcedefinition.apiextensions.k8s.io/perconaservermongodbrestores.psmdb.percona.com created
#role.rbac.authorization.k8s.io/percona-server-mongodb-operator created
#serviceaccount/percona-server-mongodb-operator created
#rolebinding.rbac.authorization.k8s.io/service-account-percona-server-mongodb-operator created
#deployment.apps/percona-server-mongodb-operator created
```

### Nová instance MongoDB
```sh
git clone (moje git repo) && cd $(basename $_ .git)
```

```yaml
kb apply -f tasks/newMongoInstance/cr.yaml
#test
db.version()
db.adminCommand( { listShards: 1 } )
```

Instalace provedena v konfiguraci:

```sh
service: LoadBalancer
sharding: on
numberOfshards: 1

#mongod daemon run
tr '\0' ' ' </proc/1/cmdline
mongod --bind_ip_all --auth --dbpath=/data/db --port=27017 --replSet=rs0 --storageEngine=wiredTiger --relaxPermChecks --clusterAuthMode=x509 --shardsvr --slowms=100 --profile=1 --rateLimit=100 --enableEncryption --encryptionKeyFile=/etc/mongodb-encryption/encryption-key --encryptionCipherMode=AES256-CBC --wiredTigerCacheSizeGB=0.25 --wiredTigerCollectionBlockCompressor=snappy --wiredTigerJournalCompressor=snappy --wiredTigerIndexPrefixCompression=true --setParameter ttlMonitorSleepSecs=60 --setParameter wiredTigerConcurrentReadTransactions=128 --setParameter wiredTigerConcurrentWriteTransactions=128 --tlsMode preferTLS --tlsCertificateKeyFile /tmp/tls.pem --tlsAllowInvalidCertificates --tlsClusterFile /tmp/tls-internal.pem --tlsCAFile /etc/mongodb-ssl/ca.crt --tlsClusterCAFile /etc/mongodb-ssl-internal/ca.crt 

#mongos router run
tr '\0' ' '</proc/1/cmdline
mongos --bind_ip_all --port=27017 --configdb cfg/mongo1-cfg-0.mongo1-cfg.percona.svc.cluster.local:27017,mongo1-cfg-1.mongo1-cfg.percona.svc.cluster.local:27017,mongo1-cfg-2.mongo1-cfg.percona.svc.cluster.local:27017 --relaxPermChecks --clusterAuthMode=x509 --tlsMode preferTLS --tlsCertificateKeyFile /tmp/tls.pem --tlsAllowInvalidCertificates --tlsClusterFile /tmp/tls-internal.pem --tlsCAFile /etc/mongodb-ssl/ca.crt --tlsClusterCAFile /etc/mongodb-ssl-internal/ca.crt 
```

```sh
kb get pods -o=custom-columns='NAME:.metadata.name,STATUS:.status.conditions[1].type'
NAME                                               STATUS
mongo1-cfg-0                                       Ready
mongo1-cfg-1                                       Ready
mongo1-cfg-2                                       Ready
mongo1-mongos-6d597c7f6-kk86h                      Ready
mongo1-mongos-6d597c7f6-lf5rn                      Ready
mongo1-mongos-6d597c7f6-mqx5k                      Ready
mongo1-rs0-0                                       Ready
mongo1-rs0-1                                       Ready
mongo1-rs0-2                                       Ready
percona-server-mongodb-operator-5dd88ff7f7-mzgmf   Ready
``` 
Overall limits and requests:


### Change POD limits/requests

```sh
kustomize build tasks/podResources |kb apply -f -
```
Pojde k postupnému STS rolloutu jednotlivých podů RS pro shard0. Databáze nebude
z clientského pohledu nijak ovlivněna.

### ShardUP/shardDown
Lze pouzit k zvětšení diskové kapacity přidáním většího shardu a následně
odebráním původního
```sh
#shardup: pridani jednoho shardu s vetsim diskem
kustomize build tasks/sharding/shardup |kb apply -f -
# automaticky dojde k rebalancovani chunku v shardech
# to check sharding status
db.adminCommand( { listShards: 1 } )
db.printShardingStatus()

```
```sh
kb get pods -o=custom-columns='NAME:.metadata.name,STATUS:.status.conditions[1].type'

NAME                                               STATUS
mongo1-cfg-0                                       Ready
mongo1-cfg-1                                       Ready
mongo1-cfg-2                                       Ready
mongo1-mongos-6d597c7f6-kk86h                      Ready
mongo1-mongos-6d597c7f6-lf5rn                      Ready
mongo1-mongos-6d597c7f6-mqx5k                      Ready
mongo1-rs0-0                                       Ready
mongo1-rs0-1                                       Ready
mongo1-rs0-2                                       Ready
mongo1-rs1-0                                       Ready
mongo1-rs1-1                                       Ready
mongo1-rs1-2                                       Ready
percona-server-mongodb-operator-5dd88ff7f7-mzgmf   Ready
```
```sh
#sharddown: odebrani puvodniho shardu
kustomize build tasks/sharding/shardown/ |kb apply -f -
# automaticky dojde k rebalancovani chunku v shardech
# to check sharding status
db.adminCommand( { listShards: 1 } )
db.printShardingStatus()
```
Po přesunu všech chunků z odstraňovaného shardu je STS smazán.

### Integrace Cert-manageru
Pokud je cert-manager nainstalován, vytvoří se při instalaci **Issuer**
```yaml
apiVersion: cert-manager.io/v1beta1
kind: Issuer
metadata:
  name: mongo1-psmdb-ca
  namespace: percona
spec:
  selfSigned: {}
```

  
Přez Issuera jsou následně vytvořeny dva identické certfifikáty s příslušnými secrety
```sh
kb get certs
NAME                  READY   SECRET                
mongo1-ssl            True    mongo1-ssl            
mongo1-ssl-internal   True    mongo1-ssl-internal   
```
Jedná se o certifikáty pro CA a jejich secret values
```sh
kb get secrets mongo1-ssl -o yaml|yq e '.data."tls.crt"' -|base64 -d|openssl x509 -subject -issuer -ext subjectAltName,basicConstraints -noout

subject=O = PSMDB, CN = mongo1
issuer=O = PSMDB, CN = mongo1
X509v3 Basic Constraints: critical
    CA:TRUE
X509v3 Subject Alternative Name:
    DNS:localhost, DNS:mongo1-rs0, DNS:mongo1-rs0.percona, DNS:mongo1-rs0.percona.svc.cluster.local, DNS:*.mongo1-rs0, DNS:*.mongo1-rs0.percona, DNS:*.mongo1-rs0.percona.svc.cluster.local, DNS:mongo1-mongos, DNS:mongo1-mongos.percona, DNS:mongo1-mongos.percona.svc.cluster.local, DNS:*.mongo1-mongos, DNS:*.mongo1-mongos.percona, DNS:*.mongo1-mongos.percona.svc.cluster.local, DNS:mongo1-cfg, DNS:mongo1-cfg.percona, DNS:mongo1-cfg.percona.svc.cluster.local, DNS:*.mongo1-cfg, DNS:*.mongo1-cfg.percona, DNS:*.mongo1-cfg.percona.svc.cluster.local
```

Secrety pro certifikáty jsou pak následně mountovány do podu.

**TODO**:  
Issuer je tedy selfSigned CA, v předinstalačních krocím lze nahradit libovolným
jiným issuerem. Problém je že je potřeba vytvořit certifikáty pro interní infra,
ACME s jeho Challenges je tedy nepoužitelný.  
Zatím nevím jak se k tomu postavit .

### bez Shardingu
Provoz se zapnutným shardigem může mít vyšší požadavky na resourci clusteru. V
některých scénářích může být vhodné mít sharding vypnutý.  

### Jak se v konfigurci operátoru pracuje s definicí velikosti PVC (lze tam konfigurovat i PV storageClass, kdyby byly pod K8s různé storage tiery?)

### Obecně, co za konfigurace lze řešit pomocí GitOPs /Operátoru.

### Backup and Recovery obecně

### PIT recovery
