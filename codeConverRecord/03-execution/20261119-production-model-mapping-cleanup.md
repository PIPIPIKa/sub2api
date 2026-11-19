# Production model mapping cleanup

## Target

BWH production server `144.34.139.244:1415`, Sub2API accounts 39 `KONG-GOOGLE` and 40 `GAO-GOOGLE`.

## Backup

`/root/sub2api-backups/accounts-model-mapping-20261119T085116Z.sql`

## Data change

- Removed `gemini-4-hydrogen-high`.
- Removed `gemini-4-hydrogen-low`.
- Removed `gemini-4-hydrogen-medium`.
- Removed `gemini-4-hydrogen-xhigh`.
- Removed Carbon tier keys and added `gemini-4-carbon` self-mapping.
- Preserved Argon mappings.

## Verification

Both accounts now have the Carbon base mapping, no Carbon tier mappings, and no Hydrogen tier mappings. The `sub2api` container was restarted and reported `running healthy`. `curl http://127.0.0.1:8080/health` returned `{"status":"ok"}`.
