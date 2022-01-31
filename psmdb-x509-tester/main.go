package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ca.crt is mounted from secret/cluster1-ssl
	caFilePath := "/etc/mongodb-ssl/ca.crt"

	// tls.pem consists of tls.key and tls.crt, they're mounted from secret/cluster1-psmdb-egegunes
	certKeyFilePath := "/tmp/tls.pem"

	endpoint := "cluster1-rs0.psmdb.svc.cluster.local"

	uri := fmt.Sprintf(
		"mongodb+srv://%s/?tlsCAFile=%s&tlsCertificateKeyFile=%s",
		endpoint,
		caFilePath,
		certKeyFilePath,
	)

	credential := options.Credential{
		AuthMechanism: "MONGODB-X509",
		AuthSource:    "$external",
	}
	opts := options.Client().SetAuth(credential).ApplyURI(uri)

	log.Println("Connecting to database")
	log.Println("URI:", opts.GetURI())
	log.Println("Username:", opts.Auth.Username)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to database", opts.GetURI())

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successful ping")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}
