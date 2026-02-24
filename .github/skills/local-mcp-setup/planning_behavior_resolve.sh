#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT=""
PROFILE_FILE=""
OUT_PATH="docs/planning-behavior-resolution.md"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target-root)
      TARGET_ROOT="${2:-}"
      shift 2
      ;;
    --profile-file)
      PROFILE_FILE="${2:-}"
      shift 2
      ;;
    --out)
      OUT_PATH="${2:-}"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$TARGET_ROOT" ]]; then
  echo "--target-root is required" >&2
  exit 2
fi

TARGET_ROOT="$(cd "$TARGET_ROOT" && pwd)"

resolve_profile() {
  local explicit="$1"
  if [[ -n "$explicit" ]]; then
    if [[ -f "$explicit" ]]; then
      echo "$explicit"
      return 0
    fi
    if [[ -f "$TARGET_ROOT/$explicit" ]]; then
      echo "$TARGET_ROOT/$explicit"
      return 0
    fi
  fi

  local candidates=(
    "$TARGET_ROOT/docs/source/02-architecture/planning-behavior-profile.yaml"
    "$TARGET_ROOT/docs/source/DemoArchitectureDocs/planning-behavior-profile.yaml"
    "$TARGET_ROOT/.github/skills/local-mcp-setup/corporate-docs/planning-behavior-profile.yaml"
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [[ -f "$candidate" ]]; then
      echo "$candidate"
      return 0
    fi
  done

  return 1
}

PROFILE_PATH="$(resolve_profile "$PROFILE_FILE" || true)"
if [[ -z "$PROFILE_PATH" ]]; then
  echo "planning behavior profile not found" >&2
  exit 2
fi

if [[ "$OUT_PATH" != /* ]]; then
  OUT_PATH="$TARGET_ROOT/$OUT_PATH"
fi
mkdir -p "$(dirname "$OUT_PATH")"

yaml_scalar() {
  local key="$1"
  local line
  line="$(grep -E "^[[:space:]]*$key:[[:space:]]*" "$PROFILE_PATH" | head -n 1 || true)"
  if [[ -z "$line" ]]; then
    echo "<unset>"
    return
  fi
  echo "$line" | sed -E "s/^[[:space:]]*$key:[[:space:]]*//" | sed -E "s/^['\"]?(.*?)['\"]?$/\1/"
}

NOW_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

cat > "$OUT_PATH" <<EOF
# Planning Behavior Resolution

- Timestamp (UTC): $NOW_UTC
- Target root: $TARGET_ROOT
- Profile source: ${PROFILE_PATH#$TARGET_ROOT/}

## Resolved controls
- profile_id: $(yaml_scalar profile_id)
- topology.default: $(yaml_scalar default)
- topology.service_per_repo_best_practice: $(yaml_scalar service_per_repo_best_practice)
- contracts.contract_first_required: $(yaml_scalar contract_first_required)
- contracts.require_openapi_for_http: $(yaml_scalar require_openapi_for_http)
- workstreams.dependency_graph_required: $(yaml_scalar dependency_graph_required)
- reviews.code_review_feedback_loop_required: $(yaml_scalar code_review_feedback_loop_required)
- integration.orchestration_requires_all_prerequisites_passed: $(yaml_scalar orchestration_requires_all_prerequisites_passed)
- evidence.ai_usage_metrics_required_when_available: $(yaml_scalar ai_usage_metrics_required_when_available)

## Notes
- Values are resolved from the selected profile source for this run.
- Missing scalar values are marked as \\`<unset>\\` and should be treated as planning blockers where required by workflow gates.
EOF

echo "$OUT_PATH"
