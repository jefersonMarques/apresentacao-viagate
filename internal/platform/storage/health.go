package storage

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (s *S3) Check(ctx context.Context) error {
	if s==nil{return fmt.Errorf("S3 client is not configured")}
	if _,err:=s.client.HeadBucket(ctx,&s3.HeadBucketInput{Bucket:aws.String(s.bucket)});err!=nil{
		return fmt.Errorf("head S3 bucket: %w",err)
	}
	return nil
}
