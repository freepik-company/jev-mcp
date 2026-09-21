#!/bin/sh
# Checks server.json against the MCP Registry rules that only surface at publish time.
# Seen in the wild: the v0.3.0 publish failed with "expected length <= 100" on description.
set -eu
cd "$(dirname "$0")/.."
python3 - <<'PY'
import json, re, sys
server = json.load(open("server.json"))
errors = []
if len(server["description"]) > 100:
    errors.append("description has %d characters; the registry accepts at most 100" % len(server["description"]))
if not re.fullmatch(r"io\.github\.[A-Za-z0-9-]+/[A-Za-z0-9._-]+", server["name"]):
    errors.append("name must be io.github.<owner>/<server> for GitHub OIDC publishing")
if not re.fullmatch(r"\d+\.\d+\.\d+([-+][0-9A-Za-z.-]+)?", server["version"]):
    errors.append("version must be bare semver, got %r" % server["version"])
for package in server["packages"]:
    if package["registryType"] == "oci" and not package["identifier"].startswith("ghcr.io/"):
        errors.append("oci identifier must live on ghcr.io, got %r" % package["identifier"])
label = 'io.modelcontextprotocol.server.name="%s"' % server["name"]
if label not in open("Dockerfile").read():
    errors.append("Dockerfile must carry LABEL %s so the registry can verify image ownership" % label)
for error in errors:
    print("server.json:", error, file=sys.stderr)
sys.exit(1 if errors else 0)
PY
echo "server.json ok"
