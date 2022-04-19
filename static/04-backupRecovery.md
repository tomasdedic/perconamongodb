## Backup and Recovery 

### Random data generator
Uděláme **database seeding** a naplníme databázi **backuptest** náhodnými daty, jsme v
konfiguraci shardované databáze s 2 shardy a RS 3, nadefinovaná service
LoadBalancer pro přístup z venku

```sh
MONGO_EP=$(kb get svc mongo1-mongos -o yaml|yq e '.status.loadBalancer.ingress[0].ip' -)
USER_ADMIN_USER=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_USER_ADMIN_USER' -|base64 -d)
USER_ADMIN_PASS=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_USER_ADMIN_PASSWORD' -|base64 -d)
CLUSTER_ADMIN_USER=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_CLUSTER_ADMIN_USER' -|base64 -d)
CLUSTER_ADMIN_PASS=$(kb get secrets internal-mongo1-users -o yaml|yq e '.data.MONGODB_CLUSTER_ADMIN_PASSWORD' -|base64 -d)

# priradime roli readwrite pro uzivatele clusterAdmin
mongosh mongodb://$MONGO_EP -u $USER_ADMIN_USER -p $USER_ADMIN_PASS
use admin
db.grantRolesToUser(
    "clusterAdmin",
    [
      { role: "readWrite", db: "backuptest" }
    ]
)

mongosh mongodb://$MONGO_EP -u $CLUSTER_ADMIN_USER -p $CLUSTER_ADMIN_PASS
#chunk size je normalne 64MB zmensime ji na 1MB jelikoz chceme rychlejsi
#rozhazovani mezi shardy
use config
db.settings.insertOne( { _id:"chunksize", value: 1 } )

use backuptest
sh.enableSharding("backuptest")
sh.shardCollection("backuptest.bbc", { _id : 1  } )   
#sh.shardCollection("backuptest.bbc")   

#create random data
var day = 1000 * 60 * 60 * 24;
var randomDate = function () {
  return new Date(Date.now() - (Math.floor(Math.random() * day)));
}
for (var i = 1; i <= 200000; ++i) {
    var randomName = (Math.random()+1).toString(36).substring(2);
    db.bbc.insertOne({name: randomName, creationDate: randomDate(), uid: i});
  }
db.bbc.find()
db.bbc.countDocuments()

db.printShardingStatus()
db.bbc.getShardDistribution()
# Ted mame rozhozene chunky mezi obe shardovane instance(rs1 a rs0) DB backuptest
```

### MINIO S3 STORAGE
Nainstalujeme minio s3 storage, neresime replicy, roztahneme minio na AzureDisku
s velikosti 10GB. Chceme jen testovat S3 API.

```sh
#preprivavime si secrets pro pouziti MINIO
kubectl create secret generic minio --from-literal=rootUser=root --from-literal=rootPassword=otevriProsim

#pripraveny helm chart, values jsou jiz pripraveny
helm install minio tasks/minio -f tasks/minio/values.yaml
```

```sh
# test S3 api call proti minio a vytvoreni bucketu testbackup
kb run -i --rm aws-cli --image=perconalab/awscli --restart=Never -- \
bash -c 'AWS_ACCESS_KEY_ID=root \
AWS_SECRET_ACCESS_KEY=otevriProsim \
/usr/bin/aws \
--endpoint-url http://minio:9000 \
s3 mb s3://testbackup'

```
```sh
wget https://dl.min.io/client/mc/release/linux-amd64/mc
chmod +x mc
mv mc mcli #bije se mi to s midnighcommanderem
mcli --help
kb port-forward svc/minio 9000
mcli alias set local http://localhost:9000 $(kb get secret --namespace percona minio -o jsonpath="{.data.rootUser}" | base64 --decode) $(kb get secret --namespace percona minio -o jsonpath="{.data.rootPassword}" | base64 --decode)
# a vypiseme si hotove buckety
mcli ls local
> [2022-02-07 18:35:12 CET]     0B testbackup/
```
Ok bucket mame vytvoren a ted uděláme samotnou backup.

### FULL BACKUP ondemand
Provedeme backup shardované databáze přičemž admin a config DB jsou původní,
systémové.  
"On demand" backup provedeme na minio S3 storage do **bucketu: testbackup**  
Pro vazbu mezi S3 storage a backupem je potřeba mít vytvořený **finalizer** který po smazání
resourcu smaže i odpovídající soubory na S3 (objekt psmdb-backup)

```yaml
kind: PerconaServerMongoDBBackup
metadata:
  finalizers:
  - delete-backup
...
```

```sh
➤ kustomize build tasks/backup/|kb apply -f -
secret/minio-backup created
perconaservermongodb.psmdb.percona.com/mongo1 configured
perconaservermongodbbackup.psmdb.percona.com/backup1 created

➤ kb get perconaservermongodbbackups.psmdb.percona.com backup1
NAME      CLUSTER   STORAGE   DESTINATION            STATUS   COMPLETED   AGE
backup1   mongo1    minio     2022-02-07T20:29:25Z   ready    9m1s        9m40s
```

Backup na s3 minio storage
```sh
➤ mcli ls local/testbackup
Handling connection for 9000
[2022-02-07 21:29:28 CET]     5B STANDARD .pbm.init
[2022-02-07 21:29:43 CET] 3.3KiB STANDARD 2022-02-07T20:29:25Z.pbm.json
[2022-02-07 21:29:32 CET] 2.4MiB STANDARD 2022-02-07T20:29:25Z_cfg.dump.gz
[2022-02-07 21:29:35 CET] 7.8KiB STANDARD 2022-02-07T20:29:25Z_cfg.oplog.gz
[2022-02-07 21:29:31 CET] 579KiB STANDARD 2022-02-07T20:29:25Z_rs0.dump.gz
[2022-02-07 21:29:42 CET] 6.0KiB STANDARD 2022-02-07T20:29:25Z_rs0.oplog.gz
[2022-02-07 21:29:30 CET] 538KiB STANDARD 2022-02-07T20:29:25Z_rs1.dump.gz
[2022-02-07 21:29:42 CET] 1.7KiB STANDARD 2022-02-07T20:29:25Z_rs1.oplog.gz
[2022-02-07 21:30:08 CET] 3.3KiB STANDARD 2022-02-07T20:29:47Z.pbm.json
[2022-02-07 21:29:54 CET] 2.4MiB STANDARD 2022-02-07T20:29:47Z_cfg.dump.gz
[2022-02-07 21:29:57 CET] 8.8KiB STANDARD 2022-02-07T20:29:47Z_cfg.oplog.gz
[2022-02-07 21:29:53 CET] 579KiB STANDARD 2022-02-07T20:29:47Z_rs0.dump.gz
[2022-02-07 21:30:07 CET] 1.7KiB STANDARD 2022-02-07T20:29:47Z_rs0.oplog.gz
[2022-02-07 21:29:53 CET] 538KiB STANDARD 2022-02-07T20:29:47Z_rs1.dump.gz
[2022-02-07 21:30:07 CET] 1.7KiB STANDARD 2022-02-07T20:29:47Z_rs1.oplog.gz
```

### OPTLOG (PITR) BACKUP
> It is necessary to have at least one full backup to use point-in-time recovery. Percona Backup for MongoDB will not upload operations logs if there is no full backup. This is true for new clusters and also true for clusters which have been just recovered from backup
```sh
# cas je nastaven na 10 minut pro ukladanni oplogu
➤ kustomize build tasks/oplogs|kb apply -f -
perconaservermongodb.psmdb.percona.com/mongo1 configured
```

```sh
# Pridame par zaznamu skriptem uvedenym vyse do **bbc** collection
# pred
backuptest> db.bbc.countDocuments()
54398

# po
backuptest> db.bbc.countDocuments()
54498

#stav na S3 minio (posledni 3 radky)
➤ mcli ls local/testbackup/ --recursive 3
Handling connection for 9000
[2022-02-07 21:29:28 CET]     5B STANDARD .pbm.init
[2022-02-07 21:29:43 CET] 3.3KiB STANDARD 2022-02-07T20:29:25Z.pbm.json
[2022-02-07 21:29:32 CET] 2.4MiB STANDARD 2022-02-07T20:29:25Z_cfg.dump.gz
[2022-02-07 21:29:35 CET] 7.8KiB STANDARD 2022-02-07T20:29:25Z_cfg.oplog.gz
[2022-02-07 21:29:31 CET] 579KiB STANDARD 2022-02-07T20:29:25Z_rs0.dump.gz
[2022-02-07 21:29:42 CET] 6.0KiB STANDARD 2022-02-07T20:29:25Z_rs0.oplog.gz
[2022-02-07 21:29:30 CET] 538KiB STANDARD 2022-02-07T20:29:25Z_rs1.dump.gz
[2022-02-07 21:29:42 CET] 1.7KiB STANDARD 2022-02-07T20:29:25Z_rs1.oplog.gz
[2022-02-07 21:30:08 CET] 3.3KiB STANDARD 2022-02-07T20:29:47Z.pbm.json
[2022-02-07 21:29:54 CET] 2.4MiB STANDARD 2022-02-07T20:29:47Z_cfg.dump.gz
[2022-02-07 21:29:57 CET] 8.8KiB STANDARD 2022-02-07T20:29:47Z_cfg.oplog.gz
[2022-02-07 21:29:53 CET] 579KiB STANDARD 2022-02-07T20:29:47Z_rs0.dump.gz
[2022-02-07 21:30:07 CET] 1.7KiB STANDARD 2022-02-07T20:29:47Z_rs0.oplog.gz
[2022-02-07 21:29:53 CET] 538KiB STANDARD 2022-02-07T20:29:47Z_rs1.dump.gz
[2022-02-07 21:30:07 CET] 1.7KiB STANDARD 2022-02-07T20:29:47Z_rs1.oplog.gz
[2022-02-07 21:51:45 CET] 130KiB STANDARD pbmPitr/cfg/20220207/20220207202954-2.20220207205145-1.oplog.snappy
[2022-02-07 21:51:55 CET] 234KiB STANDARD pbmPitr/rs0/20220207/20220207202954-2.20220207205147-1.oplog.snappy
[2022-02-07 21:51:55 CET] 233KiB STANDARD pbmPitr/rs1/20220207/20220207202954-2.20220207205154-3.oplog.snappy
```

Bereme jako test ze se nam zalohuji **op logy**

### FULL RECOVERY + PITR do stejného MongoDB clusteru
> Problem pri fullrecovery je ztrata loadBalanceru jelikoz **mongos se restartuje** takze cluster si pak leasne novou IP adresu, potreba poresit zamkem v Azure a vynutit si IP adresu LB.  

**!Pozor je zde trik mcli udavá čas lokálně, ale AKS bezi v UTC tedy -1H, PITR date je tedy
potreba uvadet taky s přepočtem -1H!**
```sh
# Pridame jeste 100 zaznamu
[direct: mongos] backuptest> db.bbc.countDocuments()
54598

# obnovu udelame do 2022-02-07 21:52:00 tedy do mista 54498 zaznamu
kb apply -f tasks/restore/samecluster/restore.yaml

➤ kb get perconaservermongodbrestores.psmdb.percona.com
NAME       CLUSTER   STATUS   AGE
restore1   mongo1    ready    4m46s

[direct: mongos] backuptest> db.bbc.countDocuments()
54498
```
**Provedla se tedy recovery do časově nejvíce relevantního fullBackup a následně 
byly aplikováný OPlogy.**

### RETENCE cron záloh a automatické mazaní z S3 storage
```sh
#vycistime stare backup 
kb delete perconaservermongodbbackups.psmdb.percona.com --all
#smazeme OPlogy na s3 storage
mcli rm --force --recursive local/testbackup/pbmPitr/
```
vytvoříme job který poběží každé **3 minuty** a udělá **full backup s retencí 2 
uspěšných záloh** a zároveň vytvoříme fullbackup onDemand
```sh
#schedule: "*/3 * * * *"
kustomize build tasks/backupretention|kb apply -f -
```
```sh
➤ kb get perconaservermongodbbackups.psmdb.percona.com
NAME                               CLUSTER   STORAGE   DESTINATION            STATUS   COMPLETED   AGE
backup1                            mongo1    minio     2022-02-14T11:55:16Z   ready    97s         2m22s

➤ mcli ls local/testbackup/ 
[2022-02-14 12:55:40 CET] 3.3KiB STANDARD 2022-02-14T11:55:16Z.pbm.json
[2022-02-14 12:55:22 CET] 3.4MiB STANDARD 2022-02-14T11:55:16Z_cfg.dump.gz
[2022-02-14 12:55:25 CET] 9.5KiB STANDARD 2022-02-14T11:55:16Z_cfg.oplog.gz
[2022-02-14 12:55:22 CET] 594KiB STANDARD 2022-02-14T11:55:16Z_rs0.dump.gz
[2022-02-14 12:55:40 CET] 1.8KiB STANDARD 2022-02-14T11:55:16Z_rs0.oplog.gz
[2022-02-14 12:55:22 CET] 554KiB STANDARD 2022-02-14T11:55:16Z_rs1.dump.gz
[2022-02-14 12:55:37 CET] 1.7KiB STANDARD 2022-02-14T11:55:16Z_rs1.oplog.gz
[2022-02-14 12:55:34 CET]     0B pbmPitr/
---full backup backup1
# po 15 minutach budou stale rotovat dve posledni scheduled zalohy
➤ kb get perconaservermongodbbackups.psmdb.percona.com
NAME                               CLUSTER   STORAGE   DESTINATION            STATUS   COMPLETED   AGE
backup1                            mongo1    minio     2022-02-14T11:55:16Z   ready    18m         19m
cron-mongo1-20220214120900-dssxz   mongo1    minio     2022-02-14T12:09:22Z   ready    4m29s       5m8s
cron-mongo1-20220214121200-mjrvl   mongo1    minio     2022-02-14T12:12:22Z   ready    82s         2m8s

➤ mcli ls local/testbackup/
[2022-02-14 12:54:03 CET]     5B STANDARD .pbm.init
[2022-02-14 12:55:40 CET] 3.3KiB STANDARD 2022-02-14T11:55:16Z.pbm.json
[2022-02-14 12:55:22 CET] 3.4MiB STANDARD 2022-02-14T11:55:16Z_cfg.dump.gz
[2022-02-14 12:55:25 CET] 9.5KiB STANDARD 2022-02-14T11:55:16Z_cfg.oplog.gz
[2022-02-14 12:55:22 CET] 594KiB STANDARD 2022-02-14T11:55:16Z_rs0.dump.gz
[2022-02-14 12:55:40 CET] 1.8KiB STANDARD 2022-02-14T11:55:16Z_rs0.oplog.gz
[2022-02-14 12:55:22 CET] 554KiB STANDARD 2022-02-14T11:55:16Z_rs1.dump.gz
[2022-02-14 12:55:37 CET] 1.7KiB STANDARD 2022-02-14T11:55:16Z_rs1.oplog.gz
---full backup backup1
[2022-02-14 13:09:40 CET] 3.3KiB STANDARD 2022-02-14T12:09:22Z.pbm.json
[2022-02-14 13:09:28 CET] 3.4MiB STANDARD 2022-02-14T12:09:22Z_cfg.dump.gz
[2022-02-14 13:09:32 CET] 8.5KiB STANDARD 2022-02-14T12:09:22Z_cfg.oplog.gz
[2022-02-14 13:09:28 CET] 594KiB STANDARD 2022-02-14T12:09:22Z_rs0.dump.gz
[2022-02-14 13:09:40 CET] 1.7KiB STANDARD 2022-02-14T12:09:22Z_rs0.oplog.gz
[2022-02-14 13:09:28 CET] 564KiB STANDARD 2022-02-14T12:09:22Z_rs1.dump.gz
[2022-02-14 13:09:37 CET] 5.5KiB STANDARD 2022-02-14T12:09:22Z_rs1.oplog.gz
---cron fullbackup cron-mongo1-20220214120900-dssxz #1
[2022-02-14 13:12:47 CET] 3.3KiB STANDARD 2022-02-14T12:12:22Z.pbm.json
[2022-02-14 13:12:30 CET] 3.4MiB STANDARD 2022-02-14T12:12:22Z_cfg.dump.gz
[2022-02-14 13:12:33 CET] 8.7KiB STANDARD 2022-02-14T12:12:22Z_cfg.oplog.gz
[2022-02-14 13:12:27 CET] 594KiB STANDARD 2022-02-14T12:12:22Z_rs0.dump.gz
[2022-02-14 13:12:40 CET] 1.8KiB STANDARD 2022-02-14T12:12:22Z_rs0.oplog.gz
[2022-02-14 13:12:27 CET] 554KiB STANDARD 2022-02-14T12:12:22Z_rs1.dump.gz
[2022-02-14 13:12:47 CET] 1.9KiB STANDARD 2022-02-14T12:12:22Z_rs1.oplog.gz
---cron fullbackup cron-mongo1-20220214121200-mjrvl #2
[2022-02-14 13:13:05 CET]     0B pbmPitr/
---OPlog
```

### FULL RECOVERY + PITR do jiného MongoDB clusteru
**Recovery jde tímto schématem provádět pouze do clusterů vytvořených přez
Percona MongoDB operátor**

#### Snížíme počet shardů a z 2 na 1 a provedeme recovery do stejného clusteru:
Snížit jednoduše počet shardů čistě přez operátor nepůjde jelikož máme custom
databázi **backuptest**
```sh
kustomize build tasks/sharding/sharddown/|kb apply -f -
> request":"percona/mongo1","error":"check remove posibility for rs rs1: non system db found: backuptest","errorVerbose":"non system db found:
#presuneme tedy databazi backuptest na chunk rs0
db.adminCommand( { movePrimary: "backuptest", to: "rs0" } )
db.adminCommand( { removeShard: "rs1" } )
db.printShardingStatus()
# operator jede v consolidation loop takze po presunu customdb na rs0 zacne
# presouvat systemove databaze automaticky
```


#### Vytvoříme novou instalace MongoDB clusteru a recovery provedeme do něj:
MongoDB se bude jmenovat mongo2, bude mít pouze 1 shard(rs0), backup je udělaný s 2 shardy (rs0,rs1)
```sh
kb apply -f tasks/newMongoInstance/mongo2/cr.yaml
kb apply -f tasks/restore/differentcluster/restore.yaml


spec:
  backupSource:
    destination: s3://backuptest/2022-03-02T14:55:48Z
    s3:
      credentialsSecret: minio-backup
      endpointUrl: http://minio:9000
      region: westeurope
  clusterName: mongo2
status:
  error: |
    set resync backup list from the store: init storage: get S3 object header: InvalidParameter: 1 validation error(s) found.
    - minimum field size of 1, HeadObjectInput.Bucket.
  state: error

```

```sh
#dabaze: backuptes collection: bbc (20000 documents)
#test

MONGO_EP=$(kb get svc mongo2-mongos -o yaml|yq e '.status.loadBalancer.ingress[0].ip' -)
CLUSTER_ADMIN_USER=$(kb get secrets internal-mongo2-users -o yaml|yq e '.data.MONGODB_CLUSTER_ADMIN_USER' -|base64 -d)
CLUSTER_ADMIN_PASS=$(kb get secrets internal-mongo2-users -o yaml|yq e '.data.MONGODB_CLUSTER_ADMIN_PASSWORD' -|base64 -d)
```

