package config

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var Minio *minio.Client

func InitMinioConnection() {
	endpoint, endpointExists := os.LookupEnv("MIO_HOST")
	accessKey, accessKeyExists := os.LookupEnv("MIO_ACCESS_ID")
	secretAccessKey, secretExists := os.LookupEnv("MIO_SECRET")
	sslStatus, sslExists := os.LookupEnv("MIO_SSL")
	bucket, bucketExists := os.LookupEnv("MIO_BUCKET")

	if !endpointExists || !accessKeyExists || !secretExists || !sslExists || !bucketExists ||
		len(endpoint) == 0 || len(accessKey) == 0 || len(secretAccessKey) == 0 || len(sslStatus) == 0 || len(bucket) == 0 {
		log.Fatalln("[MINIO] required env variable is missing.")
	}

	var convErr error
	var useSSL bool
	useSSL, convErr = strconv.ParseBool(sslStatus)
	if convErr != nil {
		log.Panicln(convErr)
		log.Fatalln("[MINIO] SSL env probs.")
	}

	var conErr error
	Minio, conErr = minio.New(endpoint,
		&minio.Options{
			Creds:  credentials.NewStaticV4(accessKey, secretAccessKey, ""),
			Secure: useSSL,
		})
	if conErr != nil {
		log.Println(conErr)
		log.Fatalln("[MINIO] Connection probs.")
	}

	log.Println("[MINIO] Connected to ", Minio.EndpointURL().Host)

	ctx := context.Background()

	err := Minio.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := Minio.BucketExists(ctx, bucket)
		if errBucketExists == nil && exists {
			log.Printf("[MINIO] Bucket %s already exists\n", bucket)
		} else {
			log.Fatalln("[MINIO] Couldn't create bucket ", err)
		}
	} else {
		log.Printf("[MINIO] Successfully created %s\n", bucket)
	}
}
