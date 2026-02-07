# Reassembling Split Files

Some files were split into 95MB chunks due to GitHub's 100MB file size limit.

## Quick reassemble (run from repo root)

```bash
find . -name "*.split.aa" -o -name "*.part.aa" | while read first; do
  base="${first%%.aa}"
  original="${base%%.split}"
  [ "$original" = "$base" ] && original="${base%%.part}"
  echo "Reassembling: $original"
  cat "${base}."* > "$original"
  rm "${base}."*
done
```

## Split files

- `speckle/db-dump/speckle_db.sql.part.*` -> `speckle_db.sql` (~1 GB)
- `speckle/db-dump/speckle_db.dump.part.*` -> `speckle_db.dump` (~152 MB)
- `valvx/minio-data/valvx/019a5da3-.../part.1.split.*` -> `part.1` (~143 MB)

## Restoring databases

```bash
# ValvX (PostgreSQL 17)
pg_restore -U valvx -d valvx valvx/db-dump/valvx_db.dump
# or
psql -U valvx -d valvx -f valvx/db-dump/valvx_db.sql

# Speckle (PostgreSQL 16)
pg_restore -U speckle -d speckle speckle/db-dump/speckle_db.dump
# or
psql -U speckle -d speckle -f speckle/db-dump/speckle_db.sql
```
