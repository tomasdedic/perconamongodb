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
for (var i = 1; i <= 100; ++i) {
    var randomName = (Math.random()+1).toString(36).substring(2);
    db.bbc.insert({name: randomName, creationDate: randomDate(), uid: i});
  }
db.bbc.find()

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

### FULL BACKUP  
Provedeme backup shardované databáze přičemž admin a config DB jsou původní,
systémové.  
"On demand" backup provedeme na minio S3 storage do **bucketu: testbackup**

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
# cas je nastaven na 5 minut pro ukladanni oplogu
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

### FULL RECOVERY + PITR do jiného MongoDB clusteru
Snížíme počet shardů a z 2 na 1 a provedeme recovery do stejného clusteru:


Vytvoříme novou instalace MongoDB clusteru a recovery provedeme do něj:

