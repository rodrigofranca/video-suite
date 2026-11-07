package storage

// Object keys are always generated from IDs, never from user input
// (file names are only stored as metadata).

// SourceKey is where an asset's original upload is stored.
func SourceKey(assetID string) string {
	return "assets/" + assetID + "/source/input.mp4"
}

// HLSPrefix is the folder that holds an asset's HLS playlist and segments.
// Everything under assets/*/hls/ is publicly readable (docs/adr/0010).
func HLSPrefix(assetID string) string {
	return "assets/" + assetID + "/hls/"
}

// HLSManifestKey is the key of an asset's HLS playlist.
func HLSManifestKey(assetID string) string {
	return HLSPrefix(assetID) + "index.m3u8"
}
