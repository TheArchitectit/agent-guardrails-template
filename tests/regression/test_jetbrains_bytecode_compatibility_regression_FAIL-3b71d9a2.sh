#!/usr/bin/env sh
# Regression test for FAILURE-ID: FAIL-3b71d9a2
# Bug: the plugin emitted Java 21 class files although its pinned IntelliJ
# platform bundles JBR 17; buildPlugin failed while loading the plugin classes.
# Fix: enforce Java release 17 and Kotlin JVM/API release 17.

set -eu
repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
plugin_source="$repo_root/ide/jetbrains-plugin"

grep -F 'options.release.set(17)' "$plugin_source/build.gradle.kts" >/dev/null
grep -F 'kotlinOptions.jvmTarget = "17"' "$plugin_source/build.gradle.kts" >/dev/null
grep -F 'kotlinOptions.freeCompilerArgs += "-Xjdk-release=17"' "$plugin_source/build.gradle.kts" >/dev/null
grep -F 'gradle-8.6-bin.zip' "$plugin_source/gradle/wrapper/gradle-wrapper.properties" >/dev/null

java_version=$(java -version 2>&1 | head -n 1)
case "$java_version" in
  *'"21.'*) ;;
  *) echo "JDK 21 is required; found: $java_version" >&2; exit 2 ;;
esac

tmp_root=$(mktemp -d)
cleanup() { rm -rf "$tmp_root"; }
trap cleanup EXIT HUP INT TERM
plugin_copy="$tmp_root/jetbrains-plugin"
mkdir -p "$plugin_copy"
cp -R "$plugin_source"/. "$plugin_copy"/
# Normalize only the disposable copy for Linux when the source came from a
# Windows checkout with CRLF in the wrapper script.
sed 's/\r$//' "$plugin_copy/gradlew" > "$tmp_root/gradlew.lf"
mv "$tmp_root/gradlew.lf" "$plugin_copy/gradlew"
chmod +x "$plugin_copy/gradlew"

build_log="$tmp_root/gradle-build-plugin.log"
cd "$plugin_copy"
if ! ./gradlew clean buildPlugin --no-daemon --stacktrace >"$build_log" 2>&1; then
  cat "$build_log"
  exit 1
fi
for task in compileKotlin instrumentCode buildSearchableOptions buildPlugin; do
  grep -E "^> Task :$task([[:space:]]|$)" "$build_log" >/dev/null || {
    echo "expected Gradle task did not run: $task" >&2
    cat "$build_log"
    exit 1
  }
done
grep -F 'BUILD SUCCESSFUL' "$build_log" >/dev/null
plugin_zip=$(find "$plugin_copy/build/distributions" -maxdepth 1 -name '*.zip' -print -quit)
if [ -z "$plugin_zip" ]; then
  echo "buildPlugin completed without a plugin ZIP" >&2
  exit 1
fi

python3 - "$plugin_zip" <<'PY'
import collections
import io
import sys
import zipfile
import xml.etree.ElementTree as ET

artifact = sys.argv[1]
plugin_jar_name = None
with zipfile.ZipFile(artifact) as distribution:
    candidates = [name for name in distribution.namelist()
                  if name.endswith("/guardrail-plugin-1.0.0.jar")]
    if len(candidates) != 1:
        raise SystemExit(f"expected one plugin jar, found {candidates}")
    plugin_jar_name = candidates[0]
    with zipfile.ZipFile(io.BytesIO(distribution.read(plugin_jar_name))) as plugin:
        descriptor = ET.fromstring(plugin.read("META-INF/plugin.xml"))
        version = descriptor.find("idea-version")
        if version is None or not version.get("since-build") or not version.get("until-build"):
            raise SystemExit("plugin descriptor does not declare both build bounds")
        class_files = [name for name in plugin.namelist()
                       if name.startswith("com/guardrail/") and name.endswith(".class")]
        majors = collections.Counter(
            int.from_bytes(plugin.read(name)[6:8], "big")
            for name in class_files
            if plugin.read(name)[:4] == bytes.fromhex("cafebabe")
        )
        if not class_files or set(majors) != {61}:
            raise SystemExit(f"expected all plugin classes at major 61, got {dict(majors)}")
        print(f"artifact={artifact}")
        print(f"plugin_jar={plugin_jar_name}")
        print(f"descriptor={version.attrib}")
        print(f"plugin_classes={len(class_files)} major_counts={dict(majors)}")
PY

echo "PASS: JetBrains plugin ZIP builds and targets the pinned JBR 17 runtime"
