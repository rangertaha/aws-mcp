// SPDX-License-Identifier: MIT

package awsx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListProfiles returns the names of every AWS profile discovered in the
// shared config and credentials files (~/.aws/config, ~/.aws/credentials, or
// the paths named by AWS_CONFIG_FILE/AWS_SHARED_CREDENTIALS_FILE),
// deduplicated and sorted. A missing file is not an error.
func ListProfiles() ([]string, error) {
	seen := make(map[string]struct{})

	for _, path := range []string{configFilePath(), credentialsFilePath()} {
		names, err := readSectionNames(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		for _, n := range names {
			seen[n] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

// nonProfileSectionPrefixes are shared-config section types that use the
// same "[type name]" header syntax as a profile but are not themselves
// profiles: an "[sso-session NAME]" is referenced by a profile's
// sso_session key, and a "[services NAME]" is referenced by a profile's
// services key for per-service endpoint overrides. Both are config-file-only
// constructs (the credentials file holds only bare-named profiles), but
// excluding them here unconditionally is safe: a credentials-file profile
// legitimately named exactly "sso-session ..." or "services ..." would be
// exceedingly unlikely.
var nonProfileSectionPrefixes = []string{"sso-session ", "services "}

// readSectionNames returns every INI section header in path that names an
// actual profile, with a leading "profile " prefix stripped (the shared
// config file names non-default profiles "[profile NAME]"; the credentials
// file and the default profile in either file use a bare "[NAME]").
// Non-profile section types (see nonProfileSectionPrefixes) are excluded.
func readSectionNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		name := strings.TrimSpace(line[1 : len(line)-1])
		if isNonProfileSection(name) {
			continue
		}
		name = strings.TrimPrefix(name, "profile ")
		if name != "" {
			names = append(names, name)
		}
	}
	return names, scanner.Err()
}

func isNonProfileSection(name string) bool {
	for _, p := range nonProfileSectionPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func configFilePath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

func credentialsFilePath() string {
	if p := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}
