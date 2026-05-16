package filter

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
)

func TestLocationNodeFilterRegionAndIDC(t *testing.T) {
	nodes := testLocationNodes()
	filterFn := LocationNodeFilter(LocationOptions{
		RegionID: "1",
		IDCID:    "123",
	})
	filtered := filterFn(context.Background(), nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.1:9000" {
		t.Fatalf("expect node 127.0.0.1:9000, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterDefaultCurrentLocation(t *testing.T) {
	nodes := testLocationNodes()
	ctx := WithCurrentLocation(context.Background(), "1", "345")
	filterFn := LocationNodeFilter(LocationOptions{})
	filtered := filterFn(ctx, nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.2:9000" {
		t.Fatalf("expect node 127.0.0.2:9000, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterDefaultRegionAndIDC(t *testing.T) {
	nodes := testLocationNodes()
	filterFn := LocationNodeFilter(LocationOptions{
		DefaultRegionID: "2",
		DefaultIDCID:    "555",
	})
	filtered := filterFn(context.Background(), nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.3:9000" {
		t.Fatalf("expect node 127.0.0.3:9000, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterHashIDCID(t *testing.T) {
	nodes := testLocationNodes()
	filterFn := LocationNodeFilter(LocationOptions{
		RegionID:     "1",
		HashIDCIDKey: "user-10086",
	})
	filtered1 := filterFn(context.Background(), nodes)
	filtered2 := filterFn(context.Background(), nodes)

	if !reflect.DeepEqual(filtered1, filtered2) {
		t.Fatal("expect stable hash idc result for same key")
	}
	if len(filtered1) == 0 {
		t.Fatal("expect hash idc to choose at least one node")
	}

	idcID := filtered1[0].Metadata()[registry.MetadataIDCIDKey]
	for _, n := range filtered1 {
		if n.Metadata()[registry.MetadataIDCIDKey] != idcID {
			t.Fatalf("expect all selected nodes in same idc, got mixed idc %q and %q", idcID, n.Metadata()[registry.MetadataIDCIDKey])
		}
	}
}

func testLocationNodes() []selector.Node {
	return []selector.Node{
		selector.NewNode("grpc", "127.0.0.1:9000", &registry.ServiceInstance{
			Name: "svc.test",
			Metadata: map[string]string{
				registry.MetadataRegionIDKey: "1",
				registry.MetadataIDCIDKey:    "123",
			},
		}),
		selector.NewNode("grpc", "127.0.0.2:9000", &registry.ServiceInstance{
			Name: "svc.test",
			Metadata: map[string]string{
				registry.MetadataRegionIDKey: "1",
				registry.MetadataIDCIDKey:    "345",
			},
		}),
		selector.NewNode("grpc", "127.0.0.3:9000", &registry.ServiceInstance{
			Name: "svc.test",
			Metadata: map[string]string{
				registry.MetadataRegionIDKey: "2",
				registry.MetadataIDCIDKey:    "555",
			},
		}),
	}
}
