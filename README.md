# Cekping Agent

Agen pemantauan jaringan ringan (Lightweight Network Monitoring Agent) untuk menjalankan instruksi Ping & MTR.

## Persyaratan Sistem
- **Sistem Operasi**: Linux, FreeBSD (Server/Desktop) atau macOS.
- **Jaringan**: Koneksi internet yang stabil.
- **Hak Akses**: Akses **Root / Sudo** wajib diperlukan karena agen menggunakan *Raw Socket* untuk melakukan eksekusi ICMP Ping yang akurat.

## Langkah Instalasi

### 1. Dapatkan Token
1. Daftar atau masuk ke web dashboard Cekping.
2. Buka menu **Volunteer**.
3. Klik tombol "Register New Probe" lalu simpan `AGENT_TOKEN` yang diberikan.

### 2. Jalankan Agent

#### A. Menggunakan Docker (Disarankan)
Jika menggunakan Docker, sangat disarankan menggunakan mode `--network host` agar agen dapat membaca network host dengan tepat:

```bash
docker run -d \
  --name cekping-agent \
  --network host \
  --pull always \
  --restart always \
  --env CEKPING_SERVER=cekping.id:50051 \
  --env CEKPING_TOKEN="TOKEN_ASLI_ANDA_DISINI" \
  --env CEKPING_SECURE=true \
  ghcr.io/awandataindonesia/cekping-agent:latest
```

#### B. Menggunakan Podman
Mirip dengan environment Docker, namun khusus untuk ekosistem Podman (terutama instalasi *rootless*), Anda wajib menambahkan flag kapabilitas `NET_RAW` agar Go Ping diizinikan oleh OS untuk membuat *socket* ICMP:

```bash
podman run -d \
  --name cekping-agent \
  --network host \
  --pull always \
  --cap-add=NET_RAW \
  --env CEKPING_SERVER=cekping.id:50051 \
  --env CEKPING_TOKEN="TOKEN_ANDA" \
  --env CEKPING_SECURE=true \
  ghcr.io/awandataindonesia/cekping-agent:latest
```

#### C. Menggunakan Install Script (Linux Native)
Jika tidak ingin menggunakan container, Anda dapat menggunakan script instalasi otomatis yang akan mengunduh binary dan menjadikannya sebagai *Systemd Service*:

```bash
curl -sL https://raw.githubusercontent.com/awandataindonesia/cekping-agent/main/scripts/install.sh | sudo bash -s -- \
  -t "TOKEN_ANDA" \
  -s "cekping.id:50051" -S 
```

Script ini secara otomatis akan mengatur *environment variables* yang dibutuhkan dan menjalankan agen di background.
