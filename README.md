# Seruwit WA Engine (Multi-Tenant WhatsApp Gateway)

Engine WhatsApp Gateway **headless multi-tenant** berkecepatan tinggi berbasis **Golang (whatsmeow)** yang dirancang khusus sebagai penyedia notifikasi dan pesan untuk **seruwit-crm**.

---

## 🚀 Keunggulan Arsitektur

- **Ultra-Lightweight & Scalable**: Berbasis *pure-protocol WebSocket* (tanpa Chromium/Puppeteer). Sanggup menangani **ratusan hingga ribuan tenant** aktif bersamaan pada 1 server VPS.
- **Dual Database Support**: 
  - **SQLite** (bawaan, zero-config untuk dev lokal).
  - **PostgreSQL** (untuk performa tinggi di production).
- **Persistent Sessions**: Sesi nomor WhatsApp disimpan rapi di database. Jika server reboot atau restart, semua sesi tenant otomatis terhubung kembali (*auto-reconnect*) tanpa perlu scan QR ulang.
- **QR Code & Pairing Code**: Mendukung pairing scan QR (format base64 PNG) dan 8-digit Pairing Code untuk kemudahan tenant di HP.
- **Secure REST API**: Dilindungi autentikasi header `X-API-Key`.
- **Asynchronous Webhook**: Mengirim callback event real-time ke `seruwit-crm` tanpa memblokir socket WhatsApp.

---

## 📁 Struktur Project

```
seruwit-wa/
├── cmd/
│   └── server/
│       └── main.go          # Entry point aplikasi
├── internal/
│   ├── api/                 # Fiber HTTP router, middleware, handlers
│   ├── config/              # Environment loader
│   ├── database/            # Abstraksi database (SQLite & PostgreSQL)
│   ├── webhook/             # Asynchronous webhook dispatcher
│   └── whatsapp/            # whatsmeow Session Manager & Event Handlers
├── data/                    # Folder database lokal SQLite
├── bin/                     # Binary build output
├── .env.example
├── .gitignore
├── go.mod
└── go.sum
```

---

## ⚙️ Konfigurasi Environment (`.env`)

Salin file `.env.example` ke `.env`:

```bash
cp .env.example .env
```

| Variabel | Default | Keterangan |
| :--- | :--- | :--- |
| `PORT` | `8080` | Port HTTP REST API |
| `API_KEY` | `seruwit-secret-key-change-me` | Secret key untuk validasi request dari CRM |
| `DB_DIALECT` | `sqlite` | Pilihan: `sqlite` atau `postgres` |
| `DB_ADDRESS` | `file:data/seruwit-wa.db?_foreign_keys=on` | DSN koneksi database |
| `CRM_WEBHOOK_URL` | `http://localhost:8000/api/webhook/whatsapp` | Endpoint webhook di seruwit-crm |
| `CRM_WEBHOOK_SECRET` | `seruwit-crm-secret` | Header `X-Webhook-Secret` saat kirim webhook |
| `LOG_LEVEL` | `INFO` | Level log: `DEBUG`, `INFO`, `WARN`, `ERROR` |

---

## 🛠️ Menjalankan Aplikasi

### Mode Development:
```bash
go run ./cmd/server
```

### Build Binary:
```bash
go build -o bin/seruwit-wa ./cmd/server
./bin/seruwit-wa
```

---

## 📮 Postman Collection

File Postman Collection v2.1 sudah disediakan di root project:
* File: [`postman_collection.json`](file:///Users/meyga/DEV/LARAVEL/seruwit-wa/postman_collection.json)

**Cara Import ke Postman:**
1. Buka aplikasi Postman.
2. Klik tombol **Import** di kiri atas.
3. Pilih atau drag-and-drop file [`postman_collection.json`](file:///Users/meyga/DEV/LARAVEL/seruwit-wa/postman_collection.json).
4. Collection **"Seruwit WA Gateway API"** akan otomatis terkonfigurasi dengan variabel:
   - `{{base_url}}`: `http://localhost:8080`
   - `{{api_key}}`: `seruwit-secret-key-change-me`
   - `{{session_id}}`: `tenant-001`

---

## 📡 Dokumentasi REST API

Semua request (kecuali `/health`) wajib menyertakan header:
```
X-API-Key: seruwit-secret-key-change-me
Content-Type: application/json
```

### 1. Health Check
- **URL**: `GET /health`
- **Response**:
```json
{
  "status": "ok",
  "time": 1792279810
}
```

---

### 2. Inisialisasi Sesi & Generate QR Code
- **URL**: `POST /api/sessions/start`
- **Request Body**:
```json
{
  "session_id": "tenant-001"
}
```
- **Response**:
```json
{
  "success": true,
  "session_id": "tenant-001",
  "status": "SCAN_QR",
  "qr_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAA..."
}
```
*(CRM tinggal menampilkan string `qr_base64` di tag `<img src="...">`)*

---

### 3. Pairing Menggunakan Kode 8-Digit (Alternatif QR)
- **URL**: `POST /api/sessions/pair-code`
- **Request Body**:
```json
{
  "session_id": "tenant-001",
  "phone": "628123456789"
}
```
- **Response**:
```json
{
  "success": true,
  "session_id": "tenant-001",
  "pairing_code": "ABCD-1234"
}
```

---

### 4. Cek Status Sesi
- **URL**: `GET /api/sessions/:id/status`
- **Response**:
```json
{
  "success": true,
  "session_id": "tenant-001",
  "status": "CONNECTED",
  "phone": "628123456789",
  "push_name": "Seruwit Admin",
  "qr_base64": ""
}
```

---

### 5. Logout & Putuskan Sesi
- **URL**: `POST /api/sessions/:id/logout`
- **Response**:
```json
{
  "success": true,
  "message": "Session logged out and removed successfully"
}
```

---

### 6. Kirim Pesan Teks (Notifikasi)
- **URL**: `POST /api/messages/send-text`
- **Request Body**:
```json
{
  "session_id": "tenant-001",
  "to": "6281234567890",
  "message": "Halo Budi, tagihan invoice #INV-2026-001 sebesar Rp 150.000 telah terbit."
}
```
- **Response**:
```json
{
  "success": true,
  "message_id": "3EB0ABCD1234567890",
  "to": "6281234567890"
}
```

---

## 🔔 Format Webhook ke `seruwit-crm`

Setiap event penting akan dikirimkan otomatis ke `CRM_WEBHOOK_URL` via method `POST`:

### Event Status Koneksi (`session.status`):
```json
{
  "event": "session.status",
  "session_id": "tenant-001",
  "timestamp": 1792279810,
  "data": {
    "status": "CONNECTED",
    "phone": "628123456789",
    "push_name": "Seruwit Admin",
    "jid": "628123456789@s.whatsapp.net"
  }
}
```

### Event Status Pengiriman Pesan (`message.receipt`):
```json
{
  "event": "message.receipt",
  "session_id": "tenant-001",
  "timestamp": 1792279810,
  "data": {
    "message_ids": ["3EB0ABCD1234567890"],
    "receipt": "read"
  }
}
```
*(receipt bernilai: `delivered` atau `read`)*

### Event Pesan Masuk / Balasan Tenant (`message.incoming`):
```json
{
  "event": "message.incoming",
  "session_id": "tenant-001",
  "timestamp": 1792279810,
  "data": {
    "message_id": "3EB0987654321",
    "sender": "6281234567890",
    "text": "Terima kasih, tagihan sudah saya bayar.",
    "push_name": "Budi Santoso",
    "timestamp": 1792279810
  }
}
```

---

## 💡 Contoh Implementasi di Laravel (`seruwit-crm`)

### Kirim Pesan via Laravel HTTP Client:
```php
use Illuminate\Support\Facades\Http;

$response = Http::withHeaders([
    'X-API-Key' => config('services.seruwit_wa.api_key'),
])->post(config('services.seruwit_wa.base_url') . '/api/messages/send-text', [
    'session_id' => 'tenant_' . $tenant->id,
    'to'         => $customer->phone,
    'message'    => 'Halo ' . $customer->name . ', ini notifikasi dari CRM.',
]);

if ($response->successful()) {
    $messageId = $response->json('message_id');
}
```
