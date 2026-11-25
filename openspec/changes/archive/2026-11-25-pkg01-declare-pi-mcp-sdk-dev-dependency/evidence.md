# PKG-01 verification

The Pi package locks MCP SDK 1.32.2 as a development dependency. The existing
production dependencies, optional host peer settings, and third-party lock
versions and integrity hashes are preserved. The added lock records are
reachable through the SDK development graph.
[SDK release](https://github.com/modelcontextprotocol/typescript-sdk/releases/tag/1.32.2)

The clean locked install, dependency-tree verification, and package test suite
passed with 199 tests. The tests exercise the published SDK client subpaths and
the bridge fallback in an isolated runtime with the SDK absent.

The package dry run reported 70 files. An isolated installation from the
packed tarball omitted development dependencies and confirmed the SDK was
absent. With the Pi host peer supplied as a test stub, the extension loaded
its standalone path and reported the MCP bridge unavailable without throwing.

The independent clean-install check passed all 12 focused bridge tests.
