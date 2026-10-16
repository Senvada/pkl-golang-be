#!/usr/bin/env bash
set -e

echo "🚀 [PKL Bootstrap] Memeriksa & menginisialisasi Database & Storage lokal..."

DB_CONTAINER=""
if docker ps --format '{{.Names}}' | grep -q "^pkl-local-postgres$"; then
    DB_CONTAINER="pkl-local-postgres"
elif docker ps --format '{{.Names}}' | grep -q "^supabase_db"; then
    DB_CONTAINER=$(docker ps --format '{{.Names}}' | grep "^supabase_db" | head -n 1)
fi

if [ -z "$DB_CONTAINER" ]; then
    echo "⚠️  Container Postgres lokal tidak terdeteksi. Menjalankan via docker-compose..."
    docker compose up -d pkl-db pkl-storage
    echo "⏳ Menunggu database siap..."
    sleep 3
    DB_CONTAINER="pkl-local-postgres"
fi

echo "✅ Menggunakan Postgres Container: $DB_CONTAINER"

# 1. Pastikan database pkl_development ada
echo "📦 Memeriksa database pkl_development..."
DB_EXISTS=$(docker exec -i "$DB_CONTAINER" psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='pkl_development';")
if [ "$DB_EXISTS" != "1" ]; then
    echo "✨ Membuat database pkl_development..."
    docker exec -i "$DB_CONTAINER" psql -U postgres -c "CREATE DATABASE \"pkl_development\";"
    docker exec -i "$DB_CONTAINER" psql -U postgres -d "pkl_development" -c "GRANT ALL ON SCHEMA public TO postgres;" 2>/dev/null || true
else
    echo "✅ Database pkl_development sudah ada."
    docker exec -i "$DB_CONTAINER" psql -U postgres -d "pkl_development" -c "GRANT ALL ON SCHEMA public TO postgres;" 2>/dev/null || true
fi

# 2. Pastikan bucket pkl-assets ada di Supabase Storage
echo "🪣 Memeriksa storage bucket pkl-assets..."
BUCKET_EXISTS=$(docker exec -i "$DB_CONTAINER" psql -U postgres -d postgres -tAc "SELECT 1 FROM storage.buckets WHERE id='pkl-assets';" 2>/dev/null || echo "0")
if [ "$BUCKET_EXISTS" != "1" ]; then
    echo "✨ Mendaftarkan bucket pkl-assets (public)..."
    docker exec -i "$DB_CONTAINER" psql -U postgres -d postgres -c "INSERT INTO storage.buckets (id, name, public) VALUES ('pkl-assets', 'pkl-assets', true) ON CONFLICT DO NOTHING;"
else
    echo "✅ Bucket pkl-assets sudah siap."
fi

echo "🎉 [PKL Bootstrap] Selesai! Database & Storage siap digunakan."
