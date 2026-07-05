package okayeg

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

// trustRelPath is where eg::trust keeps the trust set, re-read on every
// incoming connection. Matches TRUST_PATH in eg::trust.
const trustRelPath = ".eg/trust"

// egDir is the metadata subdirectory eg keeps at the repo root.
const egDir = ".eg"

// Grant is one parsed .eg/trust row.
type Grant struct {
	EndpointID string
	UserID     uint // parsed from a "user:<id>" tag, 0 if absent or unparseable
	Pull       bool
	Push       bool
	Revoked    bool
}

// SetTrust grants endpointID full access (pull+push) on the project at baseDir,
// tagging the row with the registering user so DB and file stay attributable.
// Replaces any existing row for the same endpointID: like eg::trust, the last
// row wins, so a repeat call also changes the grant. Preserves comments, blank
// lines, and other peers' rows.
func SetTrust(fs afero.Fs, baseDir string, endpointID string, userID uint) error {
	if !ValidEndpointID(endpointID) {
		return fmt.Errorf("invalid endpoint id %q", endpointID)
	}
	return rewriteTrust(fs, baseDir, func(lines []string) []string {
		kept := make([]string, 0, len(lines)+1)
		for _, line := range lines {
			if firstField(line) == endpointID {
				continue // drop the stale grant for this peer
			}
			kept = append(kept, line)
		}
		kept = append(kept, fmt.Sprintf("%s pull push user:%d", endpointID, userID))
		return kept
	})
}

// RemoveTrust drops any row for endpointID, preserving comments and other peers.
func RemoveTrust(fs afero.Fs, baseDir string, endpointID string) error {
	if !ValidEndpointID(endpointID) {
		return fmt.Errorf("invalid endpoint id %q", endpointID)
	}
	return rewriteTrust(fs, baseDir, func(lines []string) []string {
		kept := make([]string, 0, len(lines))
		for _, line := range lines {
			if firstField(line) == endpointID {
				continue
			}
			kept = append(kept, line)
		}
		return kept
	})
}

// ListTrust parses .eg/trust into the effective grant per endpoint id. Like
// eg::Trust::grants, the last matching row wins, in the order ids first appear.
func ListTrust(fs afero.Fs, baseDir string) ([]Grant, error) {
	text, err := readTrust(fs, baseDir)
	if err != nil {
		return nil, err
	}
	var rows []Grant
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if g, ok := parseGrant(line); ok {
			rows = append(rows, g)
		}
	}
	// collapse: last row wins, keep first-appear order
	grants := make([]Grant, 0, len(rows))
	for _, g := range rows {
		if i := indexOf(grants, g.EndpointID); i >= 0 {
			grants[i] = g
		} else {
			grants = append(grants, g)
		}
	}
	return grants, nil
}

func indexOf(gs []Grant, id string) int {
	for i, g := range gs {
		if g.EndpointID == id {
			return i
		}
	}
	return -1
}

// rewriteTrust reads trust lines, lets fn transform them, and writes the result
// back atomically under baseDir/.eg/.
func rewriteTrust(fs afero.Fs, baseDir string, fn func([]string) []string) error {
	text, err := readTrust(fs, baseDir)
	if err != nil {
		return err
	}
	// drop the trailing empty element from the final newline, if any
	raw := strings.Split(text, "\n")
	if n := len(raw); n > 0 && raw[n-1] == "" {
		raw = raw[:n-1]
	}
	out := fn(raw)
	text = strings.Join(out, "\n")
	if len(text) > 0 {
		text += "\n"
	}

	dir := filepath.Join(baseDir, egDir)
	if err := fs.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	tmp, err := afero.TempFile(fs, dir, ".trust-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		_ = fs.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = fs.Remove(tmp.Name())
		return err
	}
	if err := fs.Chmod(tmp.Name(), 0o600); err != nil {
		_ = fs.Remove(tmp.Name())
		return err
	}

	return fs.Rename(tmp.Name(), filepath.Join(baseDir, trustRelPath))
}

func readTrust(fs afero.Fs, baseDir string) (string, error) {
	bytes, err := afero.ReadFile(fs, filepath.Join(baseDir, trustRelPath))
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	return string(bytes), nil
}

// firstField returns the first whitespace-separated token of line, or "" for
// comment/blank lines. Used to match a row to an endpoint id without a full
// parse, so we never touch lines we didn't write.
func firstField(line string) string {
	line = strings.TrimLeft(line, " \t")
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	i := strings.IndexAny(line, " \t")
	if i < 0 {
		return line
	}
	return line[:i]
}

func parseGrant(line string) (Grant, bool) {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return Grant{}, false
	}

	id := tokens[0]
	if !ValidEndpointID(id) {
		// not a row we recognize; leave it for the file but skip from results
		return Grant{}, false
	}

	g := Grant{EndpointID: id}
	for _, tok := range tokens[1:] {
		switch {
		case tok == "pull":
			g.Pull = true
		case tok == "push":
			g.Push = true
		case tok == "revoked":
			g.Revoked = true
		case strings.HasPrefix(tok, "user:"):
			if n, err := strconv.ParseUint(tok[len("user:"):], 10, 64); err == nil {
				g.UserID = uint(n)
			}
		default:
			// tolerate unknown tags rather than dropping the row
		}
	}
	return g, true
}
