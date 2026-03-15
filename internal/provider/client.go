package provider

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type authedTransport struct {
	token     string
	userAgent string
	wrapped   http.RoundTripper
}

// graphqlOperationEnvelope is used solely to extract the operation name from
// the request body. We intentionally do NOT log the variables or
// query fields — they may contain secrets.
type graphqlOperationEnvelope struct {
	OperationName string `json:"operationName"`
}

func (t *authedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Project-Access-Token", t.token)

	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}

	// Extract the GraphQL operation name for debug logging.
	// Read the body, parse just the operationName, then re-set the body
	// so the actual request still works.
	var operationName string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			// Restore the body for the actual request.
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			var gqlReq graphqlOperationEnvelope
			if json.Unmarshal(bodyBytes, &gqlReq) == nil && gqlReq.OperationName != "" {
				operationName = gqlReq.OperationName
			}
		}
	}

	ctx := req.Context()

	if operationName != "" {
		tflog.Debug(ctx, "GraphQL request", map[string]interface{}{
			"operation": operationName,
		})
	}

	resp, err := t.wrapped.RoundTrip(req)

	if err != nil {
		tflog.Warn(ctx, "GraphQL request failed", map[string]interface{}{
			"operation": operationName,
			"error":     err.Error(),
		})
	} else if operationName != "" {
		tflog.Trace(ctx, "GraphQL response", map[string]interface{}{
			"operation":   operationName,
			"status_code": resp.StatusCode,
		})
	}

	return resp, err
}
