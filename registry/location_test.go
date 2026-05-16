package registry

import "testing"

func TestSetLocationMetadata(t *testing.T) {
	svc := &ServiceInstance{}
	SetLocationMetadata(svc, "1", "123")

	if svc.Metadata[MetadataRegionIDKey] != "1" {
		t.Fatalf("expect region id %q, got %q", "1", svc.Metadata[MetadataRegionIDKey])
	}
	if svc.Metadata[MetadataIDCIDKey] != "123" {
		t.Fatalf("expect idc id %q, got %q", "123", svc.Metadata[MetadataIDCIDKey])
	}
}

func TestLocationMetadata(t *testing.T) {
	svc := &ServiceInstance{
		Metadata: map[string]string{
			MetadataRegionIDKey: "2",
			MetadataIDCIDKey:    "345",
		},
	}
	regionID, idcID := LocationMetadata(svc)
	if regionID != "2" {
		t.Fatalf("expect region id %q, got %q", "2", regionID)
	}
	if idcID != "345" {
		t.Fatalf("expect idc id %q, got %q", "345", idcID)
	}
}

func TestLocationMetadataNil(t *testing.T) {
	regionID, idcID := LocationMetadata(nil)
	if regionID != "" || idcID != "" {
		t.Fatalf("expect empty ids for nil service, got region=%q idc=%q", regionID, idcID)
	}
}
