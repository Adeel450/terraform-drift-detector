// Package aws implements the AWS cloud provider: it fetches live resource
// metadata via the AWS SDK for Go v2 and registers the S3 Terraform state
// backend. It supports a representative set of resource types; adding more is
// a matter of implementing another provider.ResourceMapper.
package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/adeel450/terraform-drift-detector/internal/model"
	"github.com/adeel450/terraform-drift-detector/internal/provider"
	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

func init() {
	provider.Register("aws", New)
}

// Provider is the AWS provider backed by EC2 and S3 clients.
type Provider struct {
	ec2 *ec2.Client
	s3  *s3.Client
}

// New constructs the AWS provider using the default credential chain
// (environment, shared config/profile, or instance role).
func New(ctx context.Context, opts provider.Options) (provider.Provider, error) {
	cfg, err := loadConfig(ctx, opts.Region)
	if err != nil {
		return nil, err
	}
	return &Provider{ec2: ec2.NewFromConfig(cfg), s3: s3.NewFromConfig(cfg)}, nil
}

func loadConfig(ctx context.Context, region string) (aws.Config, error) {
	var loadOpts []func(*awsconfig.LoadOptions) error
	if region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(region))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}
	return cfg, nil
}

// Name implements provider.Provider.
func (p *Provider) Name() string { return "aws" }

// Mappers implements provider.Provider.
func (p *Provider) Mappers() []provider.ResourceMapper {
	return []provider.ResourceMapper{
		&instanceMapper{p: p},
		&securityGroupMapper{p: p},
		&bucketMapper{p: p},
	}
}

// --- aws_instance ---

type instanceMapper struct{ p *Provider }

func (m *instanceMapper) TerraformType() string { return "aws_instance" }

func (m *instanceMapper) FromState(raw tfstate.Instance) (model.Resource, error) {
	inst := tfInstance{raw}
	return model.Resource{
		Provider: "aws",
		Type:     "aws_instance",
		ID:       inst.str("id"),
		Name:     inst.name(),
		Attributes: map[string]string{
			"instance_type": inst.str("instance_type"),
			"ami":           inst.str("ami"),
		},
		Tags: inst.tags(),
	}, nil
}

func (m *instanceMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	out, err := m.p.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{id}})
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			if inst.State != nil && inst.State.Name == ec2types.InstanceStateNameTerminated {
				continue // a terminated instance is effectively deleted
			}
			return model.Resource{
				Provider: "aws",
				Type:     "aws_instance",
				ID:       id,
				Attributes: map[string]string{
					"instance_type": string(inst.InstanceType),
					"ami":           aws.ToString(inst.ImageId),
				},
				Tags: ec2Tags(inst.Tags),
			}, true, nil
		}
	}
	return model.Resource{}, false, nil
}

// --- aws_security_group ---

type securityGroupMapper struct{ p *Provider }

func (m *securityGroupMapper) TerraformType() string { return "aws_security_group" }

func (m *securityGroupMapper) FromState(raw tfstate.Instance) (model.Resource, error) {
	inst := tfInstance{raw}
	return model.Resource{
		Provider: "aws",
		Type:     "aws_security_group",
		ID:       inst.str("id"),
		Name:     inst.name(),
		Attributes: map[string]string{
			"name":        inst.str("name"),
			"description": inst.str("description"),
		},
		Tags: inst.tags(),
	}, nil
}

func (m *securityGroupMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	out, err := m.p.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{id}})
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	for _, sg := range out.SecurityGroups {
		return model.Resource{
			Provider: "aws",
			Type:     "aws_security_group",
			ID:       id,
			Attributes: map[string]string{
				"name":        aws.ToString(sg.GroupName),
				"description": aws.ToString(sg.Description),
			},
			Tags: ec2Tags(sg.Tags),
		}, true, nil
	}
	return model.Resource{}, false, nil
}

// --- aws_s3_bucket ---

type bucketMapper struct{ p *Provider }

func (m *bucketMapper) TerraformType() string { return "aws_s3_bucket" }

func (m *bucketMapper) FromState(raw tfstate.Instance) (model.Resource, error) {
	inst := tfInstance{raw}
	return model.Resource{
		Provider: "aws",
		Type:     "aws_s3_bucket",
		ID:       inst.str("id"),
		Name:     inst.name(),
		Attributes: map[string]string{
			"bucket": inst.str("bucket"),
		},
		Tags: inst.tags(),
	}, nil
}

func (m *bucketMapper) FetchActual(ctx context.Context, id string) (model.Resource, bool, error) {
	_, err := m.p.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(id)})
	if err != nil {
		if isNotFound(err) {
			return model.Resource{}, false, nil
		}
		return model.Resource{}, false, err
	}
	tags := map[string]string{}
	tagOut, err := m.p.s3.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(id)})
	if err == nil {
		tags = s3Tags(tagOut.TagSet)
	} else if !isNotFound(err) && !isNoSuchTagSet(err) {
		return model.Resource{}, false, err
	}
	return model.Resource{
		Provider:   "aws",
		Type:       "aws_s3_bucket",
		ID:         id,
		Attributes: map[string]string{"bucket": id},
		Tags:       tags,
	}, true, nil
}

// --- helpers ---

func ec2Tags(tags []ec2types.Tag) map[string]string {
	out := map[string]string{}
	for _, t := range tags {
		out[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return out
}

func s3Tags(tags []s3types.Tag) map[string]string {
	out := map[string]string{}
	for _, t := range tags {
		out[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return out
}

// isNotFound reports whether err is an AWS "resource does not exist" error.
func isNotFound(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		switch {
		case strings.Contains(code, "NotFound"):
			return true
		case code == "NoSuchBucket", code == "NoSuchEntity", code == "404":
			return true
		}
	}
	return false
}

// isNoSuchTagSet reports whether err means a bucket simply has no tags.
func isNoSuchTagSet(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "NoSuchTagSet"
	}
	return false
}
