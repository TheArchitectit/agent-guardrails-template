package controlplane

// Source digests (Spec 08 R08-4.1.1).
//
// A manifest that declares a digest nothing can recompute is a manifest that
// restates its own claim: the field looks like verification and is really a
// constant somebody typed. So the digest is defined here, once, and the CI gate
// calls THIS implementation rather than re-deriving it in another language —
// two implementations of one digest is how a manifest passes CI and fails at
// startup while describing the same tree.
//
// The canonical form is deliberately boring: for each declared path, sorted
// lexicographically, the line "<repo-relative path> <sha256 of that file's
// bytes>\n". Concatenating a sorted list of (path, content-hash) pairs means a
// file added, removed, renamed or edited all change the result, and neither
// directory order nor mtimes can.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SourceDigest computes the canonical digest of the declared source paths,
// resolved relative to root. A missing path, a directory, or an unreadable file
// is an error — never skipped. A digest that silently covers fewer files than
// declared is worse than no digest, because it reports "verified" for a scope it
// did not read.
func SourceDigest(root string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("component declares no source paths")
	}

	normalized := make([]string, 0, len(paths))
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimSpace(p)))
		if clean == "" || clean == "." || clean == ".." {
			return "", fmt.Errorf("source path %q is not a file path", p)
		}
		if filepath.IsAbs(clean) {
			return "", fmt.Errorf("source path %q must be repository-relative", p)
		}
		if strings.HasPrefix(clean, "../") {
			return "", fmt.Errorf("source path %q escapes the repository root", p)
		}
		if seen[clean] {
			return "", fmt.Errorf("source path %q is declared twice", clean)
		}
		seen[clean] = true
		normalized = append(normalized, clean)
	}
	sort.Strings(normalized)

	var buf strings.Builder
	for _, rel := range normalized {
		full := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(full)
		if err != nil {
			return "", fmt.Errorf("source path %q: %w", rel, err)
		}
		if info.IsDir() {
			// A directory digest would have to define recursive ordering,
			// ignore rules and empty-directory handling — three chances to
			// disagree with another implementation. Declare files.
			return "", fmt.Errorf("source path %q is a directory; declare files", rel)
		}
		content, err := os.ReadFile(full)
		if err != nil {
			return "", fmt.Errorf("read source path %q: %w", rel, err)
		}
		sum := sha256.Sum256(content)
		fmt.Fprintf(&buf, "%s %s\n", rel, hex.EncodeToString(sum[:]))
	}

	total := sha256.Sum256([]byte(buf.String()))
	return "sha256:" + hex.EncodeToString(total[:]), nil
}

// VerifySources recomputes every component's digest and reports each mismatch.
// It returns all findings rather than the first, so one pass tells an operator
// everything that drifted.
func VerifySources(root string, m *Manifest) ([]string, error) {
	if m == nil {
		return nil, fmt.Errorf("manifest is required")
	}
	var problems []string
	for _, c := range m.Components {
		got, err := SourceDigest(root, c.SourcePaths)
		if err != nil {
			problems = append(problems, fmt.Sprintf("component %q: %v", c.ID, err))
			continue
		}
		if got != c.Digest {
			problems = append(problems, fmt.Sprintf(
				"component %q: declared digest %s does not match the tree (%s)",
				c.ID, c.Digest, got))
		}
	}
	return problems, nil
}
