package sqs_queue

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

// EndpointResolver implements the sqs.EndpointResolverV2 interface to provide a custom endpoint.
var _ sqs.EndpointResolverV2 = (*EndpointResolver)(nil)

type EndpointResolver struct {
	endpointUrl string
}

func NewEndpointResolver(endpointUrl string) *EndpointResolver {
	return &EndpointResolver{endpointUrl: endpointUrl}
}

// ResolveEndpoint resolves the endpoint for SQS.
func (r *EndpointResolver) ResolveEndpoint(ctx context.Context, params sqs.EndpointParameters) (smithyendpoints.Endpoint, error) {
	if r.endpointUrl == "" {
		// Fallback to default resolver if no custom endpoint is configured.
		// This path should ideally not be reached if EndpointResolver is only instantiated when endpointUrl is non-empty.
		return sqs.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
	}

	parsedUrl, err := url.Parse(r.endpointUrl)
	if err != nil {
		// Wrap the error for clarity, or return a more specific error if needed.
		return smithyendpoints.Endpoint{}, &aws.EndpointNotFoundError{Err: fmt.Errorf("failed to parse custom endpoint Url '%s': %w", r.endpointUrl, err)}
	}

	return smithyendpoints.Endpoint{
		URI: *parsedUrl,
	}, nil
}
