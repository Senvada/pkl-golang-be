-- ==============================================================================
-- SEEDER PENILAIAN PKL (SMK NEGERI 6 MALANG)
-- Diadaptasi persis dari Berkas Dokumen Resmi PDF di: /daftar-nilai
-- Kompatibel: psql, GORM db.Exec, dan migration tools (PL/pgSQL DO block)
-- ==============================================================================

DO $$
BEGIN

-- 1. Pastikan Periode Akademik Default 2025/2026 Tersedia
INSERT INTO academic_periods (id, name, is_active, created_at, updated_at)
VALUES (
    gen_random_uuid(),
    '2025/2026',
    true,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (name) WHERE deleted_at IS NULL DO NOTHING;

-- 2. Template Penilaian 10 Jurusan Sesuai Dokumen Resmi

-- ------------------------------------------------------------------------------
-- 1. RPL (Rekayasa Perangkat Lunak)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Rekayasa Perangkat Lunak',
    '[
        {"id": "rpl-w-1", "category": "Web develop", "title": "Memahami basic Pembuatan Website Sederhana (googlesite, wordpress.com)"},
        {"id": "rpl-w-2", "category": "Web develop", "title": "Mampu membuat website sederhana dengan wordpress (instalation wordpress.org, home, menu, dashboard, web builder, use plugin, domain & hosting)"},
        {"id": "rpl-w-3", "category": "Web develop", "title": "Mampu mengoptimalkan kinerja wordpress dari speed dan SEO (web Optimization dan SEO Optimization)"},
        {"id": "rpl-w-4", "category": "Fullstack develop", "title": "Memahami basic pembuatan aplikasi berbasis web dengan php (html, CSS, Javascript, PHP, MySQL)"},
        {"id": "rpl-w-5", "category": "Fullstack develop", "title": "Mampu merancang dan membuat aplikasi berbasis web dengan menggunakan php framework (Laravel/CI, Bootstrap, Mysql)"},
        {"id": "rpl-w-6", "category": "Fullstack develop", "title": "Modeling (Flowchart, Class Diagram, DFD, ERD/CDM-PDM)"},
        {"id": "rpl-w-7", "category": "Mobile develop", "title": "Mampu membuat aplikasi mobile sederhana berbasis hybrid (Responsive Frontend, Cordova, Ionic)"},
        {"id": "rpl-w-8", "category": "Mobile develop", "title": "Mampu membuat aplikasi mobile hybrid dengan ionic framework (Ionic, React + Typescript, Ionic capasitor + Plugin, Deployment, Git)"},
        {"id": "rpl-w-9", "category": "Graphics / UI-UX Designer", "title": "Memahami dasar-dasar dan unsur desain grafis (Bidang, Ruang, Typografi, Nirmana, Komposisi, Garis dan Prespektif)"},
        {"id": "rpl-w-10", "category": "Graphics / UI-UX Designer", "title": "Mampu menjalankan software desain grafis berbasis bitmap dan vektor (Adobe Photoshop, Adobe Illustrator, CorelDRAW)"},
        {"id": "rpl-w-11", "category": "Graphics / UI-UX Designer", "title": "Mampu membuat dan mengedit desain berbasis bitmap dan vektor (Mockup, Publishing, Web Design, Art Design)"},
        {"id": "rpl-w-12", "category": "Graphics / UI-UX Designer", "title": "Mampu memahami UI/UX dan usability (pengetahuan UI/UX dan Membuat rancang komponen dasar (Design System))"},
        {"id": "rpl-w-13", "category": "K3LH", "title": "K3LH (Kesehatan dan Keselamatan Kerja dan Lingkungan Hidup)"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'RPL' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 2. TITL (Teknik Instalasi Tenaga Listrik)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Instalasi Tenaga Listrik',
    '[
        {"id": "titl-w-1", "title": "Analisa trouble shooting peralatan rumah tangga yang menggunakan motor listrik"},
        {"id": "titl-w-2", "title": "Penanganan trouble shooting peralatan rumah tangga yang menggunakan motor listrik"},
        {"id": "titl-w-3", "title": "Pengujian peralatan rumah tangga yang menggunakan motor listrik"},
        {"id": "titl-w-4", "title": "Perencanaan gulung ulang mesin listrik (motor/generator/transformator)"},
        {"id": "titl-w-5", "title": "Menggulung ulang mesin listrik"},
        {"id": "titl-w-6", "title": "Pengujian mesin listrik setelah di gulung ulang"},
        {"id": "titl-w-7", "title": "Merencanakan pekerjaan pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
        {"id": "titl-w-8", "title": "Menganalisis kebutuhan material listrik pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
        {"id": "titl-w-9", "title": "Penanganan trouble shooting pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
        {"id": "titl-w-10", "title": "Pengujian instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik setelah pemasangan"},
        {"id": "titl-w-11", "title": "Menerapkan K3 pada manusia"},
        {"id": "titl-w-12", "title": "Menerapkan K3 pada peralatan"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'TITL' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 3. TAB (Teknik Alat Berat)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Alat Berat',
    '[
        {"id": "tab-w-1", "title": "Preventive Maintenance"},
        {"id": "tab-w-2", "title": "Periodic Service"},
        {"id": "tab-w-3", "title": "Corrective Maintenance"},
        {"id": "tab-w-4", "title": "Pekerjaan Dasar Engine"},
        {"id": "tab-w-5", "title": "Pekerjaan Dasar Hidrolik"},
        {"id": "tab-w-6", "title": "Pekerjaan Dasar Powertrain dan undercarriage"},
        {"id": "tab-w-7", "title": "Pekerjaan Dasar Kelistrikan dan Sistem Kontrol"},
        {"id": "tab-w-8", "title": "Manajemen K3LH"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'TAB' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 4. OTO (Teknik Otomotif / Teknik Ototronik)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Ototronik',
    '[
        {"id": "oto-w-1", "title": "Tune up engine EFI"},
        {"id": "oto-w-2", "title": "Kelistrikan lampu"},
        {"id": "oto-w-3", "title": "Power Window / Central Lock"},
        {"id": "oto-w-4", "title": "Sistem Pengapian"},
        {"id": "oto-w-5", "title": "Sistem Rem / ABS"},
        {"id": "oto-w-6", "title": "Sistem Pemindah Tenaga"},
        {"id": "oto-w-7", "title": "Sistem Kemudi / Power Steering"},
        {"id": "oto-w-8", "title": "Sistem Suspensi"},
        {"id": "oto-w-9", "title": "Kesehatan dan Keselamatan Kerja"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'OTO' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 5. TPM (Teknik Pemesinan)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Pemesinan',
    '[
        {"id": "tpm-w-1", "title": "Gambar Teknik Mesin"},
        {"id": "tpm-w-2", "title": "Pekerjaan Dasar Teknik Mesin"},
        {"id": "tpm-w-3", "title": "Dasar Perencanaan Teknik Mesin"},
        {"id": "tpm-w-4", "title": "Gambar Teknik Manufaktur"},
        {"id": "tpm-w-5", "title": "Teknik Pemesinan Bubut"},
        {"id": "tpm-w-6", "title": "Teknik Pemesinan Frais"},
        {"id": "tpm-w-7", "title": "Teknik Pemesinan Gerindra"},
        {"id": "tpm-w-8", "title": "Teknik Pemesinan CNC"},
        {"id": "tpm-w-9", "title": "Produk Kreatif dan Kewirausahaan"},
        {"id": "tpm-w-10", "title": "Sistem Manajemen Keselamatan dan Kesehatan Kerja"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'TPM' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 6. TKJ (Teknik Jaringan Komputer dan Telekomunikasi / TJKT)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Jaringan Komputer dan Telekomunikasi',
    '[
        {"id": "tkj-w-1", "title": "Etika Profesi dan Keselamatan Kerja"},
        {"id": "tkj-w-2", "title": "Dasar Pemrograman dan Algoritma"},
        {"id": "tkj-w-3", "title": "Infrastruktur Telekomunikasi dan Media Jaringan"},
        {"id": "tkj-w-4", "title": "Perencanaan dan Implementasi Jaringan"},
        {"id": "tkj-w-5", "title": "Administrasi Sistem dan Layanan Jaringan"},
        {"id": "tkj-w-6", "title": "Teknologi Jaringan Kabel dan Nirkabel"},
        {"id": "tkj-w-7", "title": "Internet Of Things (IoT)"},
        {"id": "tkj-w-8", "title": "Teknologi Layanan Jaringan"},
        {"id": "tkj-w-9", "title": "Marketing dan Presentasi Produk"}
    ]'::jsonb,
    '[
        {"id": "tkj-b-1", "title": "Soft Skills (Komunikasi efektif, kerjasama tim, problem solving, time manajement, dan adaptif)"},
        {"id": "tkj-b-2", "title": "Tanggung Jawab dan Integritas"},
        {"id": "tkj-b-3", "title": "Pengembangan Diri"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'TKJ' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 7. DPIB (Desain Pemodelan dan Informasi Bangunan)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Desain Pemodelan dan Informasi Bangunan',
    '[
        {"id": "dpib-w-1", "title": "Melaksanakan Prosedur Keselamatan dan Kesehatan Kerja (K3)"},
        {"id": "dpib-w-2", "title": "Menggunakan Perintah Dasar Gambar, Melakukan Modifikasi Gambar, Menggunakan Fasilitas Pendukung Gambar dan Mencetak Hasil Gambar Sesuai Prosedur/Ketentuan"},
        {"id": "dpib-w-3", "title": "Menyajikan Gambar Proyeksi Bangunan Sesuai Kaidah Gambar Teknik (Gambar Situasi Denah, Potongan dan Tampak)"},
        {"id": "dpib-w-4", "title": "Menyajikan Gambar Konstruksi Sesuai Kaidah Gambar Teknik (Pondasi, Lantai, Dinding, Kusen, Daun Pintu, Jendela Ventilasi Beton Bertulang, Tangga, Atap, Langit-langit, Instalasi Listrik, Instalasi plumbing dan Drainase)"},
        {"id": "dpib-w-5", "title": "Menyusun Rencana Anggaran Terkait Dengan Tugas Perencanaan"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'DPIB' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- ------------------------------------------------------------------------------
-- 8. TKR (Teknik Kendaraan Ringan Otomotif / TKRO)
-- ------------------------------------------------------------------------------
INSERT INTO assessment_templates (
    id,
    department_id,
    period_id,
    title,
    work_aspects,
    behavior_aspects,
    created_at,
    updated_at
)
SELECT
    gen_random_uuid(),
    d.id,
    p.id,
    'Daftar Penilaian Praktik Kerja Lapangan - Teknik Kendaraan Ringan Otomotif',
    '[
        {"id": "tkr-w-1", "title": "Tune Up / Perawatan Berkala Mobil"},
        {"id": "tkr-w-2", "title": "Perawatan Sistem Utama Mesin (Mekanisme Katup, Pelumasan, Pendinginan)"},
        {"id": "tkr-w-3", "title": "Perawatan Sistem Bahan Bakar Bensin dan Diesel"},
        {"id": "tkr-w-4", "title": "Perawatan Sistem Starter"},
        {"id": "tkr-w-5", "title": "Perawatan Sistem Pengisian"},
        {"id": "tkr-w-6", "title": "Perawatan Sistem Pengapian (Konvensional dan Elektronik)"},
        {"id": "tkr-w-7", "title": "Perawatan Sistem Penerangan dan Panel Instrumen"},
        {"id": "tkr-w-8", "title": "Perawatan Aksesoris (Sistem AC, Sistem Audio, Power Window dan Central Lock)"},
        {"id": "tkr-w-9", "title": "Perawatan Kopling dan Transmisi Manual/Otomatis"},
        {"id": "tkr-w-10", "title": "Perawatan Poros Propeller dan Axle"},
        {"id": "tkr-w-11", "title": "Perawatan Gardan"},
        {"id": "tkr-w-12", "title": "Perawatan Sistem Rem (Konvensional dan ABS)"},
        {"id": "tkr-w-13", "title": "Perawatan Sistem Suspensi"},
        {"id": "tkr-w-14", "title": "Perawatan Sistem Kemudi dan Power Steering"},
        {"id": "tkr-w-15", "title": "Sistem Manajemen Kesehatan dan Keselamatan Kerja"}
    ]'::jsonb,
    '[
        {"id": "beh-1", "title": "Disiplin"},
        {"id": "beh-2", "title": "Kerjasama"},
        {"id": "beh-3", "title": "Pelayanan"},
        {"id": "beh-4", "title": "Integritas"},
        {"id": "beh-5", "title": "Kepemimpinan"}
    ]'::jsonb,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
FROM departments d
CROSS JOIN academic_periods p
WHERE d.code = 'TKR' AND d.deleted_at IS NULL
  AND p.name = '2025/2026' AND p.deleted_at IS NULL
ON CONFLICT (department_id, period_id) WHERE deleted_at IS NULL DO UPDATE
SET
    title = EXCLUDED.title,
    work_aspects = EXCLUDED.work_aspects,
    behavior_aspects = EXCLUDED.behavior_aspects,
    updated_at = CURRENT_TIMESTAMP;

-- Pastikan template SIJA dan KJIJ belum dibuat / dibersihkan karena penilaian periode 2026 masih dalam proses
DELETE FROM assessment_templates
WHERE department_id IN (SELECT id FROM departments WHERE code IN ('SIJA', 'KJIJ'))
  AND period_id IN (SELECT id FROM academic_periods WHERE name = '2025/2026');

END $$;
