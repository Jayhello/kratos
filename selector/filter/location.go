package filter

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
)

type locationKey struct{}
type designatedRegionKey struct{}
type designatedIDCIDKey struct{}
type groupRequestCodeKey struct{}

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
	// DefaultRegionIDs is ordered region fallback list when target region has no available node.
	DefaultRegionIDs []string
	// IDCID is the explicit target idc id.
	IDCID string
	// DefaultIDCID is used when IDCID and current context idc are empty.
	DefaultIDCID string
	// HashIDCIDKey enables idc hashing when not empty.
	HashIDCIDKey string
	// GroupRequestCode is alias of HashIDCIDKey.
	GroupRequestCode string
	// IDCStrategy controls idc selection behavior.
	IDCStrategy IDCStrategy
	// SequenceIDCIDs is the ordered idc fallback list used by IDCStrategySelfThenSeq.
	SequenceIDCIDs []string
}

// IDCStrategy is idc routing strategy.
type IDCStrategy uint8

const (
	// IDCStrategyNormal prefers self idc, fallback to all nodes in the region.
	IDCStrategyNormal IDCStrategy = iota
	// IDCStrategyOneRegion always uses all nodes in the region.
	IDCStrategyOneRegion
	// IDCStrategySelf only uses self idc.
	IDCStrategySelf
	// IDCStrategyCHash picks one idc by hash key.
	IDCStrategyCHash
	// IDCStrategySelfThenCHash prefers self idc, fallback to hash idc.
	IDCStrategySelfThenCHash
	// IDCStrategySelfThenSeq prefers self idc, fallback by configured idc sequence.
	IDCStrategySelfThenSeq
)

// WithDesignatedRegion stores designated target region in context.
func WithDesignatedRegion(ctx context.Context, regionID string) context.Context {
	return context.WithValue(ctx, designatedRegionKey{}, regionID)
}

// DesignatedRegion loads designated target region from context.
func DesignatedRegion(ctx context.Context) (string, bool) {
	regionID, ok := ctx.Value(designatedRegionKey{}).(string)
	return regionID, ok
}

// WithDesignatedIDCID stores designated target idc in context.
func WithDesignatedIDCID(ctx context.Context, idcID string) context.Context {
	return context.WithValue(ctx, designatedIDCIDKey{}, idcID)
}

// DesignatedIDCID loads designated target idc from context.
func DesignatedIDCID(ctx context.Context) (string, bool) {
	idcID, ok := ctx.Value(designatedIDCIDKey{}).(string)
	return idcID, ok
}

// WithGroupRequestCode stores idc-hash key in context.
func WithGroupRequestCode(ctx context.Context, requestCode string) context.Context {
	return context.WithValue(ctx, groupRequestCodeKey{}, requestCode)
}

// GroupRequestCode loads idc-hash key from context.
func GroupRequestCode(ctx context.Context) (string, bool) {
	requestCode, ok := ctx.Value(groupRequestCodeKey{}).(string)
	return requestCode, ok
}

// LocationNodeFilter filters by region/idc, supporting explicit, fallback and idc strategy behavior.
func LocationNodeFilter(opts LocationOptions) selector.NodeFilter {
	return func(ctx context.Context, nodes []selector.Node) []selector.Node {
		if len(nodes) == 0 {
			return nodes
		}

		regionID, idcID := opts.RegionID, opts.IDCID
		if regionID == "" {
			if designatedRegionID, ok := DesignatedRegion(ctx); ok {
				regionID = designatedRegionID
			}
		}
		if idcID == "" {
			if designatedIDCID, ok := DesignatedIDCID(ctx); ok {
				idcID = designatedIDCID
			}
		}
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
		hashKey := opts.HashIDCIDKey
		if hashKey == "" {
			hashKey = opts.GroupRequestCode
		}
		if hashKey == "" {
			if groupRequestCode, ok := GroupRequestCode(ctx); ok {
				hashKey = groupRequestCode
			}
		}

		regionFallbacks := append([]string(nil), opts.DefaultRegionIDs...)
		if opts.DefaultRegionID != "" {
			regionFallbacks = append(regionFallbacks, opts.DefaultRegionID)
		}
		candidates := makeRegionCandidates(regionID, regionFallbacks)
		if len(candidates) == 0 {
			candidates = []string{""}
		}
		for _, candidateRegionID := range candidates {
			regionNodes := filterByMetadata(nodes, registry.MetadataRegionIDKey, candidateRegionID)
			if len(regionNodes) == 0 {
				continue
			}
			filtered := applyIDCStrategy(ctx, regionNodes, opts, idcID, hashKey)
			if len(filtered) > 0 {
				return filtered
			}
		}
		return nil
	}
}

func applyIDCStrategy(ctx context.Context, regionNodes []selector.Node, opts LocationOptions, idcID, hashKey string) []selector.Node {
	switch opts.IDCStrategy {
	case IDCStrategyOneRegion:
		return regionNodes
	case IDCStrategySelf:
		return filterByMetadata(regionNodes, registry.MetadataIDCIDKey, idcID)
	case IDCStrategyCHash:
		if hashKey == "" {
			return regionNodes
		}
		return hashIDCID(regionNodes, hashKey)
	case IDCStrategySelfThenCHash:
		self := filterByMetadata(regionNodes, registry.MetadataIDCIDKey, idcID)
		if len(self) > 0 {
			return self
		}
		if hashKey == "" {
			hashKey = fmt.Sprintf("%p", ctx)
		}
		return hashIDCID(regionNodes, hashKey)
	case IDCStrategySelfThenSeq:
		self := filterByMetadata(regionNodes, registry.MetadataIDCIDKey, idcID)
		if len(self) > 0 {
			return self
		}
		for _, seqIDCID := range opts.SequenceIDCIDs {
			groupNodes := filterByMetadata(regionNodes, registry.MetadataIDCIDKey, seqIDCID)
			if len(groupNodes) > 0 {
				return groupNodes
			}
		}
		return nil
	case IDCStrategyNormal:
		fallthrough
	default:
		self := filterByMetadata(regionNodes, registry.MetadataIDCIDKey, idcID)
		if len(self) > 0 {
			return self
		}
		return regionNodes
	}
}

func makeRegionCandidates(primary string, fallbacks []string) []string {
	candidates := make([]string, 0, len(fallbacks)+1)
	seen := make(map[string]struct{}, len(fallbacks)+1)
	appendCandidate := func(regionID string) {
		if regionID == "" {
			return
		}
		if _, ok := seen[regionID]; ok {
			return
		}
		seen[regionID] = struct{}{}
		candidates = append(candidates, regionID)
	}
	appendCandidate(primary)
	for _, fallback := range fallbacks {
		appendCandidate(fallback)
	}
	return candidates
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
