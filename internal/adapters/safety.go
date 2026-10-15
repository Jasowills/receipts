package adapters

import (
	"fmt"
	"os"
)

// CheckFileSize rejects transcript files over MaxTranscriptBytes before
// their contents are read. Missing files return the underlying error so
// callers can distinguish absence from oversize.
func CheckFileSize(path string) error {
	return checkFileSizeLimit(path, MaxTranscriptBytes)
}

func checkFileSizeLimit(path string, limit int64) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Size() > limit {
		return fmt.Errorf("adapters: transcript %s size %d exceeds %d bytes", path, fi.Size(), limit)
	}
	return nil
}
