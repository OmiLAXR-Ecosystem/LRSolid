# Bruno Collection: LRSolid

## Setup

1. Open collection in Bruno (`bruno/LRSolid`).
2. Select environment: `local`.
3. Set `email` and `password` in `environments/local.bru`.
4. Run `Auth/Login`.
5. Copy `Set-Cookie` response value into `sessionCookie` (format: `lrsolid_session=...`).
6. Run protected requests (`Auth/Me`, `xAPI/*`, `Admin/*`).

## CLI (bru)

Example runs:

```bash
bru run bruno/LRSolid/Auth/login.bru --env local
bru run bruno/LRSolid/Auth/me.bru --env local
bru run bruno/LRSolid/xAPI/post-statement.bru --env local
bru run bruno/LRSolid/xAPI/get-statements.bru --env local
```

If you use the local Vite frontend proxy instead of API port directly, set `baseUrl` to `http://localhost:5173`.
