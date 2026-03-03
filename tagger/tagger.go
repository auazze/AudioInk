package tagger

import (
	"fmt"

	"go.senan.xyz/taglib"
)

type Tags struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Album  string `json:"album"`
	Track  int    `json:"track"`
}

func Read(path string) (Tags, error) {
	props, err := taglib.ReadTags(path)
	if err != nil {
		return Tags{}, fmt.Errorf("read tags %s: %w", path, err)
	}

	track := 0
	if v, ok := props["tracknumber"]; ok && len(v) > 0 {
		for _, c := range v[0] {
			if c >= '0' && c <= '9' {
				track = track*10 + int(c-'0')
			} else {
				break
			}
		}
	}

	return Tags{
		Artist: first(props["artist"]),
		Title:  first(props["title"]),
		Album:  first(props["album"]),
		Track:  track,
	}, nil
}

func Write(path string, tags Tags) error {
	props := make(map[string][]string)
	if tags.Artist != "" {
		props["artist"] = []string{tags.Artist}
	}
	if tags.Title != "" {
		props["title"] = []string{tags.Title}
	}
	if tags.Album != "" {
		props["album"] = []string{tags.Album}
	}
	if tags.Track > 0 {
		props["tracknumber"] = []string{fmt.Sprintf("%d", tags.Track)}
	}

	if err := taglib.WriteTags(path, props, 0); err != nil {
		return fmt.Errorf("write tags %s: %w", path, err)
	}
	return nil
}

func first(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
