# Overview

## Testovací Scénáře   
+   Deploy AKS **[DONE]**
+   Vytvoření nové instance PerconaMongo replica-setu pomocí K8s Percona operátoru
    -   Jak se v konfigurci operátoru pracuje s definicí velikosti PVC (lze tam konfigurovat i PV storageClass, kdyby byly pod K8s různé storage tiery?)
    **[DONE]**
    -   Lze přes operátor nakonfigurovat SSL certifikáty pro zabezpečenou komunikaci mezi nody replsetu? Lze využít K8s CertManger? **[DONE]**
    -   Lze přes operátor konfigurovat výkonové limity (PoDu)? **[DONE]**
    -   Obecně, co za konfigurace lze řešit pomocí GitOPs /Operátoru. **[DONE]**
+  Mongo protokol není HTTP, tedy připojení zvenčí clusteru nejde udělat přes Ingress, ale přes Service typu NodePort nebo LB (ale loadbalancing je v Mongo klientovi). Zmapovat varianty publikace Mongo replicasetu z K8s Clusteru. **[DONE]**
+  Zdokumentování podpory konfigurace Shardované DB přes operátor - podporuje to Percona Operátor? **[DONE]**
+  Failover testy - manuální drain nodu nebo brute force sestřelíme K8s node -  jak se s tím vyrovná zápis dat do replica setu (výpadek?) 
+  Budou fungovat zálohovací/recovery postupy v K8s prostředí stejně jako na VMs - viz dokument níže? **[DONE - nebudou]**
+ FullBackup + OPLog Backup na S3 storage **[DONE]**
+ PIT Recovery + Recovery do jiné konfigurace MongoDB 
+ Retence záloh a automatické mazání z S3
+ PMM (Percona Monitoring and Management) - integrace 
 
**BackLog**  
+  Several nines - Cluster manager - zjednodušení automatizace provisioningu, (re)konfigurace. Monitoring. **[vynecháme, je to samostatný konkurenční produkt]**

## LINKS
[Percona roadMap](https://github.com/percona/roadmap/projects/1)  
[TLS with CertManger](https://www.percona.com/doc/kubernetes-operator-for-psmongodb/TLS.html)  
[MongoDB manual](https://docs.mongodb.com/v4.4)  


## TOOLING
- jq
- yq
- kustomize
- mongosh
