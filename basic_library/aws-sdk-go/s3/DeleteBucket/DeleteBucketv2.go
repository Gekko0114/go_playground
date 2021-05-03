package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3DeleteBucketAPI interface {
	DeleteBucket(
		ctx context.Context,
		params *s3.DeleteBucketInput,
		optFns ...func(*s3.Options)) (*s3.DeleteBucketOutput, error)
}

func RemoveBucket(c context.Context, api S3DeleteBucketAPI, input *s3.DeleteBucketInput) (*s3.DeleteBucketOutput, error) {
	return api.DeleteBucket(c, input)
}

func main() {
	bucket := flag.String("b", "", "The name of the bucket")
	flag.Parse()

	if *bucket == "" {
		fmt.Println("You must supply a bucket name (-b BUCKET)")
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	client := s3.NewFromConfig(cfg)
	input := &s3.DeleteBucketInput{
		Bucket: bucket,
	}
	_, err = RemoveBucket(context.TODO(), client, input)
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	_, err = RemoveBucket(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Could not delete bucket " + *bucket)
	}
}
