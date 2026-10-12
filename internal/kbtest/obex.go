package kbtest

import (
	"embed"
	"fmt"
)

// OBEX fixture ids. The first three are byte copies of live KB v1.24.4 object
// bodies (p2kbCommunity<id>); OBEXMalformedID is deliberately malformed and is
// not a real object.
const (
	// OBEXStringCategoryID has category: Protocol (a string) and download_direct.
	OBEXStringCategoryID = "2842"
	// OBEXListCategoryID has category [Math, Motor Control, Protocol, Snippet].
	OBEXListCategoryID = "2811"
	// OBEXNoDownloadID has category [Display, Other, Protocol] and no download_direct.
	OBEXNoDownloadID = "4581"
	// OBEXMalformedID has functionality.category as a mapping, which must fail to parse.
	OBEXMalformedID = "9990"
)

// OBEXObjectsPath is where the KB keeps OBEX object files, repo-relative. It
// mirrors obex.OBEXPath, which kbtest cannot import without a cycle.
const OBEXObjectsPath = "deliverables/ai/P2/community/obex/objects"

//go:embed testdata/obex/*.yaml
var obexFixtures embed.FS

// OBEXObject returns the fixture body for object id. It panics on an unknown
// id: every caller names one of the constants above.
func OBEXObject(id string) string {
	b, err := obexFixtures.ReadFile("testdata/obex/" + id + ".yaml")
	if err != nil {
		panic(fmt.Sprintf("kbtest: no OBEX fixture %q: %v", id, err))
	}
	return string(b)
}

// AddOBEX lists fixture object id in the index (key p2kbCommunity<id>) and
// serves its body under OBEXObjectsPath, as the KB does.
func (r *Remote) AddOBEX(id string) {
	r.AddFile("p2kbCommunity"+id, OBEXObjectsPath+"/"+id+".yaml", 1700000000, OBEXObject(id))
}
