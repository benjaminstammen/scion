#!/bin/bash
set -eo pipefail

# Build scion images locally using plain docker build (no buildx).
# Images land directly in the local Docker daemon.
#
# Uses local-only image names during the build to avoid remote registry
# resolution, then tags with the registry prefix so scion can find them.
#
# Usage:
#   build-local.sh --registry <registry> [--target <target>] [--tag <tag>]
#
# Examples:
#   build-local.sh --registry ghcr.io/benjaminstammen
#   build-local.sh --registry ghcr.io/benjaminstammen --target harnesses
#   build-local.sh --registry ghcr.io/benjaminstammen --target core-base

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
IMAGE_BUILD_DIR="${REPO_ROOT}/image-build"

REGISTRY=""
TARGET="common"
TAG="latest"
HARNESSES=(claude gemini opencode codex)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --registry)   REGISTRY="$2"; shift 2 ;;
    --target)     TARGET="$2"; shift 2 ;;
    --tag)        TAG="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $(basename "$0") --registry <registry> [--target common|all|core-base|harnesses] [--tag <tag>]"
      exit 0 ;;
    *)            echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "${REGISTRY}" ]]; then
  echo "Error: --registry is required"
  exit 1
fi

REGISTRY="${REGISTRY%/}"

# Tag a local image with the registry prefix so scion can find it.
retag() {
  local local_name="$1"
  local registry_name="${REGISTRY}/${local_name}"
  if [[ "$local_name" != "$registry_name" ]]; then
    docker tag "${local_name}" "${registry_name}"
  fi
}

build_core_base() {
  echo "==> Building core-base..."
  docker build \
    -t "core-base:${TAG}" \
    -f "${IMAGE_BUILD_DIR}/core-base/Dockerfile" \
    "${IMAGE_BUILD_DIR}/core-base"
  retag "core-base:${TAG}"
}

build_scion_base() {
  echo "==> Building scion-base..."
  docker build \
    --build-arg "BASE_IMAGE=core-base:${1:-latest}" \
    --build-arg "GIT_COMMIT=$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || echo unknown)" \
    -t "scion-base:${TAG}" \
    -f "${IMAGE_BUILD_DIR}/scion-base/Dockerfile" \
    "${REPO_ROOT}"
  retag "scion-base:${TAG}"
}

build_harness() {
  local name="$1"
  echo "==> Building scion-${name}..."
  docker build \
    --build-arg "BASE_IMAGE=scion-base:${2:-latest}" \
    -t "scion-${name}:${TAG}" \
    -f "${IMAGE_BUILD_DIR}/${name}/Dockerfile" \
    "${IMAGE_BUILD_DIR}/${name}"
  retag "scion-${name}:${TAG}"
}

case "${TARGET}" in
  common)
    build_scion_base "latest"
    for h in "${HARNESSES[@]}"; do build_harness "${h}" "${TAG}"; done
    ;;
  all)
    build_core_base
    build_scion_base "${TAG}"
    for h in "${HARNESSES[@]}"; do build_harness "${h}" "${TAG}"; done
    ;;
  core-base)
    build_core_base
    ;;
  harnesses)
    for h in "${HARNESSES[@]}"; do build_harness "${h}" "latest"; done
    ;;
  *)
    echo "Unknown target: ${TARGET}"; exit 1 ;;
esac

echo ""
echo "Done. Images are in your local Docker daemon."
