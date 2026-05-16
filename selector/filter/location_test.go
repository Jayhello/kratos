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
		IDCStrategy:  IDCStrategyCHash,
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

func TestLocationNodeFilterRegionFallbackList(t *testing.T) {
	nodes := testLocationNodes()
	filterFn := LocationNodeFilter(LocationOptions{
		RegionID:         "9",
		DefaultRegionIDs: []string{"2", "1"},
		IDCStrategy:      IDCStrategyOneRegion,
	})
	filtered := filterFn(context.Background(), nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect fallback region to return 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.3:9000" {
		t.Fatalf("expect fallback hit region 2 node 127.0.0.3:9000, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterNormalFallbackToRegion(t *testing.T) {
	nodes := testLocationNodes()
	ctx := WithCurrentLocation(context.Background(), "1", "not-exist")
	filterFn := LocationNodeFilter(LocationOptions{IDCStrategy: IDCStrategyNormal})
	filtered := filterFn(ctx, nodes)
	if !reflect.DeepEqual(2, len(filtered)) {
		t.Fatalf("expect normal strategy fallback to region nodes, got %d", len(filtered))
	}
}

func TestLocationNodeFilterSelfThenSeq(t *testing.T) {
	nodes := testLocationNodes()
	ctx := WithCurrentLocation(context.Background(), "1", "not-exist")
	filterFn := LocationNodeFilter(LocationOptions{
		IDCStrategy:    IDCStrategySelfThenSeq,
		SequenceIDCIDs: []string{"777", "345", "123"},
	})
	filtered := filterFn(ctx, nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect self-then-seq strategy to select 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.2:9000" {
		t.Fatalf("expect seq fallback pick idc 345 node, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterDesignatedRegionAndIDC(t *testing.T) {
	nodes := testLocationNodes()
	ctx := WithDesignatedIDCID(WithDesignatedRegion(context.Background(), "1"), "123")
	filterFn := LocationNodeFilter(LocationOptions{
		IDCStrategy: IDCStrategySelf,
	})
	filtered := filterFn(ctx, nodes)
	if !reflect.DeepEqual(1, len(filtered)) {
		t.Fatalf("expect designated region/idc select 1 node, got %d", len(filtered))
	}
	if filtered[0].Address() != "127.0.0.1:9000" {
		t.Fatalf("expect designated region/idc pick 127.0.0.1:9000, got %s", filtered[0].Address())
	}
}

func TestLocationNodeFilterSelfThenCHashWithGroupRequestCode(t *testing.T) {
	nodes := testLocationNodes()
	ctx := WithGroupRequestCode(WithCurrentLocation(context.Background(), "1", "not-exist"), "order-42")
	filterFn := LocationNodeFilter(LocationOptions{
		IDCStrategy: IDCStrategySelfThenCHash,
	})
	filtered1 := filterFn(ctx, nodes)
	filtered2 := filterFn(ctx, nodes)
	if len(filtered1) == 0 {
		t.Fatal("expect self-then-chash fallback to return nodes")
	}
	if !reflect.DeepEqual(filtered1, filtered2) {
		t.Fatal("expect stable hash result when group request code is unchanged")
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
