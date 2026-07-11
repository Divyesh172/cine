package provider

import "context"

func init() { Register("demo", newDemo) }

// demo is a legal, keyless provider returning Creative-Commons sample videos
// (Blender open movies via the reliable W3C media samples). It ignores the query
// and always responds, so you can exercise search -> select -> mpv end to end.
type demo struct{}

func newDemo(opts map[string]interface{}) (Provider, error) { return &demo{}, nil }

func (d *demo) Name() string { return "Demo (open movies)" }

func (d *demo) Search(ctx context.Context, q Query) ([]Stream, error) {
	const base = "https://test-videos.co.uk/vids/bigbuckbunny/mp4"
	// Real Big Buck Bunny clips (Blender open movie) at matching resolutions,
	// all H.264 so they decode cleanly. Quality tags are honest, and seeders are
	// deliberately highest on the LOWER-quality entries to prove the Step 5
	// ranker beats raw seeder count.
	return []Stream{
		{Provider: d.Name(), Title: "Big Buck Bunny 1080p x264 BluRay", Seeders: 500, URL: base + "/h264/1080/Big_Buck_Bunny_1080_10s_5MB.mp4"},
		{Provider: d.Name(), Title: "Big Buck Bunny 720p x264 BluRay", Seeders: 900, URL: base + "/h264/720/Big_Buck_Bunny_720_10s_2MB.mp4"},
		{Provider: d.Name(), Title: "Big Buck Bunny 720p x264 WEB-DL", Seeders: 999, URL: base + "/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"},
		{Provider: d.Name(), Title: "Big Buck Bunny 360p x264 WEBRip", Seeders: 850, URL: base + "/h264/360/Big_Buck_Bunny_360_10s_1MB.mp4"},
		// Magnet entry (Sintel — the official WebTorrent demo torrent, CC-licensed).
		// Pick this to exercise the Step 6 peerflix bridge. Requires peerflix.
		{Provider: d.Name(), Title: "Sintel (magnet demo) 1080p x264 BluRay", Seeders: 300, URL: sintelMagnet},
	}, nil
}

const sintelMagnet = "magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969"
