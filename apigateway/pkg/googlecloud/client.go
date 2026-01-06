package googlecloud

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/datastore"
)

// Client wraps the Google Cloud Datastore client to provide domain-specific operations.
type Client struct {
	ds *datastore.Client
}

// NewClient creates a new Google Cloud Datastore client.
// It checks for DATASTORE_EMULATOR_HOST to verify if running against an emulator.
func NewClient(ctx context.Context, projectID string) (*Client, error) {
	// Support Emulator: The official client detects DATASTORE_EMULATOR_HOST automatically.
	// We log it here for visibility during development.
	if emulatorHost := os.Getenv("DATASTORE_EMULATOR_HOST"); emulatorHost != "" {
		fmt.Printf("Initializing Datastore Client against Emulator at %s\n", emulatorHost)
	}

	ds, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create datastore client: %w", err)
	}

	return &Client{ds: ds}, nil
}

// Close closes the underlying datastore client.
func (c *Client) Close() error {
	return c.ds.Close()
}
