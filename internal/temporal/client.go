package temporal

import (
	"go.temporal.io/sdk/client"
)

const TaskQueue = "fieldstone"

// NewClient creates a Temporal client connected to the given host.
// The caller is responsible for calling Close() on the returned client.
func NewClient(hostPort string) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  hostPort,
		Namespace: "default",
	})
}

// WorkflowID returns the canonical Temporal workflow ID for a resource.
// Convention: "<resource_type>-<resource_id>".
func WorkflowID(resourceType, resourceID string) string {
	return resourceType + "-" + resourceID
}
