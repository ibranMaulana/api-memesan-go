# Memesan API Go - Production Bundle

Bundle runtime Go native untuk API Memesan. Folder ini sengaja tidak menyertakan source PHP legacy, dump database, log request, credential, atau artifact build.

## Requirements

- Linux server
- Docker Engine
- Docker Compose plugin
- MySQL yang mengikuti schema aplikasi lama

## Install

```bash
cp .env.example .env
chmod 600 .env
nano .env

docker compose up -d --build
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/readyz
```

Required `.env` values:

```env
APP_PORT=8080
TZ=Asia/Jakarta
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=memesan
DB_USER=memesan_api
DB_PASSWORD=change-this
```

Jika MySQL berjalan di host yang sama dan API berjalan dalam container, gunakan IP gateway Docker atau nama service database yang sesuai. Jangan commit `.env`.

## Migration

Migration tambahan tersedia di `migrations/`. Backup database sebelum menjalankannya. Jalankan hanya migration yang belum pernah dijalankan:

```bash
mysql -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" -p "$DB_NAME" \
  < migrations/20261005_add_submenu_transaction_snapshot.sql
```

## Operations

```bash
docker compose ps
docker compose logs --tail=200 -f api
docker compose restart api
docker compose up -d --build
docker compose down
```

Health endpoints:

- `/healthz`: process hidup
- `/readyz`: process dan koneksi MySQL siap

Canonical routes tidak memakai `.php`; alias `.php` tetap tersedia untuk kompatibilitas dan diproses oleh Go.

## Validation before traffic switch

```bash
go test ./...
go vet ./...
docker build -t memesan-api-react:dev .
```

Lakukan backup database dan smoke test read/write di staging sebelum mengarahkan aplikasi mobile ke service ini.
