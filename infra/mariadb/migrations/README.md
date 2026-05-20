# Database Migrations

This directory contains SQL migration scripts for the CampusVision AI MariaDB schema.

## Execution

Migrations are not managed by Flyway or any migration framework. Apply manually:

```bash
# Using pipe (recommended)
docker compose exec -T mariadb mysql -uroot -proot dormitory < infra/mariadb/migrations/NNN_name.sql

# Or using source (requires copying file into container first)
docker compose cp infra/mariadb/migrations/NNN_name.sql mariadb:/tmp/
docker compose exec mariadb mysql -uroot -proot dormitory -e "SOURCE /tmp/NNN_name.sql"
```

## Naming Convention

```
NNN_description.sql
```

- `NNN` — 3-digit sequential number (001, 002, ...)
- `description` — short kebab-case description of the change

## Guidelines

1. **Order**: Apply migrations in sequential order (001, 002, 003...).
2. **Idempotency**: Use `IF NOT EXISTS` / `IF EXISTS` where possible.
3. **Backward compatibility**: Do not drop or rename columns that existing code relies on.
4. **Verification**: After applying, run `DESC <table>` to confirm columns exist.
5. **init.sql is the source of truth**: Schema definitions in `infra/mariadb/init.sql`
   must be kept in sync with migrations. When adding a new migration, update init.sql
   to reflect the final schema state for fresh deployments.

## Applied Migrations

| # | File | Applied | Description |
|---|---|---|---|
| 001 | `001_camera_platform.sql` | 2026-05-18 | Add camera platform expansion columns (type, protocol, host, port, path, username, password_enc, nonce, key_id, last_health_check) |
| 002 | `V002__face_embedding.sql` | 2026-05-20 | Create face_embedding table for face recognition pipeline (512-dim embeddings, student_id lookup) |
