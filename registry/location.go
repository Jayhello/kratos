package registry

const (
	// MetadataRegionIDKey is the metadata key for region id.
	MetadataRegionIDKey = "region_id"
	// MetadataIDCIDKey is the metadata key for idc id.
	MetadataIDCIDKey = "idc_id"
)

// SetLocationMetadata sets region/idc metadata on service instance.
func SetLocationMetadata(service *ServiceInstance, regionID, idcID string) {
	if service == nil {
		return
	}
	if service.Metadata == nil {
		service.Metadata = make(map[string]string, 2)
	}
	service.Metadata[MetadataRegionIDKey] = regionID
	service.Metadata[MetadataIDCIDKey] = idcID
}

// LocationMetadata returns region/idc metadata from service instance.
func LocationMetadata(service *ServiceInstance) (regionID, idcID string) {
	if service == nil || service.Metadata == nil {
		return "", ""
	}
	return service.Metadata[MetadataRegionIDKey], service.Metadata[MetadataIDCIDKey]
}
