package filter

import (
	"context"
	"hash/fnv"
	"sort"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
)

type locationKey struct{}

// Location carries current caller region/idc.
type Location struct {
	RegionID string
	IDCID    string
}

// WithCurrentLocation stores current region/idc in context.
func WithCurrentLocation(ctx context.Context, regionID, idcID string) context.Context {
	return context.WithValue(ctx, locationKey{}, Location{
		RegionID: regionID,
		IDCID:    idcID,
	})
}

// CurrentLocation loads current region/idc from context.
func CurrentLocation(ctx context.Context) (Location, bool) {
	loc, ok := ctx.Value(locationKey{}).(Location)
	return loc, ok
}

// LocationOptions controls location-based node filtering behavior.
type LocationOptions struct {
	// RegionID is the explicit target region id.
	RegionID string
	// DefaultRegionID is used when RegionID and current context region are empty.
	DefaultRegionID string
	// IDCID is the explicit target idc id.
	IDCID string
	// DefaultIDCID is used when IDCID and current context idc are empty.
	DefaultIDCID string
	// HashIDCIDKey enables idc hashing when not empty.
	HashIDCIDKey string
}

// LocationNodeFilter filters by region/idc, supporting explicit, default and hash-idc behavior.
func LocationNodeFilter(opts LocationOptions) selector.NodeFilter {
	return func(ctx context.Context, nodes []selector.Node) []selector.Node {
		if len(nodes) == 0 {
			return nodes
		}

		regionID, idcID := opts.RegionID, opts.IDCID
		if loc, ok := CurrentLocation(ctx); ok {
			if regionID == "" {
				regionID = loc.RegionID
			}
			if idcID == "" {
				idcID = loc.IDCID
			}
		}
		if regionID == "" {
			regionID = opts.DefaultRegionID
		}
		if idcID == "" {
			idcID = opts.DefaultIDCID
		}

		filtered := filterByMetadata(nodes, registry.MetadataRegionIDKey, regionID)
		if opts.HashIDCIDKey != "" {
			filtered = hashIDCID(filtered, opts.HashIDCIDKey)
		} else {
			filtered = filterByMetadata(filtered, registry.MetadataIDCIDKey, idcID)
		}
		return filtered
	}
}

func filterByMetadata(nodes []selector.Node, key, value string) []selector.Node {
	if value == "" {
		return nodes
	}
	filtered := make([]selector.Node, 0, len(nodes))
	for _, n := range nodes {
		if n.Metadata()[key] == value {
			filtered = append(filtered, n)
		}
	}
	return filtered
}

func hashIDCID(nodes []selector.Node, key string) []selector.Node {
	idcSet := make(map[string]struct{}, len(nodes))
	for _, n := range nodes {
		idcID := n.Metadata()[registry.MetadataIDCIDKey]
		if idcID == "" {
			continue
		}
		idcSet[idcID] = struct{}{}
	}
	if len(idcSet) == 0 {
		return nodes
	}

	idcs := make([]string, 0, len(idcSet))
	for idcID := range idcSet {
		idcs = append(idcs, idcID)
	}
	sort.Strings(idcs)

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(key))
	targetIDCID := idcs[int(hasher.Sum32())%len(idcs)]
	return filterByMetadata(nodes, registry.MetadataIDCIDKey, targetIDCID)
}
