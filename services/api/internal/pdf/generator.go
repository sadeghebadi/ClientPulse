package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrChromeNotFound = errors.New("pdf: chromium or google-chrome binary not found")
	ErrPDFGeneration  = errors.New("pdf: generation failed")
)

// Generator builds professional PDF reports from structured ReportData.
type Generator struct {
	chromeBinary string
}

// NewGenerator detects an installed Chromium or Chrome binary on the system.
func NewGenerator() (*Generator, error) {
	binaryPath := findChromeBinary()
	if binaryPath == "" {
		return nil, ErrChromeNotFound
	}
	return &Generator{chromeBinary: binaryPath}, nil
}

func findChromeBinary() string {
	if custom := os.Getenv("CHROME_BIN"); custom != "" {
		if _, err := exec.LookPath(custom); err == nil {
			return custom
		}
	}

	candidates := []string{
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"google-chrome-stable",
		"google-chrome",
		"chromium",
		"chromium-browser",
	}

	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
	}
	return ""
}

// GeneratePDF renders the report HTML and invokes Chrome in headless mode to produce a high-fidelity PDF.
func (g *Generator) GeneratePDF(ctx context.Context, data ReportData) ([]byte, error) {
	htmlContent := renderReportHTML(data)

	tempDir, err := os.MkdirTemp("", "clientpulse_pdf_*")
	if err != nil {
		return nil, fmt.Errorf("failed creating temp directory for pdf: %w", err)
	}
	defer os.RemoveAll(tempDir)

	htmlPath := filepath.Join(tempDir, "report.html")
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0600); err != nil {
		return nil, fmt.Errorf("failed writing report html: %w", err)
	}

	pdfPath := filepath.Join(tempDir, "report.pdf")

	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, g.chromeBinary,
		"--headless",
		"--disable-gpu",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-software-rasterizer",
		"--no-margins",
		"--host-resolver-rules=MAP * ~NOTFOUND",
		"--proxy-server=http://127.0.0.1:0",
		"--disable-remote-fonts",
		"--disable-background-networking",
		"--disable-sync",
		"--disable-translate",
		"--disable-extensions",
		"--disable-default-apps",
		"--print-to-pdf="+pdfPath,
		htmlPath,
	)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %v, stderr: %s", ErrPDFGeneration, err, errBuf.String())
	}

	pdfBytes, err := os.ReadFile(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading generated pdf: %w", err)
	}

	if len(pdfBytes) < 500 {
		return nil, fmt.Errorf("generated pdf is unexpectedly small or empty: %d bytes", len(pdfBytes))
	}

	return pdfBytes, nil
}

// renderReportHTML constructs modern, printable HTML/CSS with bilingual RTL/LTR typography.
func renderReportHTML(data ReportData) string {
	isFa := data.Locale == "fa"
	dir := "ltr"
	fontFamily := "'DejaVu Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif"
	if isFa {
		dir = "rtl"
		fontFamily = "'Noto Sans Arabic', 'IBM Plex Sans Arabic', 'DejaVu Sans', Tahoma, sans-serif"
	}

	primaryColor := data.Branding.PrimaryColor
	if primaryColor == "" {
		primaryColor = "#0284c7"
	}
	accentColor := data.Branding.AccentColor
	if accentColor == "" {
		accentColor = "#4f46e5"
	}

	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"")
	if isFa {
		sb.WriteString("fa\" dir=\"rtl\">")
	} else {
		sb.WriteString("en\" dir=\"ltr\">")
	}
	sb.WriteString(`
<head>
<meta charset="utf-8">
<title>`)
	sb.WriteString(html.EscapeString(data.Title))
	sb.WriteString(`</title>
<style>
  @page {
    size: A4;
    margin: 14mm 16mm 14mm 16mm;
    @bottom-center {
      content: "Page " counter(page) " of " counter(pages);
      font-size: 8pt;
      color: #64748b;
    }
  }
  * { box-sizing: border-box; }
  body {
    font-family: `)
	sb.WriteString(fontFamily)
	sb.WriteString(`;
    margin: 0;
    padding: 0;
    font-size: 11pt;
    line-height: 1.5;
    color: #1e293b;
    background: #ffffff;
    direction: `)
	sb.WriteString(dir)
	sb.WriteString(`;
  }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 2px solid `)
	sb.WriteString(primaryColor)
	sb.WriteString(`;
    padding-bottom: 12px;
    margin-bottom: 20px;
  }
  .branding-title {
    font-size: 20pt;
    font-weight: 700;
    color: `)
	sb.WriteString(primaryColor)
	sb.WriteString(`;
    margin: 0;
  }
  .branding-sub {
    font-size: 9pt;
    color: #64748b;
    margin-top: 2px;
  }
  .logo-img {
    max-height: 50px;
    max-width: 160px;
    object-fit: contain;
  }
  .meta-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 12px 16px;
    margin-bottom: 20px;
    font-size: 9.5pt;
  }
  .meta-item { display: flex; flex-direction: column; }
  .meta-label { font-size: 8pt; text-transform: uppercase; color: #64748b; font-weight: 600; margin-bottom: 2px; }
  .meta-value { font-weight: 600; color: #0f172a; }
  
  .section-title {
    font-size: 13pt;
    font-weight: 700;
    color: #0f172a;
    border-bottom: 1px solid #e2e8f0;
    padding-bottom: 6px;
    margin-top: 22px;
    margin-bottom: 12px;
    page-break-after: avoid;
  }
  
  .summary-cards {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    margin-bottom: 20px;
  }
  .card {
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    padding: 10px;
    text-align: center;
  }
  .card-val {
    font-size: 15pt;
    font-weight: 700;
    color: `)
	sb.WriteString(primaryColor)
	sb.WriteString(`;
  }
  .card-label {
    font-size: 8pt;
    color: #64748b;
    margin-top: 2px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 18px;
    font-size: 9pt;
    page-break-inside: auto;
  }
  tr { page-break-inside: avoid; page-break-after: auto; }
  th {
    background: #f1f5f9;
    color: #334155;
    font-weight: 600;
    text-align: `)
	if isFa {
		sb.WriteString("right")
	} else {
		sb.WriteString("left")
	}
	sb.WriteString(`;
    padding: 7px 10px;
    border: 1px solid #e2e8f0;
  }
  td {
    padding: 7px 10px;
    border: 1px solid #e2e8f0;
    color: #1e293b;
    vertical-align: top;
  }
  .badge {
    display: inline-block;
    padding: 2px 7px;
    border-radius: 4px;
    font-size: 7.5pt;
    font-weight: 600;
  }
  .badge-success { background: #dcfce7; color: #15803d; }
  .badge-warning { background: #fef9c3; color: #854d0e; }
  .badge-danger { background: #fee2e2; color: #b91c1c; }
  .badge-info { background: #e0f2fe; color: #0369a1; }
  .badge-neutral { background: #f1f5f9; color: #475569; }

  .rec-box {
    border-left: 3px solid `)
	sb.WriteString(accentColor)
	if isFa {
		sb.WriteString("; border-left: none; border-right: 3px solid " + accentColor)
	}
	sb.WriteString(`;
    background: #faf5ff;
    padding: 8px 12px;
    margin-bottom: 10px;
    border-radius: 0 4px 4px 0;
  }
  .rec-title { font-weight: 700; font-size: 9.5pt; color: #581c87; margin-bottom: 2px; }
  .rec-desc { font-size: 8.5pt; color: #3b0764; margin: 0; }

  .limitation-box {
    background: #f8fafc;
    border: 1px dashed #cbd5e1;
    border-radius: 6px;
    padding: 10px 14px;
    margin-top: 24px;
    font-size: 8pt;
    color: #64748b;
    line-height: 1.4;
    page-break-inside: avoid;
  }
  .footer {
    margin-top: 20px;
    padding-top: 10px;
    border-top: 1px solid #e2e8f0;
    font-size: 8pt;
    color: #94a3b8;
    text-align: center;
  }
</style>
</head>
<body>

<div class="header">
  <div>
    <h1 class="branding-title">`)
	if data.Branding.DisplayName != "" {
		sb.WriteString(html.EscapeString(data.Branding.DisplayName))
	} else {
		sb.WriteString("ClientPulse")
	}
	sb.WriteString(`</h1>
    <div class="branding-sub">`)
	if isFa {
		sb.WriteString("گزارش دوره‌ای نگهداری و پایش سلامت وب‌سایت")
	} else {
		sb.WriteString("Monthly Website Care & Operations Report")
	}
	sb.WriteString(`</div>
  </div>
`)
	if data.Branding.LogoDataURI != "" {
		sb.WriteString(`  <img src="` + data.Branding.LogoDataURI + `" class="logo-img" alt="Agency Logo">\n`)
	}
	sb.WriteString(`</div>

<div class="meta-grid">
  <div class="meta-item">
    <span class="meta-label">`)
	if isFa {
		sb.WriteString("مشتری")
	} else {
		sb.WriteString("Client")
	}
	sb.WriteString(`</span>
    <span class="meta-value">` + html.EscapeString(data.Client.Name) + `</span>
  </div>
  <div class="meta-item">
    <span class="meta-label">`)
	if isFa {
		sb.WriteString("دوره گزارش")
	} else {
		sb.WriteString("Report Period")
	}
	sb.WriteString(`</span>
    <span class="meta-value">` + data.PeriodStart.Format("2006-01-02") + ` &rarr; ` + data.PeriodEnd.Format("2006-01-02") + ` (` + html.EscapeString(data.OrganizationTimezone) + `)</span>
  </div>
  <div class="meta-item">
    <span class="meta-label">`)
	if isFa {
		sb.WriteString("تاریخ صدور")
	} else {
		sb.WriteString("Generated At")
	}
	sb.WriteString(`</span>
    <span class="meta-value">` + data.GeneratedAt.Format("2006-01-02 15:04:05 UTC") + ` (v` + fmt.Sprintf("%d", data.VersionNumber) + `)</span>
  </div>
  <div class="meta-item">
    <span class="meta-label">`)
	if isFa {
		sb.WriteString("قالب و نسخه")
	} else {
		sb.WriteString("Template Version")
	}
	sb.WriteString(`</span>
    <span class="meta-value">` + html.EscapeString(data.TemplateVersion) + `</span>
  </div>
</div>

<h2 class="section-title">`)
	if isFa {
		sb.WriteString("خلاصه وضعیت اجرایی (Executive Summary)")
	} else {
		sb.WriteString("Executive Summary")
	}
	sb.WriteString(`</h2>

<div class="summary-cards">
  <div class="card">
    <div class="card-val">`)
	if data.Summary.AverageUptime != nil {
		sb.WriteString(fmt.Sprintf("%.2f%%", *data.Summary.AverageUptime))
	} else {
		if isFa {
			sb.WriteString("داده ناکافی")
		} else {
			sb.WriteString("Insufficient data")
		}
	}
	sb.WriteString(`</div>
    <div class="card-label">`)
	if isFa {
		sb.WriteString("میانگین پایداری (Uptime)")
	} else {
		sb.WriteString("Average Uptime")
	}
	sb.WriteString(`</div>
  </div>
  <div class="card">
    <div class="card-val">` + fmt.Sprintf("%d", data.Summary.TotalMonitors) + `</div>
    <div class="card-label">`)
	if isFa {
		sb.WriteString("مانیتورهای فعال")
	} else {
		sb.WriteString("Monitored Endpoints")
	}
	sb.WriteString(`</div>
  </div>
  <div class="card">
    <div class="card-val">` + fmt.Sprintf("%d", data.Summary.TotalIncidents) + `</div>
    <div class="card-label">`)
	if isFa {
		sb.WriteString("رخدادهای قطعی")
	} else {
		sb.WriteString("Incidents Detected")
	}
	sb.WriteString(`</div>
  </div>
  <div class="card">
    <div class="card-val">` + fmt.Sprintf("%d", data.Summary.TotalActivities) + `</div>
    <div class="card-label">`)
	if isFa {
		sb.WriteString("فعالیت‌های نگهداری")
	} else {
		sb.WriteString("Maintenance Actions")
	}
	sb.WriteString(`</div>
  </div>
</div>
<p style="font-size: 9.5pt; color: #334155; margin-bottom: 18px;">` + html.EscapeString(data.Summary.ExecutiveStatement) + `</p>

<!-- 6. Assets Overview -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("وضعیت دارایی‌ها و وب‌سایت‌ها (Website & Asset Overview)")
	} else {
		sb.WriteString("Website and API Asset Overview")
	}
	sb.WriteString(`</h2>
<table>
  <thead>
    <tr>
      <th>` + ifElse(isFa, "نام دارایی", "Asset Name") + `</th>
      <th>` + ifElse(isFa, "نوع", "Type") + `</th>
      <th>` + ifElse(isFa, "آدرس", "URL") + `</th>
      <th>` + ifElse(isFa, "محیط", "Environment") + `</th>
      <th>` + ifElse(isFa, "وضعیت گواهی SSL", "SSL Certificate") + `</th>
      <th>` + ifElse(isFa, "انقضای SSL", "SSL Expiry Date") + `</th>
    </tr>
  </thead>
  <tbody>`)
	if len(data.Assets) == 0 {
		sb.WriteString(`<tr><td colspan="6" style="text-align:center; color:#94a3b8;">` + ifElse(isFa, "هیچ دارایی برای این مشتری ثبت نشده است.", "No assets registered for this client.") + `</td></tr>`)
	} else {
		for _, a := range data.Assets {
			sb.WriteString(`<tr>
        <td><strong>` + html.EscapeString(a.Name) + `</strong></td>
        <td>` + html.EscapeString(a.Type) + `</td>
        <td>` + html.EscapeString(a.URL) + `</td>
        <td>` + html.EscapeString(a.Environment) + `</td>
        <td>` + renderBadge(a.SSLStatus) + `</td>
        <td>` + html.EscapeString(ifElse(a.SSLExpiryDate != "", a.SSLExpiryDate, "N/A")) + `</td>
      </tr>`)
		}
	}
	sb.WriteString(`</tbody>
</table>

<!-- 7. Uptime & Coverage -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("پایداری و پوشش مانیتورینگ (Uptime Percentage & Coverage)")
	} else {
		sb.WriteString("Uptime Percentage and Monitoring Coverage")
	}
	sb.WriteString(`</h2>
<table>
  <thead>
    <tr>
      <th>` + ifElse(isFa, "مانیتور", "Monitor Target") + `</th>
      <th>` + ifElse(isFa, "دارایی", "Asset") + `</th>
      <th>` + ifElse(isFa, "درصد پایداری", "Uptime %") + `</th>
      <th>` + ifElse(isFa, "پوشش مانیتورینگ", "Monitoring Coverage") + `</th>
      <th>` + ifElse(isFa, "میانگین پاسخ", "Avg Latency") + `</th>
      <th>` + ifElse(isFa, "میانه پاسخ", "Median Latency") + `</th>
      <th>` + ifElse(isFa, "تعداد تست‌ها", "Scheduled Checks") + `</th>
    </tr>
  </thead>
  <tbody>`)
	if len(data.Monitors) == 0 {
		sb.WriteString(`<tr><td colspan="7" style="text-align:center; color:#94a3b8;">` + ifElse(isFa, "هیچ مانیتوری برای این مشتری ثبت نشده است.", "No monitors configured for this client.") + `</td></tr>`)
	} else {
		for _, m := range data.Monitors {
			uptimeStr := "Insufficient data"
			if isFa {
				uptimeStr = "داده ناکافی"
			}
			if !m.InsufficientData && m.UptimePercentage != nil {
				uptimeStr = fmt.Sprintf("%.2f%%", *m.UptimePercentage)
			}
			avgLat := "N/A"
			if m.AvgResponseTimeMs != nil {
				avgLat = fmt.Sprintf("%d ms", *m.AvgResponseTimeMs)
			}
			medLat := "N/A"
			if m.MedianResponseTimeMs != nil {
				medLat = fmt.Sprintf("%d ms", *m.MedianResponseTimeMs)
			}

			sb.WriteString(`<tr>
        <td>` + html.EscapeString(m.Name) + `</td>
        <td>` + html.EscapeString(m.AssetName) + `</td>
        <td><strong>` + uptimeStr + `</strong></td>
        <td>` + fmt.Sprintf("%.1f%%", m.CoveragePercentage) + `</td>
        <td>` + avgLat + `</td>
        <td>` + medLat + `</td>
        <td>` + fmt.Sprintf("%d", m.TotalChecks) + `</td>
      </tr>`)
		}
	}
	sb.WriteString(`</tbody>
</table>

<!-- 9. Incidents & Durations -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("تاریخچه رخدادها و قطعی‌ها (Incident History & Durations)")
	} else {
		sb.WriteString("Incident History and Durations")
	}
	sb.WriteString(`</h2>
<table>
  <thead>
    <tr>
      <th>` + ifElse(isFa, "هدف", "Target") + `</th>
      <th>` + ifElse(isFa, "نوع", "Type") + `</th>
      <th>` + ifElse(isFa, "علت خطا", "Failure Category") + `</th>
      <th>` + ifElse(isFa, "شروع قطعی", "Confirmed At") + `</th>
      <th>` + ifElse(isFa, "رفع قطعی", "Resolved At") + `</th>
      <th>` + ifElse(isFa, "مدت قطعی", "Duration") + `</th>
    </tr>
  </thead>
  <tbody>`)
	if len(data.Incidents) == 0 {
		sb.WriteString(`<tr><td colspan="6" style="text-align:center; color:#15803d; background:#f0fdf4;">` + ifElse(isFa, "در این دوره هیچ رخداد قطعی ثبت نشده است (100% بدون قطعی).", "No incidents were recorded in this period.") + `</td></tr>`)
	} else {
		for _, inc := range data.Incidents {
			resText := ifElse(inc.ResolvedAt != "", inc.ResolvedAt, ifElse(isFa, "در حال پیگیری", "Ongoing"))
			sb.WriteString(`<tr>
        <td><strong>` + html.EscapeString(inc.TargetName) + `</strong></td>
        <td>` + html.EscapeString(inc.TargetType) + `</td>
        <td>` + html.EscapeString(inc.FailureCategory) + `</td>
        <td>` + html.EscapeString(inc.ConfirmedAt) + `</td>
        <td>` + html.EscapeString(resText) + `</td>
        <td>` + fmt.Sprintf("%d min", inc.DurationMinutes) + `</td>
      </tr>`)
		}
	}
	sb.WriteString(`</tbody>
</table>

<!-- 14. Heartbeats & Cron Backups -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("مانیتورینگ جاب‌های پس‌زمینه و بکاپ (Heartbeat & Backup Job Status)")
	} else {
		sb.WriteString("Heartbeat and Backup Job Status")
	}
	sb.WriteString(`</h2>
<table>
  <thead>
    <tr>
      <th>` + ifElse(isFa, "عنوان جاب", "Job Name") + `</th>
      <th>` + ifElse(isFa, "دارایی مرتبط", "Asset") + `</th>
      <th>` + ifElse(isFa, "وضعیت", "Status") + `</th>
      <th>` + ifElse(isFa, "فاصله زمانی مورد انتظار", "Expected Interval") + `</th>
      <th>` + ifElse(isFa, "آخرین دریافت سیگنال", "Last Received") + `</th>
    </tr>
  </thead>
  <tbody>`)
	if len(data.Heartbeats) == 0 {
		sb.WriteString(`<tr><td colspan="5" style="text-align:center; color:#94a3b8;">` + ifElse(isFa, "مانیتورینگ ثبت نشده است (Not monitored)", "Not monitored") + `</td></tr>`)
	} else {
		for _, hb := range data.Heartbeats {
			lastRec := ifElse(hb.LastReceivedAt != "", hb.LastReceivedAt, ifElse(isFa, "دریافت نشده", "None yet"))
			sb.WriteString(`<tr>
        <td><strong>` + html.EscapeString(hb.Name) + `</strong></td>
        <td>` + html.EscapeString(hb.AssetName) + `</td>
        <td>` + renderBadge(hb.Status) + `</td>
        <td>` + html.EscapeString(hb.ExpectedInterval) + `</td>
        <td>` + html.EscapeString(lastRec) + `</td>
      </tr>`)
		}
	}
	sb.WriteString(`</tbody>
</table>
<p style="font-size: 8pt; color: #64748b; margin-top: -8px; margin-bottom: 16px;">
  * ` + ifElse(isFa, "توجه: دریافت سیگنال سلامت به معنی اجرای اسکریپت بکاپ است؛ این امر به تنهایی صحت بازگردانی اطلاعات یا بی‌نقص بودن فایل را اثبات نمی‌کند.", "Notice: Heartbeat receipt indicates that the background script executed, but does not independently verify backup restoration validity or completeness.") + `
</p>

<!-- 16. Maintenance Activities -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("کارهای نگهداری و ارتقای انجام‌شده (Maintenance Activities)")
	} else {
		sb.WriteString("Maintenance Activities")
	}
	sb.WriteString(`</h2>
<table>
  <thead>
    <tr>
      <th>` + ifElse(isFa, "عنوان فعالیت", "Title") + `</th>
      <th>` + ifElse(isFa, "دسته‌بندی", "Category") + `</th>
      <th>` + ifElse(isFa, "تاریخ انجام", "Date Performed") + `</th>
      <th>` + ifElse(isFa, "مدت", "Duration") + `</th>
      <th>` + ifElse(isFa, "دارایی", "Asset") + `</th>
      <th>` + ifElse(isFa, "شرح اقدام", "Description") + `</th>
    </tr>
  </thead>
  <tbody>`)
	if len(data.Maintenance) == 0 {
		sb.WriteString(`<tr><td colspan="6" style="text-align:center; color:#64748b; padding: 14px;">` + ifElse(isFa, "هیچ فعالیت نگهداری در این دوره ثبت نشده است.", "No activities were recorded in this period.") + `</td></tr>`)
	} else {
		for _, act := range data.Maintenance {
			durStr := "—"
			if act.DurationMinutes != nil {
				durStr = fmt.Sprintf("%d min", *act.DurationMinutes)
			}
			sb.WriteString(`<tr>
        <td><strong>` + html.EscapeString(act.Title) + `</strong></td>
        <td>` + renderCategoryBadge(act.Category) + `</td>
        <td>` + html.EscapeString(act.PerformedAt) + `</td>
        <td>` + durStr + `</td>
        <td>` + html.EscapeString(act.AssetName) + `</td>
        <td style="font-size:8.5pt;">` + html.EscapeString(act.Description) + `</td>
      </tr>`)
		}
	}
	sb.WriteString(`</tbody>
</table>

<!-- 17. Recommendations & Open Issues -->
<h2 class="section-title">`)
	if isFa {
		sb.WriteString("توصیه‌های فنی و موارد نیازمند اقدام (Open Issues & Recommendations)")
	} else {
		sb.WriteString("Open Issues and Recommendations")
	}
	sb.WriteString(`</h2>`)
	if len(data.Recommendations) == 0 {
		sb.WriteString(`<p style="font-size:9pt; color:#15803d;">` + ifElse(isFa, "تمامی شاخص‌های پایداری و نگهداری در وضعیت بهینه هستند و هیچ اقدام فوری پیشنهاد نمی‌شود.", "All operational and maintenance metrics are optimal. No urgent actions required.") + `</p>`)
	} else {
		for _, rec := range data.Recommendations {
			sb.WriteString(`<div class="rec-box">
        <div class="rec-title">` + html.EscapeString(rec.Title) + `</div>
        <p class="rec-desc">` + html.EscapeString(rec.Description) + `</p>
      </div>`)
		}
	}

	sb.WriteString(`
<!-- 18. Limitations and Methods -->
<div class="limitation-box">
  <strong>` + ifElse(isFa, "روش محاسبه، پوشش و محدودیت‌ها:", "Calculation Methods, Coverage and Limitations:") + `</strong><br>
  ` + html.EscapeString(data.LimitationNotice) + `
</div>

<div class="footer">
  ` + html.EscapeString(data.Branding.FooterText) + ` | ` + html.EscapeString(data.Branding.ContactEmail) + ` | ` + html.EscapeString(data.Branding.WebsiteURL) + `
</div>

</body>
</html>`)

	return sb.String()
}

func ifElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func renderBadge(status string) string {
	s := strings.ToLower(status)
	switch s {
	case "up", "healthy", "valid", "active", "resolved":
		return `<span class="badge badge-success">` + html.EscapeString(status) + `</span>`
	case "down", "missed", "expired", "open":
		return `<span class="badge badge-danger">` + html.EscapeString(status) + `</span>`
	case "pending", "late", "expiring_soon", "warning":
		return `<span class="badge badge-warning">` + html.EscapeString(status) + `</span>`
	default:
		return `<span class="badge badge-neutral">` + html.EscapeString(status) + `</span>`
	}
}

func renderCategoryBadge(cat string) string {
	return `<span class="badge badge-info">` + html.EscapeString(cat) + `</span>`
}

