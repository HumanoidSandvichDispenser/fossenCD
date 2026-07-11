package okayeg

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// configRelPath is where eg reads its per-repo config. Matches CONFIG_PATH in
// eg::config.
const configRelPath = ".eg/config.toml"

// WriteAuthzConfig writes .eg/config.toml pointing eg's connection gate at an
// authz command. eg runs the argv for each incoming connection, writes the
// peer's endpoint id to its stdin, and reads "pull"/"push" from its stdout; a
// nonzero exit refuses the peer.
func WriteAuthzConfig(fs afero.Fs, baseDir string, authz []string) error {
	if len(authz) == 0 {
		return fmt.Errorf("okayeg: authz command must name a program")
	}

	quoted := make([]string, len(authz))
	for i, arg := range authz {
		q, err := tomlString(arg)
		if err != nil {
			return err
		}
		quoted[i] = q
	}
	text := fmt.Sprintf("[authz]\ncommand = [%s]\n", strings.Join(quoted, ", "))

	if err := fs.MkdirAll(filepath.Join(baseDir, egDir), 0o700); err != nil {
		return err
	}
	return afero.WriteFile(fs, filepath.Join(baseDir, configRelPath), []byte(text), 0o600)
}

// tomlString quotes s as a TOML basic string. Control characters are rejected
// rather than escaped; nothing we write (paths, project ids) contains them.
func tomlString(s string) (string, error) {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return "", fmt.Errorf("okayeg: control character in authz command arg %q", s)
		}
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`, nil
}
