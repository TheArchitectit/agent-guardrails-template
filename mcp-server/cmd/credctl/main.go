// Command credctl is the operator interface to the registered-credential
// lifecycle required by Spec 16 Sec. 4 R16-09.
//
// Before this command the repository could verify, reload, rotate and revoke
// credentials and could not ISSUE one: Digest() had a single non-test caller
// (Resolve), so minting a credential meant computing an HMAC by hand and
// hand-writing registry JSON that the loader might then reject at startup.
//
// R16-09 explicitly says "Do not claim a lifecycle API exists until one is
// implemented and protected." This is that API, and it is deliberately narrow:
//
//   - the credential secret is never accepted in argv and never printed to a
//     log; it is written to a 0600 file the operator creates (R17-08)
//   - the verifier key is read from a restricted secret file or the
//     environment, never from argv (R17-06 / R17-08)
//   - every write goes through the same validation the server uses
//     (auth.New -> buildSnapshot), so credctl cannot author a registry the
//     server will reject, and cannot author one weaker than the server accepts
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/secretfile"
)

const usage = `credctl — registered-credential lifecycle (Spec 16 R16-09)

Usage:
  credctl issue   --registry <file> --id <credential-id> --principal <principal-id>
                  --role <role> --scopes <csv> [--resources <csv>]
                  [--days <n>] --secret-out <file>
  credctl revoke  --registry <file> --id <credential-id>
  credctl list    --registry <file>

Verifier key:
  --verifier-key-file <file>   restricted (0600) secret file, or
  CREDENTIAL_VERIFIER_KEY      environment variable
  The key is never accepted as an argv value: /proc/<pid>/cmdline and
  container inspect output are readable by other local processes.

Secret handling:
  The issued secret is written to --secret-out (0600, must not already exist).
  It is never printed, so it never reaches a terminal scrollback, CI log or
  shell history. Hand the file to the credential holder and delete it.

Roles:      reader | developer | security-operator | administrator
Scopes:     mcp:read | mcp:validate | mcp:mutate | rest:read | rest:write |
            ide:validate | admin:manage
Validity:   1..90 days (Spec 16 R16-09: new credentials expire within 90 days)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	registry := fs.String("registry", "", "path to the credential registry JSON file (0600)")
	verifierKeyFile := fs.String("verifier-key-file", "", "restricted file holding the credential verifier key")
	credID := fs.String("id", "", "credential id")
	principal := fs.String("principal", "", "principal id")
	role := fs.String("role", "", "role from the published catalog")
	scopes := fs.String("scopes", "", "comma-separated scopes from the published catalog")
	resources := fs.String("resources", "", "comma-separated resource/project ids (default: none)")
	days := fs.Int("days", 0, "validity in days (1..90, default 90)")
	secretOut := fs.String("secret-out", "", "file to receive the issued secret (0600, must not exist)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if strings.TrimSpace(*registry) == "" {
		fail("--registry is required")
	}

	key, err := resolveVerifierKey(*verifierKeyFile)
	if err != nil {
		fail(err.Error())
	}

	switch cmd {
	case "issue":
		runIssue(key, *registry, *credID, *principal, *role, *scopes, *resources, *days, *secretOut)
	case "revoke":
		runRevoke(key, *registry, *credID)
	case "list":
		runList(*registry)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", cmd, usage)
		os.Exit(2)
	}
}

func runIssue(key, registry, credID, principal, role, scopeCSV, resourceCSV string, days int, secretOut string) {
	if strings.TrimSpace(secretOut) == "" {
		fail("--secret-out is required: the issued secret is never printed to stdout")
	}
	if _, err := os.Lstat(secretOut); err == nil {
		fail(fmt.Sprintf("refusing to overwrite existing %s: the secret is delivered once", secretOut))
	}
	validityDays := days
	if validityDays == 0 {
		validityDays = int(auth.MaxCredentialValidity.Hours() / 24)
	}
	if validityDays < 1 || time.Duration(validityDays)*24*time.Hour > auth.MaxCredentialValidity {
		fail(fmt.Sprintf("--days must be between 1 and %d", int(auth.MaxCredentialValidity.Hours()/24)))
	}

	// Load and validate the existing set first: an operator who mistyped the
	// registry path must not create a brand-new file containing one record,
	// silently replacing a live registry with a single credential.
	records, err := auth.LoadRegistryFile(registry)
	if err != nil {
		fail(err.Error())
	}

	rec, secret, err := auth.IssueRecord(key, auth.IssueOptions{
		CredentialID: credID,
		PrincipalID:  principal,
		Role:         role,
		Scopes:       splitCSV(scopeCSV),
		Resources:    splitCSV(resourceCSV),
		Validity:     time.Duration(validityDays) * 24 * time.Hour,
	})
	if err != nil {
		fail(err.Error())
	}
	for _, existing := range records {
		if existing.CredentialID == rec.CredentialID {
			fail(fmt.Sprintf("credential id %q already exists in %s", rec.CredentialID, registry))
		}
	}

	if err := auth.WriteRegistryFile(registry, key, append(records, rec)); err != nil {
		fail(err.Error())
	}
	// Write the secret only after the registry is durable: if the registry
	// write fails, a secret file on disk would be a credential that authorizes
	// nothing and that nobody can revoke (it is not in the file).
	if err := secretfile.Write(secretOut, []byte(secret+"\n")); err != nil {
		fail(err.Error())
	}

	fmt.Printf("issued credential_id=%s principal_id=%s role=%s expires_at=%s\n",
		rec.CredentialID, rec.PrincipalID, rec.Role, rec.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("secret written to %s (0600). Deliver it, then delete the file.\n", secretOut)
	fmt.Printf("registry now holds %d record(s): %s\n", len(records)+1, registry)
}

func runRevoke(key, registry, credID string) {
	if strings.TrimSpace(credID) == "" {
		fail("--id is required")
	}
	rec, err := auth.RevokeInFile(registry, key, credID)
	if err != nil {
		fail(err.Error())
	}
	fmt.Printf("revoked credential_id=%s principal_id=%s\n", rec.CredentialID, rec.PrincipalID)
	fmt.Println("the running server denies it on its next registry reload (Spec 16 R16-09 bound: 60s)")
}

func runList(registry string) {
	records, err := auth.LoadRegistryFile(registry)
	if err != nil {
		fail(err.Error())
	}
	now := time.Now()
	for _, r := range records {
		state := "active"
		switch {
		case r.Revoked:
			state = "revoked"
		case !r.ExpiresAt.IsZero() && now.After(r.ExpiresAt):
			state = "expired"
		}
		fmt.Printf("%-24s principal=%-20s role=%-18s %-8s expires=%s scopes=%s\n",
			r.CredentialID, r.PrincipalID, r.Role, state,
			r.ExpiresAt.UTC().Format(time.RFC3339), strings.Join(r.Scopes, ","))
	}
}

// resolveVerifierKey reads the key from a restricted secret file or the
// environment. An argv value is deliberately not supported (R17-08).
func resolveVerifierKey(filePath string) (string, error) {
	if strings.TrimSpace(filePath) != "" {
		data, err := secretfile.Read(filePath)
		if err != nil {
			return "", err
		}
		key := strings.TrimSpace(string(data))
		if key == "" {
			return "", fmt.Errorf("verifier key file %s is empty", filePath)
		}
		return key, nil
	}
	key := strings.TrimSpace(os.Getenv("CREDENTIAL_VERIFIER_KEY"))
	if key == "" {
		return "", fmt.Errorf("no verifier key: pass --verifier-key-file or set CREDENTIAL_VERIFIER_KEY")
	}
	return key, nil
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "credctl: %s\n", msg)
	os.Exit(1)
}
