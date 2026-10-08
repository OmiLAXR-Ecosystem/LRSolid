package mongodb

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/SGoerzen/LRSolid/internal/store"
	"github.com/SGoerzen/LRSolid/internal/store/storetest"
)

func TestStatementsContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:7",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor:   wait.ForListeningPort("27017/tcp"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start mongo: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(context.Background()) })

	host, err := ctr.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := ctr.MappedPort(ctx, "27017/tcp")
	if err != nil {
		t.Fatal(err)
	}
	uri := fmt.Sprintf("mongodb://%s:%s/lrsolid", host, port.Port())
	bundle, err := NewStoreBundle(ctx, store.DataStoreConfig{ConnectionString: uri, DatabaseType: "mongodb"})
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	t.Cleanup(func() {
		if bundle.Lifecycle != nil {
			_ = bundle.Lifecycle.Close()
		}
	})
	storetest.RunStatementTests(t, bundle.Statements)
}
