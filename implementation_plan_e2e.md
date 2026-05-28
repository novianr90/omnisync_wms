# Rencana Implementasi: Otomatisasi Pengujian End-to-End (E2E) dengan Playwright

## Overview
Pengujian unit telah menjamin integritas logika backend, namun belum dapat mendeteksi regresi pada UI seperti layout HTML yang rusak, kegagalan swap HTMX, modal kustom yang tidak responsif, kesalahan skrip JavaScript, atau tabrakan styling CSS. 

Dokumen ini memetakan rencana untuk mengimplementasikan **Automated End-to-End (E2E) Testing Suite** yang tangguh menggunakan **Playwright** di bawah direktori `wms_dashboard/e2e/` untuk memvalidasi 6 alur operasional utama Omnisync WMS.

---

## Architecture Decisions & Infrastructure

- **Tooling**: Playwright dengan Node.js (JavaScript). Playwright dipilih karena kecepatan eksekusi, auto-waiting locators yang sangat cocok untuk siklus swap HTMX asinkron, dan dukungan visual debugging (Screenshots & Traces).
- **Auto Server Provisioning**: Konfigurasi Playwright akan menggunakan array `webServer` untuk memicu secara otomatis baik `auth_services` maupun `wms_dashboard` sebelum pengujian dimulai, sehingga pengujian bersifat mandiri (*fully self-contained*) dan dapat dieksekusi dengan satu perintah tunggal: `npx playwright test`.
- **Database State**: Karena pengujian berjalan secara live terhadap server dev lokal, pengujian akan memanfaatkan kredensial default yang telah di-seed (`admin@omnisync.com` / `admin123` dan `operator@omnisync.com` / `operator123`).

---

## Task List

### Phase 1: Foundation & Infrastructure Setup

#### Task 1: Playwright Installation & Configuration
- **Description:** Menginisialisasi dependensi Node.js Playwright di dalam `wms_dashboard/` dan mengonfigurasi `playwright.config.js` agar secara otomatis menjalankan server `auth_services` (:8000) dan `wms_dashboard` (:9901) sebelum pengujian dimulai.
- **Acceptance Criteria:**
  - File `package.json` memiliki dependensi `@playwright/test`.
  - Konfigurasi `playwright.config.js` mencakup pengaturan multi-webServer, resolusi port, headless browser (Chromium), serta penangkapan screenshots & traces saat kegagalan uji (*on-failure*).
- **Verification:**
  - Pengujian infrastruktur dasar lulus: `npx playwright test --help`
  - Konfigurasi divalidasi tidak mengalami konflik port.
- **Estimated Scope:** Small (1-2 files)
- **Files touched:**
  - `wms_dashboard/package.json` (update dependencies)
  - `wms_dashboard/playwright.config.js` (new configuration)

#### Task 2: Auth Utility Fixtures
- **Description:** Membuat berkas pembantu (*helper/fixture*) otentikasi untuk menyimpan dan memuat ulang status sesi login (`auth.json`), guna menghindari pengulangan proses login di awal setiap file uji.
- **Acceptance Criteria:**
  - Adanya fungsi pembantu untuk login sebagai Admin (`admin@omnisync.com`) dan Operator (`operator@omnisync.com`).
  - Sesi disimpan secara aman sebagai storage state cookie Playwright.
- **Verification:**
  - File sesi berhasil digenerasikan saat login pertama kali.
- **Estimated Scope:** Small (1-2 files)
- **Files touched:**
  - `wms_dashboard/e2e/helpers/auth.js`

### Checkpoint: Foundation Setup
- [ ] Dependensi Playwright terinstal tanpa error.
- [ ] Server auth & dashboard dapat dihidupkan secara otomatis oleh Playwright.

---

### Phase 2: Core E2E Journeys Implementation

#### Task 3: Authentication & Navigation Flows
- **Description:** Menguji alur login negatif (kredensial salah menampilkan pesan error toast) dan positif (kredensial benar meredirect ke dashboard `/`), serta navigasi menu sidebar menggunakan swap asinkron HTMX.
- **Acceptance Criteria:**
  - Tes memvalidasi munculnya toast alert merah "Invalid credentials" saat salah password.
  - Tes memvalidasi transisi URL ke `/` saat login berhasil.
  - Tes mengklik menu Master Data (Products, UoMs, Warehouses) dan memastikan konten bertukar secara dinamis tanpa full page reload.
- **Verification:**
  - Jalankan pengujian: `npx playwright test e2e/auth_nav.spec.js`
- **Estimated Scope:** Small (1-2 files)
- **Files touched:**
  - `wms_dashboard/e2e/auth_nav.spec.js`

#### Task 4: Inbound Receipt & Outbound Dispatch FIFO Flows
- **Description:** Menguji siklus masuk dan keluar barang secara penuh. Memverifikasi barang masuk terdaftar sebagai batch lot baru, dan barang keluar memotong stok berdasarkan batch tertua terlebih dahulu (FIFO).
- **Acceptance Criteria:**
  - **Inbound**: Mengklik "Register Inbound", mengisi form modal, mengirimkan data, menuntut tugas (claim), menyetujui jurnal, dan memverifikasi kuantitas katalog produk bertambah secara dinamis.
  - **Outbound**: Mengklik "Dispatch Outbound", mencoba mengeluarkan stok melebihi batas (memastikan muncul error), memasukkan jumlah valid, memverifikasi status, dan memastikan stok berkurang sesuai FIFO lot.
- **Verification:**
  - Jalankan pengujian: `npx playwright test e2e/inventory_movements.spec.js`
- **Estimated Scope:** Medium (2-3 files)
- **Files touched:**
  - `wms_dashboard/e2e/inventory_movements.spec.js`

#### Task 5: QC Hold, Quarantine & Stock Adjustments
- **Description:** Menguji pembekuan stok di bawah QC Hold, verifikasi pengurangan stok tersedia pada Dashboard, pelepasan hold via modal konfirmasi, serta pengujian penyesuaian stok positif & negatif secara real time.
- **Acceptance Criteria:**
  - Menghasikan QC Hold untuk kuantitas tertentu, memastikan stok tersedia berkurang.
  - Melepas hold via modal HTML, memastikan status menjadi `RELEASED` dan stok pulih.
  - Mengirim tiket stock adjustment (FOUND & LOST) dan memverifikasi kuantitas di katalog terupdate seketika.
- **Verification:**
  - Jalankan pengujian: `npx playwright test e2e/qc_hold_adjustments.spec.js`
- **Estimated Scope:** Medium (2-3 files)
- **Files touched:**
  - `wms_dashboard/e2e/qc_hold_adjustments.spec.js`

#### Task 6: Kitting / Light Assembly Flow
- **Description:** Menguji proses perakitan barang (kitting). Memastikan dropdown lokasi hanya menampilkan locator dengan stok komponen yang tersedia, melakukan assembly, dan memverifikasi pemotongan bahan baku serta penambahan stok barang jadi bundle.
- **Acceptance Criteria:**
  - Membuat tiket kitting order baru untuk produk bundle.
  - Mengirimkan form dan menjurnal kitting.
  - Memverifikasi bahan baku berkurang secara FIFO dan bundle bertambah di locator target.
- **Verification:**
  - Jalankan pengujian: `npx playwright test e2e/kitting.spec.js`
- **Estimated Scope:** Small (1-2 files)
- **Files touched:**
  - `wms_dashboard/e2e/kitting.spec.js`

### Checkpoint: Core Features complete
- [ ] Seluruh 6 perjalanan pengguna (Authentication, Navigation, Inbound, Outbound FIFO, QC Holds, Adjustments, Kitting) lulus uji E2E di Playwright.
- [ ] Screenshot & Traces tersimpan rapi jika ada langkah yang gagal.

---

### Phase 3: Add Unit Testing & CI Integration

#### Task 7: Add Unit Testing
- **Description:** Menambahkan pengujian unit baru khusus untuk memverifikasi fungsionalitas visual/render template HTML, routing parsial Fiber, dan data parsing form di tingkat handler master data.
- **Acceptance Criteria:**
  - Membuat berkas test unit baru `wms_dashboard/internal/handlers/master_handler_test.go` untuk menguji *view rendering* (seperti `ServeProductsMaster`, `ServeNewProductForm`).
  - Memastikan seluruh pengujian unit backend tetap lulus (`go test ./...`).
- **Verification:**
  - Jalankan pengujian unit: `go test -v ./internal/handlers/...`
- **Estimated Scope:** Small (1-2 files)
- **Files touched:**
  - `wms_dashboard/internal/handlers/master_handler_test.go`

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
| :--- | :--- | :--- |
| **Port Conflicts** | High | Mengonfigurasi `playwright.config.js` dengan opsi `reuseExistingServer: true` untuk mendeteksi apakah dev server sudah berjalan secara lokal, menghindari crash benturan alokasi port. |
| **Flaky HTMX Waitings** | Medium | Menggunakan selector asertif Playwright yang secara native memiliki *auto-waiting* (misal: `page.locator('text=Registered successfully')`) alih-alih menggunakan static sleep/delay. |
| **Dirty Test Data DB** | Medium | Memanfaatkan SQLite db in-memory khusus untuk test unit, dan untuk E2E menggunakan server lokal. Pengujian E2E didesain independen dengan membuat data produk/lokasi unik baru di setiap sesinya. |

---

## Open Questions
- Apakah Playwright test runner perlu dipicu secara headless default pada pipeline GitHub Actions di masa mendatang? *(Direkomendasikan: Ya, headless browser sangat optimal untuk CI/CD)*.
