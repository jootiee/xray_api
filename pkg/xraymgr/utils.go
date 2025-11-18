package xraymgr

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
)

func newUUID() string { return uuid.New().String() }

func newShortID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func usernameToEmail(u string) string { return fmt.Sprintf("%s@example.com", u) }
func emailToUsername(e string) string {
	parts := strings.Split(e, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return e
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyWithBackup copies src to dst, moving existing dst to dst.backup (rotating one level).
func copyWithBackup(src, dst string) error {
	if fileExists(dst) {
		backup := dst + ".backup"
		if fileExists(backup) {
			_ = os.Remove(backup)
		}
		if err := os.Rename(dst, backup); err != nil {
			return fmt.Errorf("backup rename: %w", err)
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
