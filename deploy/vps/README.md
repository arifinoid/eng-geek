# Deploy eng-geek ke VPS (NixOS, tanpa Docker)

Host: `enggeek.arifinoid.web.id` → VPS NixOS (`root@43.173.26.172`).
Konfigurasi host ada di repo [nixos](https://github.com/arifinoid/nixos):
`nix/modules/nixos/vps-enggeek.nix`.

## Bentuk deployment

```
deploy/vps/deploy.sh (dijalankan dari checkout ini)
  ├─ bun run build:web                       → apps/web/out (static export)
  ├─ go build ./cmd/server + ./cmd/seed      → binary statis (CGO_ENABLED=0)
  ├─ stage: server, seed, out/, apps/api/db/migrations, apps/api/content/a1
  ├─ rsync → /srv/enggeek/releases/<sha-waktu>/
  ├─ symlink /srv/enggeek/current → release baru (atomik, mv -Tf)
  ├─ systemctl start eng-geek-migrate        (goose up + seed, sebagai eng-geek)
  ├─ systemctl start|restart eng-geek
  └─ health check 127.0.0.1:8083/healthz, lalu https://enggeek.arifinoid.web.id/healthz
     gagal? symlink dikembalikan ke release sebelumnya
```

Host:

```
nginx :443 ── /            → /srv/enggeek/current/out  (tryFiles …/index.html)
             ├─ /api/v1/*  → 127.0.0.1:8083
             ├─ /healthz   → 127.0.0.1:8083
             └─ = /        → 307 /id
```

## Kenapa Postgres lokal

Aplikasi lain di VPS memakai Neon, tapi eng-geek masih satu rumah tangga dan
satu database kecil. Menjalankan Postgres 17 lokal menghindari dependensi luar
dan biaya, dan peer auth over Unix socket berarti tidak ada password yang
berserakan. Database `eng-geek` dimiliki oleh role `eng-geek` (nama sama,
diatur `ensureDBOwnership`), jadi migrasi bisa membuat schema sendiri.

> Backup: belum ada timer `pg_dump`. Tambahkan nanti (mis. mengikuti
> `deploy/vps/backup.sh` di repo nata) sebelum diisi data nyata.

## Persiapan sekali

1. DNS A record `enggeek.arifinoid.web.id` → IP VPS.
2. Aktifkan modul NixOS:

   ```bash
   nixos-rebuild switch --flake ~/Documents/repos/nixos#vps \
     --target-host root@43.173.26.172
   ```

   Ini membuat user/group `eng-geek`, Postgres 17 + database `eng-geek`,
   unit `eng-geek` dan `eng-geek-migrate`, vhost nginx, dan sertifikat ACME.
   Service belum menyala sampai release pertama ada
   (`ConditionPathExists = /srv/enggeek/current/server`).

## Deploy

```bash
bash deploy/vps/deploy.sh
```

Verifikasi:

```bash
curl -fsS https://enggeek.arifinoid.web.id/healthz
curl -fsS https://enggeek.arifinoid.web.id/api/v1/profiles
```

Rollback manual:

```bash
ssh root@43.173.26.172 '
  ln -sfn /srv/enggeek/releases/<sha-lama> /srv/enggeek/current.new
  mv -Tf /srv/enggeek/current.new /srv/enggeek/current
  systemctl restart eng-geek
'
```

Tiga release terakhir disimpan di host; sisanya dipangkas setelah deploy sukses.

## Catatan

- Binary Go diganti dari WSL; `CGO_ENABLED=0` membuatnya statis, jadi tidak
  masalah glibc VPS beda.
- `bun run build:web` memakai base URL `/api/v1` (default orval), same-origin
  lewat nginx.
- Repository working tree harus bersih saat deploy: SHA judul release adalah
  `git rev-parse --short HEAD`.
