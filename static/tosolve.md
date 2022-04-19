## 1 .correct way how to scale down
I have a seeded database across shards: rs0,rs1. I would like to shard down.
```sh
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
for (var i = 1; i <= 20000; ++i) {
    var randomName = (Math.random()+1).toString(36).substring(2);
    db.bbc.insertOne({name: randomName, creationDate: randomDate(), uid: i});
  }
db.bbc.countDocuments()

db.printShardingStatus()
#check chunks distribution
db.bbc.getShardDistribution()
```
```sh
db.adminCommand( { movePrimary: "backuptest", to: "rs0" } )
db.adminCommand( { removeShard: "rs1" } )
db.printShardingStatus()
#sharddown by CR
kustomize build tasks/sharding/sharddown/|kb apply -f -
> request":"percona/mongo1","error":"check remove posibility for rs rs1: non system db found: backuptest","errorVerbose":"non system db found:
```
## 2. OP logs retention
How to auto delete OP logs in S3 storage

## 3. Recovery into different database with different numbers of shards
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

## 4. PMM integration
Is it necessary to make a part of creating custom PMM mongoDB  instance MongoDB **Instances OverviewPMM ---> InventoryPMM ---> Add Instance ---> Add a Remote MongoDB Instance**.  
In fact question would be " Is it necessary to configure PMM or all important
stuff is already defined in CR"

