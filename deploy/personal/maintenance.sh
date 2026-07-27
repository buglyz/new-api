#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-${SCRIPT_DIR}/compose.yml}"
IMAGE_ENV_FILE="${IMAGE_ENV_FILE:-${SCRIPT_DIR}/.image.env}"
APP_ENV_FILE="${APP_ENV_FILE:-${SCRIPT_DIR}/.env}"
DATA_DIR="${DATA_DIR:-${SCRIPT_DIR}/data}"
BACKUP_ROOT="${BACKUP_ROOT:-${SCRIPT_DIR}/backups}"
DATABASE_FILE="${DATABASE_FILE:-${DATA_DIR}/one-api.db}"
HEALTH_TIMEOUT_SECONDS="${HEALTH_TIMEOUT_SECONDS:-150}"
LAST_BACKUP_DIR=""

usage() {
  cat <<'EOF'
Usage:
  maintenance.sh status
  maintenance.sh backup
  maintenance.sh upgrade IMAGE
  maintenance.sh rollback BACKUP_DIR [--restore-database] [--restore-config]

IMAGE must be immutable: ghcr.io/...@sha256:... or :sha-<40 hex commit>.
rollback restores the image only unless an explicit restore flag is supplied.
EOF
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "required command not found: $1"
}

compose() {
  docker compose --env-file "$IMAGE_ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

validate_paths() {
  [[ -f "$COMPOSE_FILE" ]] || die "compose file not found: $COMPOSE_FILE"
  [[ -f "$IMAGE_ENV_FILE" ]] || die "image env file not found: $IMAGE_ENV_FILE"
  [[ -f "$APP_ENV_FILE" ]] || die "application env file not found: $APP_ENV_FILE"
  [[ "$BACKUP_ROOT" != *"'"* && "$DATABASE_FILE" != *"'"* ]] || die "single quotes are not supported in backup paths"
}

read_image() {
  local image
  image="$(sed -n 's/^NEW_API_IMAGE=//p' "$IMAGE_ENV_FILE" | tail -n 1)"
  [[ -n "$image" ]] || die "NEW_API_IMAGE is missing from $IMAGE_ENV_FILE"
  printf '%s\n' "$image"
}

validate_immutable_image() {
  local image="$1"
  if [[ "$image" =~ @sha256:[0-9a-f]{64}$ ]]; then
    return
  fi
  if [[ "$image" =~ :sha-[0-9a-f]{40}$ ]]; then
    return
  fi
  die "image must use a digest or full commit SHA tag"
}

write_image() {
  local image="$1"
  local temp_file
  temp_file="$(mktemp "${IMAGE_ENV_FILE}.XXXXXX")"
  awk -v image="$image" '
    BEGIN { replaced = 0 }
    /^NEW_API_IMAGE=/ {
      print "NEW_API_IMAGE=" image
      replaced = 1
      next
    }
    { print }
    END {
      if (!replaced) print "NEW_API_IMAGE=" image
    }
  ' "$IMAGE_ENV_FILE" >"$temp_file"
  chmod --reference="$IMAGE_ENV_FILE" "$temp_file"
  mv -f -- "$temp_file" "$IMAGE_ENV_FILE"
}

wait_for_health() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT_SECONDS))
  local container_id status
  while ((SECONDS < deadline)); do
    container_id="$(compose ps -q new-api)"
    if [[ -n "$container_id" ]]; then
      status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id" 2>/dev/null || true)"
      if [[ "$status" == "healthy" ]]; then
        return 0
      fi
      if [[ "$status" == "exited" || "$status" == "dead" ]]; then
        return 1
      fi
    fi
    sleep 2
  done
  return 1
}

backup_database() {
  local destination="$1"
  [[ -f "$DATABASE_FILE" ]] || die "SQLite database not found: $DATABASE_FILE"
  require_command sqlite3
  sqlite3 "$DATABASE_FILE" 'PRAGMA busy_timeout=5000; PRAGMA wal_checkpoint(PASSIVE);' >/dev/null
  sqlite3 "$DATABASE_FILE" ".timeout 5000" ".backup '${destination}'"
  [[ "$(sqlite3 "$destination" 'PRAGMA quick_check;')" == "ok" ]] || die "SQLite backup quick_check failed"
}

create_backup() {
  validate_paths
  local timestamp backup_dir image
  timestamp="$(date -u +'%Y%m%dT%H%M%SZ')"
  mkdir -p -- "$BACKUP_ROOT"
  chmod 0700 "$BACKUP_ROOT"
  backup_dir="$(mktemp -d "${BACKUP_ROOT}/${timestamp}.XXXXXX")"
  chmod 0700 "$backup_dir"
  image="$(read_image)"

  backup_database "${backup_dir}/one-api.db"
  install -m 0600 "$IMAGE_ENV_FILE" "${backup_dir}/image.env"
  install -m 0600 "$APP_ENV_FILE" "${backup_dir}/app.env"
  install -m 0600 "$COMPOSE_FILE" "${backup_dir}/compose.yml"
  printf 'created_at_utc=%s\nimage=%s\ndatabase=%s\n' \
    "$timestamp" "$image" "$DATABASE_FILE" >"${backup_dir}/manifest"
  chmod 0600 "${backup_dir}/manifest"
  LAST_BACKUP_DIR="$backup_dir"
  printf '%s\n' "$backup_dir"
}

upgrade_image() {
  local new_image="$1"
  require_command docker
  validate_paths
  validate_immutable_image "$new_image"
  compose config --quiet

  local old_image
  old_image="$(read_image)"
  validate_immutable_image "$old_image"
  create_backup >/dev/null
  printf 'backup: %s\n' "$LAST_BACKUP_DIR"
  write_image "$new_image"

  if ! compose pull new-api || ! compose up -d --no-deps new-api || ! wait_for_health; then
    printf 'upgrade failed; restoring image %s\n' "$old_image" >&2
    write_image "$old_image"
    compose up -d --no-deps new-api
    wait_for_health || die "automatic image rollback did not become healthy"
    die "upgrade failed and image was rolled back"
  fi
  printf 'healthy image: %s\n' "$new_image"
}

resolve_backup_dir() {
  local requested="$1"
  local root resolved
  root="$(realpath -e "$BACKUP_ROOT")"
  resolved="$(realpath -e "$requested")"
  [[ "$resolved" == "$root"/* ]] || die "backup must be inside $BACKUP_ROOT"
  [[ -f "${resolved}/manifest" && -f "${resolved}/image.env" ]] || die "invalid backup directory"
  printf '%s\n' "$resolved"
}

rollback_backup() {
  local requested="$1"
  shift
  local restore_database=false restore_config=false flag backup_dir image
  for flag in "$@"; do
    case "$flag" in
      --restore-database) restore_database=true ;;
      --restore-config) restore_config=true ;;
      *) die "unknown rollback flag: $flag" ;;
    esac
  done

  require_command docker
  validate_paths
  backup_dir="$(resolve_backup_dir "$requested")"
  image="$(sed -n 's/^NEW_API_IMAGE=//p' "${backup_dir}/image.env" | tail -n 1)"
  validate_immutable_image "$image"
  create_backup >/dev/null
  printf 'pre-rollback backup: %s\n' "$LAST_BACKUP_DIR"
  docker image inspect "$image" >/dev/null 2>&1 || docker pull "$image"

  if [[ "$restore_database" == true || "$restore_config" == true ]]; then
    compose stop new-api
  fi
  if [[ "$restore_database" == true ]]; then
    [[ -f "${backup_dir}/one-api.db" ]] || die "database is missing from backup"
    install -m 0600 "${backup_dir}/one-api.db" "$DATABASE_FILE"
    rm -f -- "${DATABASE_FILE}-wal" "${DATABASE_FILE}-shm"
  fi
  if [[ "$restore_config" == true ]]; then
    [[ -f "${backup_dir}/app.env" ]] || die "application config is missing from backup"
    install -m 0600 "${backup_dir}/app.env" "$APP_ENV_FILE"
  fi

  write_image "$image"
  compose up -d --no-deps new-api
  wait_for_health || die "rollback did not become healthy"
  printf 'rollback healthy: %s\n' "$image"
}

show_status() {
  require_command docker
  validate_paths
  local image container_id status
  image="$(read_image)"
  container_id="$(compose ps -q new-api)"
  status="absent"
  if [[ -n "$container_id" ]]; then
    status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container_id")"
  fi
  printf 'image=%s\ncontainer=%s\nstatus=%s\ndatabase=%s\n' "$image" "${container_id:-}" "$status" "$DATABASE_FILE"
}

main() {
  local command="${1:-}"
  case "$command" in
    status) show_status ;;
    backup) create_backup ;;
    upgrade)
      [[ $# -eq 2 ]] || die "upgrade requires one immutable image"
      upgrade_image "$2"
      ;;
    rollback)
      [[ $# -ge 2 ]] || die "rollback requires a backup directory"
      shift
      rollback_backup "$@"
      ;;
    -h|--help|help) usage ;;
    *) usage; exit 1 ;;
  esac
}

main "$@"
