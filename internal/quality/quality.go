package quality

import (
	"regexp"
	"strconv"
	"strings"
)

// Info is quality metadata parsed from a stream's title.
type Info struct {
	Resolution int    // 2160, 1080, 720, 480; 0 = unknown
	HDR        bool
	DV         bool   // Dolby Vision
	Codec      string // "x265", "x264", "AV1", ""
	Source     string // "REMUX", "BluRay", "WEB-DL", "WEBRip", "HDTV", ""
}

var (
	reRes2160 = regexp.MustCompile(`(?i)\b(2160p|4k|uhd)\b`)
	reRes1080 = regexp.MustCompile(`(?i)\b1080p?\b`)
	reRes720  = regexp.MustCompile(`(?i)\b720p?\b`)
	reRes480  = regexp.MustCompile(`(?i)\b480p?\b`)
	reResP    = regexp.MustCompile(`(?i)\b(\d{3,4})p\b`)
	reHDR     = regexp.MustCompile(`(?i)\bhdr10?\+?\b|\bhdr\b`)
	reDV      = regexp.MustCompile(`(?i)\bdolby ?vision\b|\bdovi\b|\bdv\b`)
	reAV1     = regexp.MustCompile(`(?i)\bav1\b`)
	reX265    = regexp.MustCompile(`(?i)\bx265\b|\bh\.?265\b|\bhevc\b`)
	reX264    = regexp.MustCompile(`(?i)\bx264\b|\bh\.?264\b|\bavc\b`)
	reRemux   = regexp.MustCompile(`(?i)\bremux\b`)
	reBluRay  = regexp.MustCompile(`(?i)\bblu-?ray\b|\bbdrip\b|\bbrrip\b`)
	reWebDL   = regexp.MustCompile(`(?i)\bweb-?dl\b`)
	reWebRip  = regexp.MustCompile(`(?i)\bweb-?rip\b|\bwebrip\b|\bweb\b`)
	reHDTV    = regexp.MustCompile(`(?i)\bhdtv\b`)
)

// Parse extracts quality info from a stream title.
func Parse(title string) Info {
	var in Info
	switch {
	case reRes2160.MatchString(title):
		in.Resolution = 2160
	case reRes1080.MatchString(title):
		in.Resolution = 1080
	case reRes720.MatchString(title):
		in.Resolution = 720
	case reRes480.MatchString(title):
		in.Resolution = 480
	default:
		if m := reResP.FindStringSubmatch(title); m != nil {
			in.Resolution, _ = strconv.Atoi(m[1])
		}
	}
	in.HDR = reHDR.MatchString(title)
	in.DV = reDV.MatchString(title)
	switch {
	case reAV1.MatchString(title):
		in.Codec = "AV1"
	case reX265.MatchString(title):
		in.Codec = "x265"
	case reX264.MatchString(title):
		in.Codec = "x264"
	}
	switch {
	case reRemux.MatchString(title):
		in.Source = "REMUX"
	case reBluRay.MatchString(title):
		in.Source = "BluRay"
	case reWebDL.MatchString(title):
		in.Source = "WEB-DL"
	case reWebRip.MatchString(title):
		in.Source = "WEBRip"
	case reHDTV.MatchString(title):
		in.Source = "HDTV"
	}
	return in
}

// Label returns a compact human label, e.g. "1080p HDR x265 WEB-DL".
func (in Info) Label() string {
	var parts []string
	if in.Resolution > 0 {
		parts = append(parts, strconv.Itoa(in.Resolution)+"p")
	}
	switch {
	case in.DV:
		parts = append(parts, "DV")
	case in.HDR:
		parts = append(parts, "HDR")
	}
	if in.Codec != "" {
		parts = append(parts, in.Codec)
	}
	if in.Source != "" {
		parts = append(parts, in.Source)
	}
	if len(parts) == 0 {
		return "?"
	}
	return strings.Join(parts, " ")
}

// Score ranks a stream (higher = better): resolution dominates, then HDR/DV,
// then source tier, then a small modern-codec bonus. Seeders break ties upstream.
func (in Info) Score() int {
	s := in.Resolution * 100
	switch {
	case in.DV:
		s += 40
	case in.HDR:
		s += 30
	}
	switch in.Source {
	case "REMUX":
		s += 25
	case "BluRay":
		s += 20
	case "WEB-DL":
		s += 15
	case "WEBRip":
		s += 10
	case "HDTV":
		s += 5
	}
	switch in.Codec {
	case "x265", "AV1":
		s += 3
	}
	return s
}
