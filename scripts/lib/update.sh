#!/bin/bash

json_escape() {
  echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

base64_encode() {
  printf "%s" "$1" | base64 | tr -d '\n'
}

base64_decode() {
  if base64 -D >/dev/null 2>&1 <<<''; then
    printf "%s" "$1" | base64 -D
  else
    printf "%s" "$1" | base64 -d
  fi
}

get_file_mtime() {
  local target="$1"

  if [ -z "$target" ] || [ ! -f "$target" ]; then
    echo ""
    return
  fi

  if stat -f %m "$target" >/dev/null 2>&1; then
    stat -f %m "$target"
  elif stat -c %Y "$target" >/dev/null 2>&1; then
    stat -c %Y "$target"
  else
    echo ""
  fi
}

load_update_cache() {
  local now_ts="$1"

  if [ ! -f "$UPDATE_CACHE_FILE" ]; then
    return 1
  fi

  local mtime
  mtime="$(get_file_mtime "$UPDATE_CACHE_FILE")"
  if [ -z "$mtime" ]; then
    return 1
  fi

  local age=$((now_ts - mtime))
  if [ "$age" -ge "$UPDATE_CACHE_TTL_SECONDS" ]; then
    return 1
  fi

  local cached_status cached_method cached_local cached_latest cached_message
  cached_status="$(sed -n 's/^status_b64=//p' "$UPDATE_CACHE_FILE" | head -n 1)"
  cached_method="$(sed -n 's/^method_b64=//p' "$UPDATE_CACHE_FILE" | head -n 1)"
  cached_local="$(sed -n 's/^local_b64=//p' "$UPDATE_CACHE_FILE" | head -n 1)"
  cached_latest="$(sed -n 's/^latest_b64=//p' "$UPDATE_CACHE_FILE" | head -n 1)"
  cached_message="$(sed -n 's/^message_b64=//p' "$UPDATE_CACHE_FILE" | head -n 1)"

  if [ -z "$cached_status" ]; then
    return 1
  fi

  cached_status="$(base64_decode "$cached_status")"
  cached_method="$(base64_decode "$cached_method")"
  cached_local="$(base64_decode "$cached_local")"
  cached_latest="$(base64_decode "$cached_latest")"
  cached_message="$(base64_decode "$cached_message")"

  if [ -n "$cached_local" ] && [ "$cached_local" != "$AGENT47_VERSION" ]; then
    return 1
  fi

  UPDATE_STATUS="$cached_status"
  UPDATE_METHOD="$cached_method"
  UPDATE_LOCAL_VERSION="${cached_local:-$AGENT47_VERSION}"
  UPDATE_LATEST_VERSION="$cached_latest"
  UPDATE_MESSAGE="$cached_message"
  UPDATE_FROM_CACHE=true
  UPDATE_CACHE_AGE="$age"
  return 0
}

save_update_cache() {
  mkdir -p "$CACHE_DIR"

  local now_ts
  now_ts="$(date +%s)"
  cat >"$UPDATE_CACHE_FILE" <<EOF
checked_at=$now_ts
status_b64=$(base64_encode "$UPDATE_STATUS")
method_b64=$(base64_encode "$UPDATE_METHOD")
local_b64=$(base64_encode "$UPDATE_LOCAL_VERSION")
latest_b64=$(base64_encode "$UPDATE_LATEST_VERSION")
message_b64=$(base64_encode "$UPDATE_MESSAGE")
EOF
}

print_update_result() {
  if [ "${UPDATE_FROM_CACHE:-false}" = true ]; then
    local hours=$((UPDATE_CACHE_AGE / 3600))
    echo "[INFO] Using cached update check (age ${hours}h)"
  fi

  case "$UPDATE_STATUS" in
    up-to-date)
      echo "[OK] Up to date (version ${UPDATE_LOCAL_VERSION})"
      ;;
    update-available)
      echo "Update available: ${UPDATE_LOCAL_VERSION} -> ${UPDATE_LATEST_VERSION}"
      if [ -d "$ROOT_DIR/.git" ]; then
        echo "[HINT] Update via: git -C \"$ROOT_DIR\" pull && ./install.sh"
      else
        echo "[HINT] Update via: re-download agent47 and rerun install.sh"
      fi
      ;;
    local-ahead)
      echo "[INFO] Local copy is ahead of ${UPDATE_METHOD}; no update needed"
      ;;
    *)
      echo "[WARN] Cannot check for updates: ${UPDATE_MESSAGE}"
      return 1
      ;;
  esac

  return 0
}

git_update_check() {
  if ! command -v git >/dev/null 2>&1; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="git not available"
    UPDATE_METHOD="git"
    return 1
  fi

  if [ ! -d "$ROOT_DIR/.git" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="agent47 not installed from a git checkout"
    UPDATE_METHOD="git"
    return 1
  fi

  local remote_list
  remote_list="$(git -C "$ROOT_DIR" remote 2>/dev/null)"
  if [ -z "$remote_list" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="no git remotes configured"
    UPDATE_METHOD="git"
    return 1
  fi

  local remote
  if [ -n "${AGENT47_REMOTE:-}" ]; then
    remote="$AGENT47_REMOTE"
  else
    remote="$(echo "$remote_list" | head -n 1)"
  fi

  if ! git -C "$ROOT_DIR" fetch "$remote" --quiet 2>/dev/null; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="git fetch failed for remote '$remote' (network or access issue)"
    UPDATE_METHOD="git"
    return 1
  fi

  local upstream_ref
  upstream_ref="$(git -C "$ROOT_DIR" rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null)"

  local remote_head
  remote_head="$(git -C "$ROOT_DIR" symbolic-ref --quiet --short "refs/remotes/$remote/HEAD" 2>/dev/null)"

  local current_branch
  current_branch="$(git -C "$ROOT_DIR" rev-parse --abbrev-ref HEAD 2>/dev/null)"

  local remote_ref=""
  if [ -n "$upstream_ref" ]; then
    remote_ref="$upstream_ref"
  elif [ -n "$remote_head" ]; then
    remote_ref="$remote_head"
  elif [ -n "$current_branch" ] && [ "$current_branch" != "HEAD" ]; then
    remote_ref="$remote/$current_branch"
  fi

  if [ -z "$remote_ref" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="no upstream branch detected"
    UPDATE_METHOD="git"
    return 1
  fi

  local remote_commit local_commit
  remote_commit="$(git -C "$ROOT_DIR" rev-parse "$remote_ref" 2>/dev/null)"
  local_commit="$(git -C "$ROOT_DIR" rev-parse HEAD 2>/dev/null)"

  if [ -z "$remote_commit" ] || [ -z "$local_commit" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="unable to resolve commits for comparison"
    UPDATE_METHOD="git"
    return 1
  fi

  local remote_version
  remote_version="$(git -C "$ROOT_DIR" show "$remote_ref:VERSION" 2>/dev/null || echo "")"
  if [ -z "$remote_version" ]; then
    remote_version="unknown"
  fi

  UPDATE_LOCAL_VERSION="$AGENT47_VERSION"
  UPDATE_LATEST_VERSION="$remote_version"
  UPDATE_METHOD="git ($remote_ref)"

  if [ "$local_commit" = "$remote_commit" ]; then
    UPDATE_STATUS="up-to-date"
    UPDATE_MESSAGE="local matches $remote_ref"
    return 0
  fi

  local behind_count ahead_count
  behind_count="$(git -C "$ROOT_DIR" rev-list --count "$local_commit..$remote_ref" 2>/dev/null || echo "0")"
  ahead_count="$(git -C "$ROOT_DIR" rev-list --count "$remote_ref..$local_commit" 2>/dev/null || echo "0")"

  if [ "$behind_count" -gt 0 ]; then
    UPDATE_STATUS="update-available"
    UPDATE_MESSAGE="behind by $behind_count commit(s)"
    return 0
  fi

  if [ "$ahead_count" -gt 0 ]; then
    UPDATE_STATUS="local-ahead"
    UPDATE_MESSAGE="local branch is ahead of $remote_ref"
    return 0
  fi

  UPDATE_STATUS="error"
  UPDATE_MESSAGE="unable to determine ahead/behind status"
  return 1
}

remote_update_check() {
  if [ -z "$REMOTE_VERSION_URL" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="no remote VERSION URL configured"
    UPDATE_METHOD="remote"
    return 1
  fi

  if ! command -v curl >/dev/null 2>&1; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="curl not available for remote version check"
    UPDATE_METHOD="remote"
    return 1
  fi

  local latest_version
  latest_version="$(curl -fsSL --connect-timeout 5 --max-time 10 "$REMOTE_VERSION_URL" 2>/dev/null | head -n 1)"

  if [ -z "$latest_version" ]; then
    UPDATE_STATUS="error"
    UPDATE_MESSAGE="failed to download VERSION from $REMOTE_VERSION_URL"
    UPDATE_METHOD="remote"
    return 1
  fi

  UPDATE_LOCAL_VERSION="$AGENT47_VERSION"
  UPDATE_LATEST_VERSION="$latest_version"
  UPDATE_METHOD="remote"

  if [ "$AGENT47_VERSION" = "$latest_version" ]; then
    UPDATE_STATUS="up-to-date"
    UPDATE_MESSAGE="remote VERSION matches local"
  else
    UPDATE_STATUS="update-available"
    UPDATE_MESSAGE="remote VERSION differs"
  fi
  return 0
}

check_update() {
  UPDATE_STATUS=""
  UPDATE_METHOD=""
  UPDATE_LOCAL_VERSION="$AGENT47_VERSION"
  UPDATE_LATEST_VERSION=""
  UPDATE_MESSAGE=""
  UPDATE_FROM_CACHE=false
  UPDATE_CACHE_AGE=0

  local force_refresh=false
  if [ "${1:-}" = "--force" ]; then
    force_refresh=true
  fi

  mkdir -p "$CACHE_DIR"

  local now_ts
  now_ts="$(date +%s)"

  if [ "$force_refresh" = false ] && load_update_cache "$now_ts"; then
    print_update_result || true
    return 0
  fi

  if git_update_check; then
    save_update_cache
    print_update_result
    return 0
  fi

  if remote_update_check; then
    save_update_cache
    print_update_result
    return 0
  fi

  UPDATE_STATUS="error"
  if [ -z "$UPDATE_MESSAGE" ]; then
    UPDATE_MESSAGE="all update methods failed"
  fi
  print_update_result || true
  return 0
}
