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
// authz command, and its session hook at a session command when one is given.
// eg runs the authz argv for each incoming connection, writes the peer's
// endpoint id to its stdin, and reads "pull"/"push" from its stdout; a nonzero
// exit refuses the peer. The session argv runs per peer join and leave with
// the event, endpoint id, and claimed name and email on stdin; it decides
// nothing.
func WriteAuthzConfig(fs afero.Fs, baseDir string, authz, session []string) error {
	if len(authz) == 0 {
		return fmt.Errorf("okayeg: authz command must name a program")
	}

	table := func(name string, argv []string) (string, error) {
		quoted := make([]string, len(argv))
		for i, arg := range argv {
			q, err := tomlString(arg)
			if err != nil {
				return "", err
			}
			quoted[i] = q
		}
		return fmt.Sprintf("[%s]\ncommand = [%s]\n", name, strings.Join(quoted, ", ")), nil
	}

	text, err := table("authz", authz)
	if err != nil {
		return err
	}
	if len(session) > 0 {
		hook, err := table("session", session)
		if err != nil {
			return err
		}
		text += hook
	}

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
