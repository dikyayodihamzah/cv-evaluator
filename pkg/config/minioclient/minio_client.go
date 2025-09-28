package minioclient

import (
	"context"
	"log"

	"github.com/dikyayodihamzah/cv-evaluator/pkg/env"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	endpoint        = env.GetString("MINIO_ENDPOINT")
	accessKeyID     = env.GetString("MINIO_ACCESS_KEY")
	bucketName      = env.GetString("MINIO_BUCKET", "cv-evaluator")
	secretAccessKey = env.GetString("MINIO_SECRET_KEY")
	ssl             = env.GetBool("MINIO_USE_SSL", false)
	region          = env.GetString("MINIO_BUCKET_LOCATION")
)

// New creates a new MinIO client and ensures bucket exists
func New() *minio.Client {
	ctx := context.Background()

	// Validate required environment variables
	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" {
		log.Fatal("MinIO configuration incomplete: missing MINIO_ENDPOINT, MINIO_ACCESS_KEY, or MINIO_SECRET_KEY")
	}

	// Initialize minio client object
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: ssl,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Create bucket if it doesn't exist
	if err := minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region}); err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Println("Bucket already exists",
				"bucketname", bucketName,
			)
		} else {
			log.Fatal(err)
		}
	} else {
		log.Println("Successfully created bucket in Minio",
			"bucketname", bucketName,
		)
	}
	return minioClient
}
