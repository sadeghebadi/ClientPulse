package pdf

import (
	"context"
	"testing"
	"time"
)

func TestGenerateAndValidateEnglishReportPDF(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Skipf("skipping PDF test: Chrome not available: %v", err)
	}

	uptimeVal := 99.85
	avgLat := int64(142)
	medLat := int64(135)
	durationMins := 45

	data := ReportData{
		Title:                "Apex Operations - Monthly Website Care Report (October 2026)",
		ReportType:           "monthly_care",
		Locale:               "en",
		PeriodStart:          time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC),
		OrganizationTimezone: "UTC",
		GeneratedAt:          time.Now().UTC(),
		VersionNumber:        1,
		TemplateVersion:      "v1.0",
		Branding: BrandingData{
			DisplayName:  "Apex Digital Agency",
			PrimaryColor: "#0284c7",
			AccentColor:  "#6366f1",
			FooterText:   "Confidential client report prepared by Apex Digital Agency.",
			ContactEmail: "support@apexdigital.example",
			WebsiteURL:   "https://apexdigital.example",
		},
		Client: ClientData{
			Name: "Acme Ecommerce Corp",
		},
		Summary: SummaryData{
			TotalMonitors:       3,
			AverageUptime:       &uptimeVal,
			TotalIncidents:      1,
			TotalActivities:     2,
			TotalMaintenanceMin: 75,
			ExecutiveStatement:  "During October 2026, all critical eCommerce storefront assets achieved 99.85% operational uptime. A 12-minute outage was swiftly remediated.",
		},
		Assets: []AssetReportData{
			{
				Name:          "Primary Storefront",
				Type:          "web",
				URL:           "https://shop.acme.example",
				Environment:   "production",
				SSLStatus:     "valid",
				SSLExpiryDate: "2027-04-15",
			},
		},
		Monitors: []MonitorReportData{
			{
				Name:                 "Storefront Homepage HTTP",
				AssetName:            "Primary Storefront",
				UptimePercentage:     &uptimeVal,
				CoveragePercentage:   100.0,
				AvgResponseTimeMs:    &avgLat,
				MedianResponseTimeMs: &medLat,
				TotalChecks:          8920,
				FailedChecks:         12,
			},
		},
		Incidents: []IncidentReportData{
			{
				TargetName:      "Storefront Homepage HTTP",
				TargetType:      "monitor",
				ConfirmedAt:     "2026-10-14 09:20 UTC",
				ResolvedAt:      "2026-10-14 09:32 UTC",
				DurationMinutes: 12,
				FailureCategory: "http_5xx",
			},
		},
		Heartbeats: []HeartbeatReportData{
			{
				Name:             "Nightly Database Backup Cron",
				AssetName:        "Primary Storefront",
				Status:           "healthy",
				ExpectedInterval: "24 hours",
				LastReceivedAt:   "2026-10-31 03:00 UTC",
			},
		},
		Maintenance: []MaintenanceReportData{
			{
				Title:           "Security Core & Plugin Updates",
				Category:        "security_review",
				PerformedAt:     "2026-10-18 02:00 UTC",
				DurationMinutes: &durationMins,
				AssetName:       "Primary Storefront",
				Description:     "Applied critical security patches to CMS core and active checkout extensions.",
			},
		},
		Recommendations: []RecommendationData{
			{
				Severity:    "info",
				Title:       "Database Table Reindexing",
				Description: "Recommend scheduled off-peak reindexing of WooCommerce order tables next quarter.",
			},
		},
		LimitationNotice: "Metrics calculated from real scheduled HTTP checks and confirmed incidents. Outage onset measured from confirmed alert trigger. Manual tests excluded.",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pdfBytes, err := gen.GeneratePDF(ctx, data)
	if err != nil {
		t.Fatalf("failed generating English PDF: %v", err)
	}

	requiredPhrases := []string{
		"Apex Digital Agency",
		"Acme Ecommerce Corp",
		"Executive Summary",
		"Website and API Asset Overview",
		"Uptime Percentage and Monitoring Coverage",
		"Incident History and Durations",
		"Heartbeat and Backup Job Status",
		"Maintenance Activities",
		"Open Issues and Recommendations",
		"Calculation Methods, Coverage and Limitations",
	}

	valRes, err := ValidatePDF(ctx, pdfBytes, requiredPhrases, false)
	if err != nil {
		t.Fatalf("PDF validation failed: %v", err)
	}

	if valRes.PageCount < 1 {
		t.Errorf("expected at least 1 page, got %d", valRes.PageCount)
	}
}

func TestGenerateAndValidatePersianReportPDF(t *testing.T) {
	gen, err := NewGenerator()
	if err != nil {
		t.Skipf("skipping PDF test: Chrome not available: %v", err)
	}

	uptimeVal := 99.90
	durationMins := 60

	data := ReportData{
		Title:                "گزارش ماهانه سلامت و نگهداری وب‌سایت - شرکت فناوری آلفا",
		ReportType:           "monthly_care",
		Locale:               "fa",
		PeriodStart:          time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:            time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC),
		OrganizationTimezone: "Asia/Tehran",
		GeneratedAt:          time.Now().UTC(),
		VersionNumber:        1,
		TemplateVersion:      "v1.0",
		Branding: BrandingData{
			DisplayName:  "آژانس دیجیتال توسعه وب پارس",
			PrimaryColor: "#0284c7",
			AccentColor:  "#6366f1",
			FooterText:   "این گزارش محرمانه توسط آژانس توسعه وب پارس برای کارفرما تهیه شده است.",
			ContactEmail: "info@parsweb.example",
			WebsiteURL:   "https://parsweb.example",
		},
		Client: ClientData{
			Name: "شرکت فناوری آلفا",
		},
		Summary: SummaryData{
			TotalMonitors:       2,
			AverageUptime:       &uptimeVal,
			TotalIncidents:      0,
			TotalActivities:     1,
			TotalMaintenanceMin: 60,
			ExecutiveStatement:  "در مهرماه ۱۴۰۵، سرویس‌ها با ضریب پایداری ۹۹.۹۰٪ و بدون هیچ قطعی ثبت‌شده به فعالیت خود ادامه دادند.",
		},
		Assets: []AssetReportData{
			{
				Name:          "وب‌سایت اصلی فروشگاه",
				Type:          "web",
				URL:           "https://alfa.example",
				Environment:   "production",
				SSLStatus:     "valid",
				SSLExpiryDate: "2027-02-20",
			},
		},
		Monitors: []MonitorReportData{
			{
				Name:               "بررسی صفحه اصلی",
				AssetName:          "وب‌سایت اصلی فروشگاه",
				UptimePercentage:   &uptimeVal,
				CoveragePercentage: 100.0,
				TotalChecks:        8900,
			},
		},
		Heartbeats: []HeartbeatReportData{
			{
				Name:             "پک بکاپ شبانه سرور",
				AssetName:        "وب‌سایت اصلی فروشگاه",
				Status:           "healthy",
				ExpectedInterval: "24 hours",
				LastReceivedAt:   "2026-10-31 04:00 UTC",
			},
		},
		Maintenance: []MaintenanceReportData{
			{
				Title:           "به‌روزرسانی امنیتی سیستم و پلاگین‌ها",
				Category:        "security_review",
				PerformedAt:     "2026-10-20 01:00 UTC",
				DurationMinutes: &durationMins,
				AssetName:       "وب‌سایت اصلی فروشگاه",
				Description:     "اعمال وصله‌های امنیتی برای سیستم مدیریت محتوا و بازبینی لاگ‌های امنیتی.",
			},
		},
		Recommendations: []RecommendationData{
			{
				Severity:    "info",
				Title:       "بهینه‌سازی تصاویر",
				Description: "تبدیل فرمت بنرهای صفحه اصلی به WebP برای ارتقای رتبه سرعت لودینگ.",
			},
		},
		LimitationNotice: "محاسبه‌شده بر اساس بررسی‌های دوره‌ای زمان‌بندی‌شده واقعی. دریافت سیگنال سلامت به تنهایی صحت بازگردانی بکاپ را اثبات نمی‌کند.",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pdfBytes, err := gen.GeneratePDF(ctx, data)
	if err != nil {
		t.Fatalf("failed generating Persian PDF: %v", err)
	}

	requiredPhrases := []string{
		"آژانس دیجیتال توسعه وب پارس",
		"شرکت فناوری آلفا",
		"وضعیت اجرایی",
		"وضعیت دارایی‌ها",
		"پایداری و پوشش مانیتورینگ",
		"کارهای نگهداری و ارتقای انجام‌شده",
	}

	valRes, err := ValidatePDF(ctx, pdfBytes, requiredPhrases, true)
	if err != nil {
		if valRes != nil {
			t.Logf("Extracted Text:\n%s\n", valRes.ExtractableText)
		}
		t.Fatalf("Persian PDF validation failed: %v", err)
	}

	if valRes.PageCount < 1 {
		t.Errorf("expected at least 1 page, got %d", valRes.PageCount)
	}
	if !valRes.HasPersianText {
		t.Errorf("expected detected Persian characters in extracted text")
	}
}
