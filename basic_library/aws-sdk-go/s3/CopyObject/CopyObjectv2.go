package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3CopyObjectAPI interface {
	CopyObject(
		ctx context.Context,
		params *s3.CopyObjectInput,
		optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

func CopyItem(c context.Context, api S3CopyObjectAPI, input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	return api.CopyObject(c, input)
}

func main() {
	sourceBucket := flag.String("s", "", "The source bucket containing the object to copy")
	destinationBucket := flag.String("d", "", "The destination bucket to which the object is copied")
	objectName := flag.String("o", "", "The object to copy")
	flag.Parse()

	if *sourceBucket == "" || *destinationBucket == "" || *objectName == "" {
		fmt.Println("You must supply the bucket to copy from (-s BUCKET), to (-td BUCKET), and object to copy (-o OBJECT")
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	client := s3.NewFromConfig(cfg)

	input := &s3.CopyObjectInput{
		Bucket:     aws.String(url.PathEscape(*sourceBucket)),
		CopySource: destinationBucket,
		Key:        objectName,
	}

	_, err = CopyItem(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Got an error copying item:")
		fmt.Println(err)
		return
	}

	fmt.Println("Copied " + *objectName + " from " + *sourceBucket + " to " + *destinationBucket)
}
