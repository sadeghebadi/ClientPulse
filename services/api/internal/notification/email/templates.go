package email

import (
	"fmt"
	"html"
	"strings"
	"time"
)

type RenderedEmail struct {
	Subject  string
	HTMLBody string
	TextBody string
}

// RenderNotificationEmail generates localized email content for an event.
func RenderNotificationEmail(eventType, locale string, payload map[string]interface{}, appBaseURL string) RenderedEmail {
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	appBaseURL = strings.TrimRight(appBaseURL, "/")

	isFA := strings.ToLower(locale) == "fa"

	switch eventType {
	case "incident.opened":
		return renderIncidentOpened(payload, appBaseURL, isFA)
	case "incident.acknowledged":
		return renderIncidentAcknowledged(payload, appBaseURL, isFA)
	case "incident.resolved":
		return renderIncidentResolved(payload, appBaseURL, isFA)
	case "heartbeat.missed":
		return renderHeartbeatMissed(payload, appBaseURL, isFA)
	case "heartbeat.recovered":
		return renderHeartbeatRecovered(payload, appBaseURL, isFA)
	case "report.ready":
		return renderReportReady(payload, appBaseURL, isFA)
	default:
		return renderGenericTest(payload, appBaseURL, isFA)
	}
}

func getString(m map[string]interface{}, key, def string) string {
	if val, ok := m[key]; ok && val != nil {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return def
}

func baseHTML(dir, lang, title, contentHTML string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin: 0; padding: 24px; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #0f172a;">
  <div style="max-width: 600px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.05);">
    <div style="background-color: #0f172a; padding: 20px 24px; text-align: %s;">
      <h1 style="color: #ffffff; margin: 0; font-size: 18px; font-weight: 600; letter-spacing: -0.025em;">ClientPulse</h1>
    </div>
    <div style="padding: 24px;">
      %s
    </div>
    <div style="background-color: #f1f5f9; padding: 16px 24px; text-align: center; font-size: 12px; color: #64748b; border-top: 1px solid #e2e8f0;">
      <p style="margin: 0;">ClientPulse • Agency Reliability & Maintenance Platform</p>
    </div>
  </div>
</body>
</html>`, lang, dir, html.EscapeString(title), func() string {
		if dir == "rtl" {
			return "right"
		}
		return "left"
	}(), contentHTML)
}

func renderIncidentOpened(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	monitorName := getString(p, "monitor_name", getString(p, "summary", "Website Monitor"))
	incidentID := getString(p, "incident_id", "")
	category := getString(p, "failure_category", "service_error")
	url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incidentID)
	timestamp := time.Now().UTC().Format(time.RFC3339)

	if isFA {
		subject := fmt.Sprintf("🚨 هشدار خرابی: %s از دسترس خارج شد", monitorName)
		content := fmt.Sprintf(`
      <h2 style="color: #dc2626; margin-top: 0;">🚨 حادثه جدید: سرویس از دسترس خارج شد</h2>
      <p>سرویس <strong>%s</strong> دچار اختلال شده و از دسترس خارج شده است.</p>
      <table style="width: 100%%; border-collapse: collapse; margin: 20px 0; font-size: 14px;">
        <tr><td style="padding: 8px 0; color: #64748b;">دسته خرابی:</td><td style="padding: 8px 0; font-weight: 600;">%s</td></tr>
        <tr><td style="padding: 8px 0; color: #64748b;">زمان تایید:</td><td style="padding: 8px 0;">%s</td></tr>
      </table>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #dc2626; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">مشاهده و بررسی حادثه</a>
      </div>
    `, html.EscapeString(monitorName), html.EscapeString(category), timestamp, url)

		text := fmt.Sprintf("هشدار خرابی: %s از دسترس خارج شد\nدسته: %s\nزمان: %s\nبررسی: %s", monitorName, category, timestamp, url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: text}
	}

	subject := fmt.Sprintf("🚨 INCIDENT: %s is DOWN", monitorName)
	content := fmt.Sprintf(`
    <h2 style="color: #dc2626; margin-top: 0;">🚨 Incident Opened: Service Down</h2>
    <p>Service <strong>%s</strong> is failing health checks and has been marked as down.</p>
    <table style="width: 100%%; border-collapse: collapse; margin: 20px 0; font-size: 14px;">
      <tr><td style="padding: 8px 0; color: #64748b;">Failure Category:</td><td style="padding: 8px 0; font-weight: 600;">%s</td></tr>
      <tr><td style="padding: 8px 0; color: #64748b;">Confirmed At:</td><td style="padding: 8px 0;">%s</td></tr>
    </table>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #dc2626; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Incident Details</a>
    </div>
  `, html.EscapeString(monitorName), html.EscapeString(category), timestamp, url)

	text := fmt.Sprintf("INCIDENT OPENED: %s is DOWN\nCategory: %s\nTime: %s\nLink: %s", monitorName, category, timestamp, url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: text}
}

func renderIncidentAcknowledged(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	incidentID := getString(p, "incident_id", "")
	url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incidentID)

	if isFA {
		subject := fmt.Sprintf("👁️ تایید بررسی حادثه #%s", incidentID[:8])
		content := fmt.Sprintf(`
      <h2 style="color: #d97706; margin-top: 0;">👁️ بررسی حادثه آغاز شد</h2>
      <p>حادثه به شناسه <code>%s</code> توسط اعضای تیم تایید و در حال بررسی است.</p>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #d97706; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">مشاهده جزئیات</a>
      </div>
    `, incidentID, url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: fmt.Sprintf("بررسی حادثه: %s", url)}
	}

	subject := fmt.Sprintf("👁️ ACKNOWLEDGED: Incident #%s", incidentID[:8])
	content := fmt.Sprintf(`
    <h2 style="color: #d97706; margin-top: 0;">👁️ Incident Acknowledged</h2>
    <p>Incident <code>%s</code> has been acknowledged by an on-call team member.</p>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #d97706; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Incident</a>
    </div>
  `, incidentID, url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: fmt.Sprintf("Incident acknowledged: %s", url)}
}

func renderIncidentResolved(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	incidentID := getString(p, "incident_id", "")
	monitorName := getString(p, "monitor_name", "Service")
	url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incidentID)

	if isFA {
		subject := fmt.Sprintf("✅ رفع مشکل: %s به حالت عادی بازگشت", monitorName)
		content := fmt.Sprintf(`
      <h2 style="color: #16a34a; margin-top: 0;">✅ رفع مشکل سرویس</h2>
      <p>سرویس <strong>%s</strong> مجدداً در دسترس قرار گرفت و حادثه برطرف شد.</p>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #16a34a; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">مشاهده گزارش رفع اختلال</a>
      </div>
    `, html.EscapeString(monitorName), url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: fmt.Sprintf("رفع مشکل: %s بازگشت به حالت عادی. %s", monitorName, url)}
	}

	subject := fmt.Sprintf("✅ RESOLVED: %s is back UP", monitorName)
	content := fmt.Sprintf(`
    <h2 style="color: #16a34a; margin-top: 0;">✅ Incident Resolved</h2>
    <p>Service <strong>%s</strong> has passed health checks and is now fully operational.</p>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #16a34a; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Resolution Report</a>
    </div>
  `, html.EscapeString(monitorName), url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: fmt.Sprintf("RESOLVED: %s is UP. %s", monitorName, url)}
}

func renderHeartbeatMissed(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	hbName := getString(p, "heartbeat_name", "Heartbeat Monitor")
	url := fmt.Sprintf("%s/heartbeats", appBaseURL)

	if isFA {
		subject := fmt.Sprintf("⚠️ عدم دریافت پالس: %s", hbName)
		content := fmt.Sprintf(`
      <h2 style="color: #ea580c; margin-top: 0;">⚠️ عدم دریافت پالس سیستم</h2>
      <p>پالس دوره‌ای مورد انتظار برای <strong>%s</strong> در موعد مقرر دریافت نشد.</p>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #ea580c; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">بررسی پالس‌ها</a>
      </div>
    `, html.EscapeString(hbName), url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: fmt.Sprintf("عدم دریافت پالس: %s. %s", hbName, url)}
	}

	subject := fmt.Sprintf("⚠️ HEARTBEAT MISSED: %s", hbName)
	content := fmt.Sprintf(`
    <h2 style="color: #ea580c; margin-top: 0;">⚠️ Expected Heartbeat Missed</h2>
    <p>Heartbeat monitor <strong>%s</strong> did not receive an expected ping before its deadline.</p>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #ea580c; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Heartbeats</a>
    </div>
  `, html.EscapeString(hbName), url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: fmt.Sprintf("HEARTBEAT MISSED: %s. %s", hbName, url)}
}

func renderHeartbeatRecovered(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	hbName := getString(p, "heartbeat_name", "Heartbeat Monitor")
	url := fmt.Sprintf("%s/heartbeats", appBaseURL)

	if isFA {
		subject := fmt.Sprintf("✅ بازگشت پالس: %s دریافت شد", hbName)
		content := fmt.Sprintf(`
      <h2 style="color: #16a34a; margin-top: 0;">✅ پالس سیستم مجدداً دریافت شد</h2>
      <p>پالس سیستم <strong>%s</strong> با موفقیت دریافت و وضعیت به حالت سالم تغییر یافت.</p>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #16a34a; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">مشاهده پالس</a>
      </div>
    `, html.EscapeString(hbName), url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: fmt.Sprintf("بازگشت پالس: %s. %s", hbName, url)}
	}

	subject := fmt.Sprintf("✅ HEARTBEAT RECOVERED: %s", hbName)
	content := fmt.Sprintf(`
    <h2 style="color: #16a34a; margin-top: 0;">✅ Heartbeat Ping Received</h2>
    <p>Heartbeat <strong>%s</strong> received a ping and has returned to healthy state.</p>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #16a34a; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Heartbeats</a>
    </div>
  `, html.EscapeString(hbName), url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: fmt.Sprintf("HEARTBEAT RECOVERED: %s. %s", hbName, url)}
}

func renderReportReady(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	reportTitle := getString(p, "report_title", "Monthly Maintenance Report")
	reportID := getString(p, "report_id", "")
	url := fmt.Sprintf("%s/reports/%s", appBaseURL, reportID)

	if isFA {
		subject := fmt.Sprintf("📊 گزارش جدید آماده شد: %s", reportTitle)
		content := fmt.Sprintf(`
      <h2 style="color: #2563eb; margin-top: 0;">📊 گزارش نگهداری و پایش آماده مشاهده است</h2>
      <p>گزارش <strong>%s</strong> با موفقیت تولید شد و آماده دانلود و بررسی است.</p>
      <div style="margin-top: 24px;">
        <a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">مشاهده گزارش</a>
      </div>
    `, html.EscapeString(reportTitle), url)
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: fmt.Sprintf("گزارش آماده شد: %s. %s", reportTitle, url)}
	}

	subject := fmt.Sprintf("📊 Report Ready: %s", reportTitle)
	content := fmt.Sprintf(`
    <h2 style="color: #2563eb; margin-top: 0;">📊 Maintenance Report Generated</h2>
    <p>Report <strong>%s</strong> is ready for review and download.</p>
    <div style="margin-top: 24px;">
      <a href="%s" style="background-color: #2563eb; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">View Report</a>
    </div>
  `, html.EscapeString(reportTitle), url)
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: fmt.Sprintf("Report Ready: %s. %s", reportTitle, url)}
}

func renderGenericTest(p map[string]interface{}, appBaseURL string, isFA bool) RenderedEmail {
	if isFA {
		subject := "🔔 پیام آزمایشی اعلان ClientPulse"
		content := `
      <h2 style="color: #0284c7; margin-top: 0;">🔔 تست دریافت اعلان</h2>
      <p>این یک پیام آزمایشی برای تایید صحت اتصال ایمیل به سامانه <strong>ClientPulse</strong> است.</p>
      <p>کانال اعلان شما به درستی متصل شده و آماده دریافت هشدارهای خرابی و پایش است.</p>
    `
		return RenderedEmail{Subject: subject, HTMLBody: baseHTML("rtl", "fa", subject, content), TextBody: "پیام آزمایشی اعلان ClientPulse با موفقیت تحویل شد."}
	}

	subject := "🔔 ClientPulse Notification Test"
	content := `
    <h2 style="color: #0284c7; margin-top: 0;">🔔 Notification Channel Test</h2>
    <p>This is a test notification confirming that your email channel is properly configured with <strong>ClientPulse</strong>.</p>
    <p>Your destination is active and ready to deliver real-time incident and monitoring alerts.</p>
  `
	return RenderedEmail{Subject: subject, HTMLBody: baseHTML("ltr", "en", subject, content), TextBody: "ClientPulse test notification delivered successfully."}
}

