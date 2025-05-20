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
type EndpointResolver struct {
	endpointURL string
}

func NewEndpointResolver(endpointURL string) *EndpointResolver {
	return &EndpointResolver{endpointURL: endpointURL}
}

// ResolveEndpoint resolves the endpoint for DynamoDb.
func (r *EndpointResolver) ResolveEndpoint(ctx context.Context, params dynamodb.EndpointParameters) (smithyendpoints.Endpoint, error) {
	if r.endpointURL == "" {
		// Fallback to default resolver if no custom endpoint is configured.
		// This path should ideally not be reached if EndpointResolver is only instantiated when endpointURL is non-empty.
		return dynamodb.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
	}

	parsedURL, err := url.Parse(r.endpointURL)
	if err != nil {
		// Wrap the error for clarity, or return a more specific error if needed.
		return smithyendpoints.Endpoint{}, &aws.EndpointNotFoundError{Err: fmt.Errorf("failed to parse custom endpoint URL '%s': %w", r.endpointURL, err)}
	}

	return smithyendpoints.Endpoint{
		URI: *parsedURL,
	}, nil
}

// Ensure EndpointResolver implements the interface.
var _ dynamodb.EndpointResolverV2 = (*EndpointResolver)(nil)
