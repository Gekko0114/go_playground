package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type EC2CreateImageAPI interface {
	CreateImage(
		ctx context.Context,
		params *ec2.CreateImageInput,
		optFns ...func(*ec2.Options)) (*ec2.CreateImageOutput, error)
}

func MakeImage(c context.Context, api EC2CreateImageAPI, input *ec2.CreateImageInput) (*ec2.CreateImageOutput, error) {
	return api.CreateImage(c, input)
}

func main() {
	description := flag.String("d", "", "The description of the image")
	instanceID := flag.String("i", "", "The ID of the instance")
	name := flag.String("n", "", "The name of the image")
	flag.Parse()

	if *description == "" || *instanceID == "" || *name == "" {
		fmt.Println("You must supply an image description, instance ID, and image name")
		fmt.Println("(-d IMAGE-DESCRIPTION -i INSTANCE-ID -n IMAGE-NAME")
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	client := ec2.NewFromConfig(cfg)

	input := &ec2.CreateImageInput{
		Description: description,
		InstanceId:  instanceID,
		Name:        name,
		BlockDeviceMappings: []types.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				NoDevice:   aws.String(""),
			},
			{
				DeviceName: aws.String("/dev/sdb"),
				NoDevice:   aws.String(""),
			},
			{
				DeviceName: aws.String("/dev/sdc"),
				NoDevice:   aws.String(""),
			},
		},
	}

	resp, err := MakeImage(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Got an error createing image:")
		fmt.Println(err)
		return
	}
	fmt.Println("ID: ", resp.ImageId)
}
