# Personal Deployment and Rollback

This directory provides a single-container SQLite template and explicit maintenance commands. Nothing here changes an existing deployment automatically.

## Initialize

```bash
cd deploy/personal
cp .env.example .env
cp .image.env.example .image.env
chmod 600 .env .image.env
```

Set a unique session secret and trusted HTTPS origin in `.env`. Replace the image in `.image.env` with an immutable full commit tag or digest. Before the first start, validate the configuration with `docker compose --env-file .image.env -f compose.yml config --quiet`.

## Back Up and Upgrade

```bash
./maintenance.sh backup
./maintenance.sh upgrade ghcr.io/buglyz/new-api:sha-<full-40-character-commit>
```

The backup uses SQLite's online `.backup` command and then runs `PRAGMA quick_check`. It also saves the Compose file, application environment, and current image. Backup directories use a unique UTC-prefixed name with mode `0700` and may contain session configuration, so never upload or commit them.

Upgrades only accept a digest or full commit SHA tag. If the health check fails, the script restores the previous image automatically; it does not roll back the database automatically.

## Roll Back

```bash
./maintenance.sh rollback backups/<UTC-timestamp>
./maintenance.sh rollback backups/<UTC-timestamp> --restore-database --restore-config
```

The first command rolls back only the image. The second stops the container and restores SQLite plus application configuration. Before either operation, the script creates another backup of the current state.

This template only handles the SQLite database in this directory. External MySQL, PostgreSQL, or separate log databases require their native consistent backup tools; copying their data files is not a database backup.
