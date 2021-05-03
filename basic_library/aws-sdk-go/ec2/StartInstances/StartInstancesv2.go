package main

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/smithy-go"
)

type EC2StartInstancesAPI interface {
	StartInstances(
		ctx context.Context,
		params *ec2.StartInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.StartInstancesOutput, error)
}

func StartInstance(c context.Context, api EC2StartInstancesAPI, input *ec2.StartInstancesInput) (*ec2.StartInstancesOutput, error) {
	resp, err := api.StartInstances(c, input)

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && apiErr.ErrorCode() == "DryRunOperation" {
		fmt.Println("User has permission to start an instance.")
		input.DryRun = false
		return api.StartInstances(c, input)
	}
	return resp, err
}

func main() {
	instanceID := flag.String("i", "", "The ID of the instance to start")
	flag.Parse()

	if *instanceID == "" {
		fmt.Println("You must supply an instance ID (-i INSTANCE-ID")
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	client := ec2.NewFromConfig(cfg)

	input := &ec2.StartInstancesInput{
		InstanceIds: []string{
			*instanceID,
		},
		DryRun: true,
	}

	_, err = StartInstance(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Got an error starting the instance")
		fmt.Println(err)
		return
	}

	fmt.Println("Started instance with ID " + *instanceID)
}
