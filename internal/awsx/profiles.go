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

// readSectionNames returns every INI section header in path, with a leading
// "profile " prefix stripped (the shared config file names non-default
// profiles "[profile NAME]"; the credentials file and the default profile in
// either file use a bare "[NAME]").
func readSectionNames(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, "]") {
			continue
		}
		name := strings.TrimSpace(line[1 : len(line)-1])
		name = strings.TrimPrefix(name, "profile ")
		if name != "" {
			names = append(names, name)
		}
	}
	return names, scanner.Err()
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
