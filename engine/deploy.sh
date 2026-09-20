#!/usr/bin/env bash
#
# Script deploy resmi. Agent DevOps hanya boleh memanggil file ini apa adanya,
# tanpa argumen dan tanpa mengubah isinya. Semua perintah yang menyentuh server
# production ditulis dan direview manusia di sini.
#
# Sebelum dipakai: isi variabel di bawah, hapus baris SQUAD_DEPLOY_UNCONFIGURED,
# lalu jalankan manual sekali untuk memastikan benar.

set -euo pipefail

# ---- HAPUS BLOK INI SETELAH SCRIPT DIISI ------------------------------------
echo "deploy.sh masih template. Isi dulu konfigurasinya, lalu hapus blok penjaga ini." >&2
exit 78
# -----------------------------------------------------------------------------

DEPLOY_HOST="${DEPLOY_HOST:?DEPLOY_HOST belum diset}"   # contoh: deploy@203.0.113.10
DEPLOY_PATH="${DEPLOY_PATH:?DEPLOY_PATH belum diset}"   # contoh: /srv/app
SERVICE_NAME="${SERVICE_NAME:-app}"
HEALTH_URL="${HEALTH_URL:?HEALTH_URL belum diset}"      # contoh: https://app.example.com/health

echo "==> Kirim kode ke server"
rsync -az --delete \
  --exclude '.git' --exclude '__pycache__' --exclude '.venv' \
  ./ "${DEPLOY_HOST}:${DEPLOY_PATH}/"

echo "==> Restart service"
# Pakai user deploy dengan sudo terbatas, bukan root.
ssh "${DEPLOY_HOST}" "cd ${DEPLOY_PATH} && sudo systemctl restart ${SERVICE_NAME}"

echo "==> Health check"
for attempt in 1 2 3 4 5; do
  if curl -fsS --max-time 10 "${HEALTH_URL}" > /dev/null; then
    echo "Health check lolos pada percobaan ${attempt}."
    exit 0
  fi
  echo "Health check gagal, percobaan ${attempt}. Menunggu 5 detik."
  sleep 5
done

echo "Health check gagal terus. Deploy dianggap GAGAL." >&2
echo "Rollback: ssh ${DEPLOY_HOST} dan kembalikan rilis sebelumnya." >&2
exit 1
