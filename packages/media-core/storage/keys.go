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

// ThumbnailKey is where an asset's thumbnail image is stored. Thumbnails are
// publicly readable, like the HLS output (docs/adr/0013).
func ThumbnailKey(assetID string) string {
	return "assets/" + assetID + "/thumbnails/thumbnail.jpg"
}

// AudioKey is where an asset's extracted audio (16 kHz mono FLAC) is stored.
// It stays private: only the workers read it.
func AudioKey(assetID string) string {
	return "assets/" + assetID + "/audio/audio.flac"
}

// SubtitlesKey is where an asset's WebVTT subtitles for a language are
// stored. Subtitles are publicly readable, like the HLS output (docs/adr/0017).
func SubtitlesKey(assetID, language string) string {
	return "assets/" + assetID + "/subtitles/" + language + ".vtt"
}
