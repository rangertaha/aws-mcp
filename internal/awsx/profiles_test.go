// SPDX-License-Identifier: MIT

package awsx

import (
	"os"
	"path/filepath"
	"testing"
)

// withSharedFiles points AWS_CONFIG_FILE/AWS_SHARED_CREDENTIALS_FILE at fresh
// files in a temp directory for the duration of the test, isolating it from
// any real ~/.aws files on the host.
func withSharedFiles(t *testing.T, configBody, credentialsBody string) {
	t.Helper()
	dir := t.TempDir()

	configPath := filepath.Join(dir, "config")
	credsPath := filepath.Join(dir, "credentials")
	if configBody != "" {
		if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if credentialsBody != "" {
		if err := os.WriteFile(credsPath, []byte(credentialsBody), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credsPath)
}

func TestListProfilesMergesAndDedupes(t *testing.T) {
	withSharedFiles(t, `
[default]
region = us-east-1

[profile staging]
region = us-west-2
`, `
[default]
aws_access_key_id = x

[staging]
aws_access_key_id = y

[prod]
aws_access_key_id = z
`)

	got, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}

	want := []string{"default", "prod", "staging"}
	if len(got) != len(want) {
		t.Fatalf("ListProfiles() = %v, want %v", got, want)
	}
	for i, name := range want {
		if got[i] != name {
			t.Fatalf("ListProfiles() = %v, want %v", got, want)
		}
	}
}

func TestListProfilesMissingFilesIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(dir, "no-such-config"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(dir, "no-such-credentials"))

	got, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ListProfiles() = %v, want empty", got)
	}
}

func TestListProfilesExcludesSSOSessionAndServicesSections(t *testing.T) {
	withSharedFiles(t, `
[profile my-sso-profile]
sso_session = my-sso
sso_account_id = 123456789012
sso_role_name = MyRole
region = us-east-1

[sso-session my-sso]
sso_region = us-east-1
sso_start_url = https://my-sso-portal.awsapps.com/start
sso_registration_scopes = sso:account:access

[services my-services]
s3 =
  endpoint_url = https://s3.example.com
`, "")

	got, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	want := []string{"my-sso-profile"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("ListProfiles() = %v, want %v (sso-session/services sections must not appear as profiles)", got, want)
	}
}

func TestListProfilesStripsProfilePrefixOnly(t *testing.T) {
	withSharedFiles(t, `
[profile a]
region = us-east-1

[b]
region = us-east-1
`, "")

	got, err := ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("ListProfiles() = %v, want %v", got, want)
	}
}
