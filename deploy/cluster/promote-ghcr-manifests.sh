#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 4 ]]; then
  echo "Usage: $0 <image> <tag> <amd64-digest> <arm64-digest>" >&2
  exit 2
fi

image=$1
image_tag=$2
amd64_digest=$3
arm64_digest=$4

if [[ ! "$image" =~ ^ghcr\.io/[a-z0-9._-]+/[a-z0-9._-]+$ ]]; then
  echo "Invalid GHCR image: $image" >&2
  exit 1
fi
if [[ ! "$image_tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([._-][a-zA-Z0-9.-]+)*$ ]]; then
  echo "Invalid immutable image tag: $image_tag" >&2
  exit 1
fi
if [[ ! "$amd64_digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then
  echo "Invalid amd64 digest: $amd64_digest" >&2
  exit 1
fi
if [[ ! "$arm64_digest" =~ ^sha256:[a-f0-9]{64}$ ]]; then
  echo "Invalid arm64 digest: $arm64_digest" >&2
  exit 1
fi
if [[ "$amd64_digest" == "$arm64_digest" ]]; then
  echo "Architecture digests must differ" >&2
  exit 1
fi

digest_for_ref() {
  local ref=$1
  local raw_manifest
  raw_manifest=$(mktemp)
  docker buildx imagetools inspect "$ref" --raw > "$raw_manifest"
  printf 'sha256:%s' "$(sha256sum "$raw_manifest" | awk '{print $1}')"
  rm -f "$raw_manifest"
}

ensure_arch_tag() {
  local suffix=$1
  local expected_digest=$2
  local ref="$image:$image_tag-$suffix"

  if docker buildx imagetools inspect "$ref" >/dev/null 2>&1; then
    local actual_digest
    actual_digest=$(digest_for_ref "$ref")
    if [[ "$actual_digest" != "$expected_digest" ]]; then
      echo "Refusing to overwrite $ref: expected $expected_digest, found $actual_digest" >&2
      exit 1
    fi
    echo "Reusing matching partial-release tag: $ref@$actual_digest"
    return
  fi

  docker buildx imagetools create \
    --prefer-index=false \
    --tag "$ref" \
    "$image@$expected_digest"
  local published_digest
  published_digest=$(digest_for_ref "$ref")
  if [[ "$published_digest" != "$expected_digest" ]]; then
    echo "Published tag verification failed for $ref: expected $expected_digest, found $published_digest" >&2
    exit 1
  fi
  echo "Published immutable architecture tag: $ref@$expected_digest"
}

multiarch_ref="$image:$image_tag"
if docker buildx imagetools inspect "$multiarch_ref" >/dev/null 2>&1; then
  echo "Refusing to overwrite existing multi-architecture tag: $multiarch_ref" >&2
  exit 1
fi

ensure_arch_tag amd64 "$amd64_digest"
ensure_arch_tag arm64 "$arm64_digest"

if docker buildx imagetools inspect "$multiarch_ref" >/dev/null 2>&1; then
  echo "Refusing to overwrite existing multi-architecture tag: $multiarch_ref" >&2
  exit 1
fi

docker buildx imagetools create \
  --tag "$multiarch_ref" \
  "$image@$amd64_digest" \
  "$image@$arm64_digest"

multiarch_manifest=$(mktemp)
docker buildx imagetools inspect "$multiarch_ref" --raw > "$multiarch_manifest"
if ! jq -e \
  --arg amd64_digest "$amd64_digest" \
  --arg arm64_digest "$arm64_digest" \
  '
    (.manifests | length) == 2 and
    any(.manifests[]; .digest == $amd64_digest and .platform.os == "linux" and .platform.architecture == "amd64") and
    any(.manifests[]; .digest == $arm64_digest and .platform.os == "linux" and .platform.architecture == "arm64")
  ' "$multiarch_manifest" >/dev/null; then
  echo "Multi-architecture manifest verification failed: $multiarch_ref" >&2
  rm -f "$multiarch_manifest"
  exit 1
fi
rm -f "$multiarch_manifest"

echo
echo "Published immutable multi-architecture tag: $multiarch_ref"
echo "Expected linux/amd64 digest: $amd64_digest"
echo "Expected linux/arm64 digest: $arm64_digest"

for ref in \
  "$image:$image_tag-amd64" \
  "$image:$image_tag-arm64" \
  "$multiarch_ref"; do
  echo
  docker buildx imagetools inspect "$ref"
done
