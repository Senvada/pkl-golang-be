package seeder

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AspectTemplateItem struct {
	ID       string `json:"id"`
	Category string `json:"category,omitempty"`
	Title    string `json:"title"`
}

func SeedAssessments(db *gorm.DB) error {
	// 1. Ensure default academic period exists
	var periodID uuid.UUID
	var count int64
	err := db.Table("academic_periods").Where("name = ? AND deleted_at IS NULL", "2025/2026").Count(&count).Error
	if err != nil {
		return fmt.Errorf("failed to check academic period: %w", err)
	}

	if count == 0 {
		periodID = uuid.New()
		if err := db.Exec(
			"INSERT INTO academic_periods (id, name, is_active, created_at, updated_at) VALUES (?, '2025/2026', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			periodID,
		).Error; err != nil {
			return fmt.Errorf("failed to insert academic period: %w", err)
		}
		slog.Info("✅ Seeded default academic period 2025/2026")
	} else {
		var period struct{ ID uuid.UUID }
		if err := db.Table("academic_periods").Select("id").Where("name = ? AND deleted_at IS NULL", "2025/2026").Scan(&period).Error; err != nil {
			return fmt.Errorf("failed to fetch academic period id: %w", err)
		}
		periodID = period.ID
	}

	// 2. Fetch all departments
	type deptInfo struct {
		ID   uuid.UUID
		Code string
		Name string
	}
	var depts []deptInfo
	if err := db.Table("departments").Select("id, code, name").Where("deleted_at IS NULL").Scan(&depts).Error; err != nil {
		return fmt.Errorf("failed to fetch departments: %w", err)
	}

	deptMap := make(map[string]deptInfo)
	for _, d := range depts {
		deptMap[d.Code] = d
	}

	standardBehaviorAspects := []AspectTemplateItem{
		{ID: uuid.NewString(), Title: "Disiplin"},
		{ID: uuid.NewString(), Title: "Kerjasama"},
		{ID: uuid.NewString(), Title: "Pelayanan"},
		{ID: uuid.NewString(), Title: "Integritas"},
		{ID: uuid.NewString(), Title: "Kepemimpinan"},
	}

	tjktBehaviorAspects := []AspectTemplateItem{
		{ID: uuid.NewString(), Title: "Soft Skills (Komunikasi efektif, kerjasama tim, problem solving, time manajement, dan adaptif)"},
		{ID: uuid.NewString(), Title: "Tanggung Jawab dan Integritas"},
		{ID: uuid.NewString(), Title: "Pengembangan Diri"},
	}

	templates := []struct {
		DeptCode string
		Title    string
		Work     []AspectTemplateItem
		Behavior []AspectTemplateItem
	}{
		// 1. RPL
		{
			DeptCode: "RPL",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Rekayasa Perangkat Lunak",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Category: "Web develop", Title: "Memahami basic Pembuatan Website Sederhana (googlesite, wordpress.com)"},
				{ID: uuid.NewString(), Category: "Web develop", Title: "Mampu membuat website sederhana dengan wordpress (instalation wordpress.org, home, menu, dashboard, web builder, use plugin, domain & hosting)"},
				{ID: uuid.NewString(), Category: "Web develop", Title: "Mampu mengoptimalkan kinerja wordpress dari speed dan SEO (web Optimization dan SEO Optimization)"},
				{ID: uuid.NewString(), Category: "Fullstack develop", Title: "Memahami basic pembuatan aplikasi berbasis web dengan php (html, CSS, Javascript, PHP, MySQL)"},
				{ID: uuid.NewString(), Category: "Fullstack develop", Title: "Mampu merancang dan membuat aplikasi berbasis web dengan menggunakan php framework (Laravel/CI, Bootstrap, Mysql)"},
				{ID: uuid.NewString(), Category: "Fullstack develop", Title: "Modeling (Flowchart, Class Diagram, DFD, ERD/CDM-PDM)"},
				{ID: uuid.NewString(), Category: "Mobile develop", Title: "Mampu membuat aplikasi mobile sederhana berbasis hybrid (Responsive Frontend, Cordova, Ionic)"},
				{ID: uuid.NewString(), Category: "Mobile develop", Title: "Mampu membuat aplikasi mobile hybrid dengan ionic framework (Ionic, React + Typescript, Ionic capasitor + Plugin, Deployment, Git)"},
				{ID: uuid.NewString(), Category: "Graphics / UI-UX Designer", Title: "Memahami dasar-dasar dan unsur desain grafis (Bidang, Ruang, Typografi, Nirmana, Komposisi, Garis dan Prespektif)"},
				{ID: uuid.NewString(), Category: "Graphics / UI-UX Designer", Title: "Mampu menjalankan software desain grafis berbasis bitmap dan vektor (Adobe Photoshop, Adobe Illustrator, CorelDRAW)"},
				{ID: uuid.NewString(), Category: "Graphics / UI-UX Designer", Title: "Mampu membuat dan mengedit desain berbasis bitmap dan vektor (Mockup, Publishing, Web Design, Art Design)"},
				{ID: uuid.NewString(), Category: "Graphics / UI-UX Designer", Title: "Mampu memahami UI/UX dan usability (pengetahuan UI/UX dan Membuat rancang komponen dasar (Design System))"},
				{ID: uuid.NewString(), Category: "K3LH", Title: "K3LH (Kesehatan dan Keselamatan Kerja dan Lingkungan Hidup)"},
			},
		},
		// 2. TITL
		{
			DeptCode: "TITL",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Instalasi Tenaga Listrik",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Analisa trouble shooting peralatan rumah tangga yang menggunakan motor listrik"},
				{ID: uuid.NewString(), Title: "Penanganan trouble shooting peralatan rumah tangga yang menggunakan motor listrik"},
				{ID: uuid.NewString(), Title: "Pengujian peralatan rumah tangga yang menggunakan motor listrik"},
				{ID: uuid.NewString(), Title: "Perencanaan gulung ulang mesin listrik (motor/generator/transformator)"},
				{ID: uuid.NewString(), Title: "Menggulung ulang mesin listrik"},
				{ID: uuid.NewString(), Title: "Pengujian mesin listrik setelah di gulung ulang"},
				{ID: uuid.NewString(), Title: "Merencanakan pekerjaan pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
				{ID: uuid.NewString(), Title: "Menganalisis kebutuhan material listrik pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
				{ID: uuid.NewString(), Title: "Penanganan trouble shooting pemasangan instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik"},
				{ID: uuid.NewString(), Title: "Pengujian instalasi penerangan/instalasi tenaga listrik/instalasi motor listrik setelah pemasangan"},
				{ID: uuid.NewString(), Title: "Menerapkan K3 pada manusia"},
				{ID: uuid.NewString(), Title: "Menerapkan K3 pada peralatan"},
			},
		},
		// 3. TAB
		{
			DeptCode: "TAB",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Alat Berat",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Preventive Maintenance"},
				{ID: uuid.NewString(), Title: "Periodic Service"},
				{ID: uuid.NewString(), Title: "Corrective Maintenance"},
				{ID: uuid.NewString(), Title: "Pekerjaan Dasar Engine"},
				{ID: uuid.NewString(), Title: "Pekerjaan Dasar Hidrolik"},
				{ID: uuid.NewString(), Title: "Pekerjaan Dasar Powertrain dan undercarriage"},
				{ID: uuid.NewString(), Title: "Pekerjaan Dasar Kelistrikan dan Sistem Kontrol"},
				{ID: uuid.NewString(), Title: "Manajemen K3LH"},
			},
		},
		// 4. OTO (Teknik Otomotif / Ototronik)
		{
			DeptCode: "OTO",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Ototronik",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Tune up engine EFI"},
				{ID: uuid.NewString(), Title: "Kelistrikan lampu"},
				{ID: uuid.NewString(), Title: "Power Window / Central Lock"},
				{ID: uuid.NewString(), Title: "Sistem Pengapian"},
				{ID: uuid.NewString(), Title: "Sistem Rem / ABS"},
				{ID: uuid.NewString(), Title: "Sistem Pemindah Tenaga"},
				{ID: uuid.NewString(), Title: "Sistem Kemudi / Power Steering"},
				{ID: uuid.NewString(), Title: "Sistem Suspensi"},
				{ID: uuid.NewString(), Title: "Kesehatan dan Keselamatan Kerja"},
			},
		},
		// 5. TPM
		{
			DeptCode: "TPM",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Pemesinan",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Gambar Teknik Mesin"},
				{ID: uuid.NewString(), Title: "Pekerjaan Dasar Teknik Mesin"},
				{ID: uuid.NewString(), Title: "Dasar Perencanaan Teknik Mesin"},
				{ID: uuid.NewString(), Title: "Gambar Teknik Manufaktur"},
				{ID: uuid.NewString(), Title: "Teknik Pemesinan Bubut"},
				{ID: uuid.NewString(), Title: "Teknik Pemesinan Frais"},
				{ID: uuid.NewString(), Title: "Teknik Pemesinan Gerindra"},
				{ID: uuid.NewString(), Title: "Teknik Pemesinan CNC"},
				{ID: uuid.NewString(), Title: "Produk Kreatif dan Kewirausahaan"},
				{ID: uuid.NewString(), Title: "Sistem Manajemen Keselamatan dan Kesehatan Kerja"},
			},
		},
		// 6. TKJ (TJKT)
		{
			DeptCode: "TKJ",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Jaringan Komputer dan Telekomunikasi",
			Behavior: tjktBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Etika Profesi dan Keselamatan Kerja"},
				{ID: uuid.NewString(), Title: "Dasar Pemrograman dan Algoritma"},
				{ID: uuid.NewString(), Title: "Infrastruktur Telekomunikasi dan Media Jaringan"},
				{ID: uuid.NewString(), Title: "Perencanaan dan Implementasi Jaringan"},
				{ID: uuid.NewString(), Title: "Administrasi Sistem dan Layanan Jaringan"},
				{ID: uuid.NewString(), Title: "Teknologi Jaringan Kabel dan Nirkabel"},
				{ID: uuid.NewString(), Title: "Internet Of Things (IoT)"},
				{ID: uuid.NewString(), Title: "Teknologi Layanan Jaringan"},
				{ID: uuid.NewString(), Title: "Marketing dan Presentasi Produk"},
			},
		},
		// 7. DPIB
		{
			DeptCode: "DPIB",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Desain Pemodelan dan Informasi Bangunan",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Melaksanakan Prosedur Keselamatan dan Kesehatan Kerja (K3)"},
				{ID: uuid.NewString(), Title: "Menggunakan Perintah Dasar Gambar, Melakukan Modifikasi Gambar, Menggunakan Fasilitas Pendukung Gambar dan Mencetak Hasil Gambar Sesuai Prosedur/Ketentuan"},
				{ID: uuid.NewString(), Title: "Menyajikan Gambar Proyeksi Bangunan Sesuai Kaidah Gambar Teknik (Gambar Situasi Denah, Potongan dan Tampak)"},
				{ID: uuid.NewString(), Title: "Menyajikan Gambar Konstruksi Sesuai Kaidah Gambar Teknik (Pondasi, Lantai, Dinding, Kusen, Daun Pintu, Jendela Ventilasi Beton Bertulang, Tangga, Atap, Langit-langit, Instalasi Listrik, Instalasi plumbing dan Drainase)"},
				{ID: uuid.NewString(), Title: "Menyusun Rencana Anggaran Terkait Dengan Tugas Perencanaan"},
			},
		},
		// 8. TKR (TKRO)
		{
			DeptCode: "TKR",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Teknik Kendaraan Ringan Otomotif",
			Behavior: standardBehaviorAspects,
			Work: []AspectTemplateItem{
				{ID: uuid.NewString(), Title: "Tune Up / Perawatan Berkala Mobil"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Utama Mesin (Mekanisme Katup, Pelumasan, Pendinginan)"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Bahan Bakar Bensin dan Diesel"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Starter"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Pengisian"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Pengapian (Konvensional dan Elektronik)"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Penerangan dan Panel Instrumen"},
				{ID: uuid.NewString(), Title: "Perawatan Aksesoris (Sistem AC, Sistem Audio, Power Window dan Central Lock)"},
				{ID: uuid.NewString(), Title: "Perawatan Kopling dan Transmisi Manual/Otomatis"},
				{ID: uuid.NewString(), Title: "Perawatan Poros Propeller dan Axle"},
				{ID: uuid.NewString(), Title: "Perawatan Gardan"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Rem (Konvensional dan ABS)"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Suspensi"},
				{ID: uuid.NewString(), Title: "Perawatan Sistem Kemudi dan Power Steering"},
				{ID: uuid.NewString(), Title: "Sistem Manajemen Kesehatan dan Keselamatan Kerja"},
			},
		},
		// 9. SIJA (Format belum final dari PDF, template dibuat kosong)
		{
			DeptCode: "SIJA",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Sistem Informasi, Jaringan, dan Aplikasi",
			Behavior: standardBehaviorAspects,
			Work:     []AspectTemplateItem{},
		},
		// 10. KJIJ (Format belum final dari PDF, template dibuat kosong)
		{
			DeptCode: "KJIJ",
			Title:    "Daftar Penilaian Praktik Kerja Lapangan - Konstruksi Jalan, Irigasi, dan Jembatan",
			Behavior: standardBehaviorAspects,
			Work:     []AspectTemplateItem{},
		},
	}

	for _, tpl := range templates {
		dept, exists := deptMap[tpl.DeptCode]
		if !exists {
			continue
		}

		var tplCount int64
		err := db.Table("assessment_templates").
			Where("department_id = ? AND period_id = ? AND deleted_at IS NULL", dept.ID, periodID).
			Count(&tplCount).Error
		if err != nil {
			slog.Warn("Failed to check assessment template", "dept", tpl.DeptCode, "error", err)
			continue
		}

		if tplCount == 0 {
			workJSON, _ := json.Marshal(tpl.Work)
			behaviorJSON, _ := json.Marshal(tpl.Behavior)
			newID := uuid.New()

			if err := db.Exec(
				"INSERT INTO assessment_templates (id, department_id, period_id, title, work_aspects, behavior_aspects, created_at, updated_at) VALUES (?, ?, ?, ?, ?::jsonb, ?::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
				newID, dept.ID, periodID, tpl.Title, string(workJSON), string(behaviorJSON),
			).Error; err != nil {
				slog.Error("Failed to seed template", "dept", tpl.DeptCode, "error", err)
			} else {
				slog.Info("✅ Seeded assessment template", "dept", tpl.DeptCode, "title", tpl.Title)
			}
		}
	}

	return nil
}
