package audio

import (
	"context"
	"encoding/json"
	"fmt"
	"math/bits"
)

// Fingerprint is a Chromaprint acoustic fingerprint of one file. The Raw vector
// is a sequence of 32-bit sub-fingerprint hashes; comparing two Raw vectors by
// bit-error-rate identifies the same song across different names/bitrates.
//
// This is entirely LOCAL — we compare the user's own files to each other. No
// network, no AcoustID, no API key. (Online identification, which WOULD need a
// database, was deliberately dropped from scope.)
type Fingerprint struct {
	Path        string   `json:"path"`
	Raw         []uint32 `json:"-"`
	DurationSec float64  `json:"durationSec"`
}

// Fingerprint computes a Chromaprint fingerprint via fpcalc. -raw gives the
// integer vector needed for bit-level comparison; -length caps analysis at the
// first 120s (plenty to match, and fast).
func (r *Runner) Fingerprint(ctx context.Context, path string) (Fingerprint, error) {
	if !r.HasFpcalc() {
		return Fingerprint{}, fmt.Errorf("fpcalc not available")
	}
	stdout, stderr, err := r.run(ctx, fpTimeout, r.fpcalc,
		"-raw", "-json", "-length", "120", path,
	)
	if err != nil {
		return Fingerprint{}, fmt.Errorf("fpcalc %s: %w (%s)", path, err, tail(stderr))
	}
	fp, err := parseFpcalcJSON(stdout)
	if err != nil {
		return Fingerprint{}, err
	}
	fp.Path = path
	return fp, nil
}

func parseFpcalcJSON(data []byte) (Fingerprint, error) {
	var raw struct {
		Duration    float64  `json:"duration"`
		Fingerprint []uint32 `json:"fingerprint"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return Fingerprint{}, fmt.Errorf("parse fpcalc json: %w", err)
	}
	return Fingerprint{Raw: raw.Fingerprint, DurationSec: raw.Duration}, nil
}

// dupeSimilarityThreshold: 1 - max-bit-error-rate. 0.85 ≈ allow 15% of bits to
// differ, which tolerates different encoders/bitrates of the same recording
// without matching merely-similar songs.
const dupeSimilarityThreshold = 0.85

// GroupDuplicates clusters fingerprints whose pairwise similarity exceeds
// threshold. Returns groups of >=2 file paths. Pure (no exec) for unit testing.
func GroupDuplicates(fps []Fingerprint, threshold float64) [][]string {
	n := len(fps)
	uf := newUnionFind(n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if similarity(fps[i].Raw, fps[j].Raw) >= threshold {
				uf.union(i, j)
			}
		}
	}

	groups := map[int][]string{}
	for i := 0; i < n; i++ {
		root := uf.find(i)
		groups[root] = append(groups[root], fps[i].Path)
	}

	var out [][]string
	for _, paths := range groups {
		if len(paths) >= 2 {
			out = append(out, paths)
		}
	}
	return out
}

// similarity returns 1 - minimum bit-error-rate over a small alignment window,
// so two recordings that differ by a short lead-in still match.
func similarity(a, b []uint32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	best := berAtOffset(a, b, 0)
	const maxOffset = 12
	for off := 1; off <= maxOffset; off++ {
		if ber := berAtOffset(a, b, off); ber < best {
			best = ber
		}
		if ber := berAtOffset(b, a, off); ber < best {
			best = ber
		}
	}
	return 1 - best
}

// berAtOffset computes the mean per-bit error rate aligning b shifted right by
// offset against a, over their overlapping region.
func berAtOffset(a, b []uint32, offset int) float64 {
	overlap := minInt(len(a)-offset, len(b))
	if overlap <= 0 {
		return 1
	}
	var diffBits int
	for i := 0; i < overlap; i++ {
		diffBits += bits.OnesCount32(a[i+offset] ^ b[i])
	}
	return float64(diffBits) / float64(overlap*32)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- union-find ---

type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{parent: p}
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra != rb {
		u.parent[ra] = rb
	}
}
