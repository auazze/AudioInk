package main

// ManualEntry is the JSON shape exchanged between the main process and the
// confirm-dialog child process (see confirm.go). JSON tags are required so
// the child's results survive marshal/unmarshal.
type ManualEntry struct {
	Artist  string `json:"artist"`
	Title   string `json:"title"`
	Skipped bool   `json:"skipped"`
}
