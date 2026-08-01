# NewAPI e00a04a5 Production Deployment

## Target

- Commit: `e00a04a52e974d86ee855651c8ca8876fdc1641a`
- Image: `ghcr.io/buglyz/new-api:sha-e00a04a52e974d86ee855651c8ca8876fdc1641a`
- Service: `/opt/newapi`, container `new-api`
- Action: https://github.com/buglyz/new-api/actions/runs/30530494927

## Checklist

- [x] Push target commit without force.
- [x] Confirm multi-arch Action completed successfully.
- [x] Pull and verify exact SHA image.
- [x] Create SQLite, Compose, and old-image rollback points.
- [x] Pin Compose and recreate only `new-api`.
- [x] Verify health, SQLite, local/public endpoints, logs, and a real stream.

## Rollback

- Manifest: `/opt/newapi/backups/deploy-e00a04a5-20260730-173304.manifest`
- Database: `/opt/newapi/backups/one-api-before-e00a04a5-20260730-173304.db`
- Original Compose: `/opt/newapi/backups/compose-before-e00a04a5-20260730-173304.yaml`
- Rollback Compose: `/opt/newapi/backups/compose-rollback-e00a04a5-20260730-173304.yaml`
- Old image source: `calciumion/new-api@sha256:d600f20c2781e1a173c2a02f8c33b0c4b1b4e8e5a8b107bafaf2442ae2c9386c`
- The local rc.22 image was removed during the later unused-image cleanup;
  rollback now requires a network pull of this immutable digest.

```bash
cp /opt/newapi/backups/compose-before-e00a04a5-20260730-173304.yaml /opt/newapi/compose.yaml
docker compose --project-directory /opt/newapi \
  -f /opt/newapi/backups/compose-rollback-e00a04a5-20260730-173304.yaml \
  up -d --force-recreate new-api
```

Restore the database backup only if a schema migration prevents the old image
from starting. Stop the container before an atomic database restore.

```bash
docker stop --timeout 30 new-api
install -m 0644 \
  /opt/newapi/backups/one-api-before-e00a04a5-20260730-173304.db \
  /opt/newapi/data/one-api.db.restore
mv /opt/newapi/data/one-api.db.restore /opt/newapi/data/one-api.db
docker compose --project-directory /opt/newapi \
  -f /opt/newapi/backups/compose-rollback-e00a04a5-20260730-173304.yaml \
  up -d --force-recreate new-api
```

## Result

- Action run `30530494927` completed successfully.
- Multi-arch digest: `sha256:84960ef3a075685b19dfdddbe4c3b8e488e12b15abdf27e0c373bb46ae60772e`.
- Runtime version/revision: `main-e00a04a` / `e00a04a52e974d86ee855651c8ca8876fdc1641a`.
- Container is healthy with zero restarts; SQLite `quick_check` is `ok`.
- Local/public status, homepage, and `/dashboard/models` return HTTP 200.
- Controlled public `gpt-5.6-sol` Responses stream returned HTTP 200,
  emitted `response.completed`, and logged normal `end_reason=eof` on channel 13.
- No startup panic, fatal error, database error, or migration failure was found.
- The first Compose recreation found a stale same-service container label and
  stopped before touching the running service. The stale replacement was removed,
  then the verified new container took over the fixed `new-api` name successfully.
