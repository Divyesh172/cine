package provider

import (
	"encoding/xml"
	"testing"
)

func TestJackettSize(t *testing.T) {
	cases := map[int64]string{
		0:          "",
		512:        "512 B",
		1024:       "1.00 KB",
		1536:       "1.50 KB",
		1073741824: "1.00 GB",
	}
	for in, want := range cases {
		if got := jackettSize(in); got != want {
			t.Errorf("jackettSize(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestTorznabParse(t *testing.T) {
	const body = `<?xml version="1.0"?>
<rss><channel>
<item>
  <title>Example.Movie.2022.1080p.BluRay.x264</title>
  <size>1610612736</size>
  <enclosure url="magnet:?xt=urn:btih:deadbeef" type="application/x-bittorrent"/>
  <torznab:attr name="seeders" value="17"/>
</item>
</channel></rss>`
	var resp torznabResp
	if err := xml.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Channel.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.Channel.Items))
	}
	it := resp.Channel.Items[0]
	if it.seeders() != 17 {
		t.Errorf("seeders = %d, want 17", it.seeders())
	}
	if it.magnet() != "magnet:?xt=urn:btih:deadbeef" {
		t.Errorf("magnet = %q, want magnet:?xt=urn:btih:deadbeef", it.magnet())
	}
}
