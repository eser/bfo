package dynamodb_store

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

// EndpointResolver implements the dynamodb.EndpointResolverV2 interface to provide a custom endpoint.
var _ dynamodb.EndpointResolverV2 = (*EndpointResolver)(nil)

type EndpointResolver struct {
	endpointUrl string
}

func NewEndpointResolver(endpointUrl string) *EndpointResolver {
	return &EndpointResolver{endpointUrl: endpointUrl}
}

// ResolveEndpoint resolves the endpoint for DynamoDb.
func (r *EndpointResolver) ResolveEndpoint(ctx context.Context, params dynamodb.EndpointParameters) (smithyendpoints.Endpoint, error) {
	if r.endpointUrl == "" {
		// Fallback to default resolver if no custom endpoint is configured.
		// This path should ideally not be reached if EndpointResolver is only instantiated when endpointUrl is non-empty.
		return dynamodb.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
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
