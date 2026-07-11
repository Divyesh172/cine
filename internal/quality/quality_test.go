package quality

import "testing"

func TestParseResolution(t *testing.T) {
	cases := map[string]int{
		"Movie 2160p HDR x265":     2160,
		"Show S01E01 1080p WEB-DL": 1080,
		"Something 720p BluRay":    720,
		"Old 480p":                 480,
		"No resolution here":       0,
		"Weird 1440p release":      1440,
	}
	for title, want := range cases {
		if got := Parse(title).Resolution; got != want {
			t.Errorf("Parse(%q).Resolution = %d, want %d", title, got, want)
		}
	}
}

func TestParseSourceCodecHDR(t *testing.T) {
	in := Parse("The Film 2022 1080p BluRay REMUX x265 HDR")
	if in.Source != "REMUX" {
		t.Errorf("Source = %q, want REMUX", in.Source)
	}
	if in.Codec != "x265" {
		t.Errorf("Codec = %q, want x265", in.Codec)
	}
	if !in.HDR {
		t.Error("HDR = false, want true")
	}
}

func TestScoreOrdering(t *testing.T) {
	better := Parse("Movie 2160p BluRay x265").Score()
	worse := Parse("Movie 720p WEBRip x264").Score()
	if better <= worse {
		t.Errorf("expected 2160p BluRay (%d) to outscore 720p WEBRip (%d)", better, worse)
	}
}

func TestLabel(t *testing.T) {
	if got := Parse("Movie 1080p WEB-DL x264").Label(); got != "1080p x264 WEB-DL" {
		t.Errorf("Label() = %q, want %q", got, "1080p x264 WEB-DL")
	}
}
