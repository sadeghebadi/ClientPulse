package email

import (
	"fmt"
	"html"
	"strings"

	"clientpulse/services/api/internal/notification"
)

type RenderedEmail struct {
	Subject  string
	HTMLBody string
	TextBody string
}

// RenderNotificationEmail generates localized email content for an event from raw payload.
func RenderNotificationEmail(eventType, locale string, payload map[string]interface{}, appBaseURL string) RenderedEmail {
	ctx := notification.ParseContextFromPayload(eventType, payload, appBaseURL, locale)
	return RenderNotificationEmailFromContext(ctx, appBaseURL)
}

// RenderNotificationEmailFromContext formats a rich, localized email from typed context.
func RenderNotificationEmailFromContext(ctx *notification.NotificationContext, appBaseURL string) RenderedEmail {
	if ctx == nil {
		ctx = &notification.NotificationContext{
			EventType: "generic",
			Locale:    "en",
			OrgName:   "ClientPulse",
		}
	}
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	if ctx.ViewURL == "" {
		ctx.ViewURL = appBaseURL
	}

	isFA := strings.ToLower(ctx.Locale) == "fa"

	switch ctx.EventType {
	case "incident.opened":
		return renderIncidentOpened(ctx, isFA)
	case "incident.acknowledged":
		return renderIncidentAcknowledged(ctx, isFA)
	case "incident.resolved":
		return renderIncidentResolved(ctx, isFA)
	case "heartbeat.missed":
		return renderHeartbeatMissed(ctx, isFA)
	case "heartbeat.recovered":
		return renderHeartbeatRecovered(ctx, isFA)
	case "report.ready":
		return renderReportReady(ctx, isFA)
	default:
		return renderGenericTest(ctx, isFA)
	}
}

func baseHTML(dir, lang, title, contentHTML string) string {
	align := "left"
	if dir == "rtl" {
		align = "right"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin: 0; padding: 24px; background-color: #f8fafc; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; color: #0f172a; direction: %s; text-align: %s;">
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
</html>`, lang, dir, html.EscapeString(title), dir, align, align, contentHTML)
}

func tableRow(label, value string, isRTL bool) string {
	align := "left"
	if isRTL {
		align = "right"
	}
	return fmt.Sprintf(`<tr>
  <td style="padding: 8px 12px; color: #64748b; font-size: 13px; text-align: %s; width: 35%%; border-bottom: 1px solid #f1f5f9; vertical-align: top;">%s</td>
  <td style="padding: 8px 12px; color: #0f172a; font-size: 13px; font-weight: 500; text-align: %s; width: 65%%; border-bottom: 1px solid #f1f5f9; word-break: break-all;">%s</td>
</tr>`, align, label, align, value)
}

func formatTable(rows []string) string {
	return fmt.Sprintf(`<table style="width: 100%%; border-collapse: collapse; margin: 20px 0; background: #f8fafc; border: 1px solid #e2e8f0; border-radius: 6px; overflow: hidden;"><tbody>%s</tbody></table>`, strings.Join(rows, "\n"))
}

func actionButton(url, label, bgColor string) string {
	return fmt.Sprintf(`<div style="margin: 24px 0 16px 0;">
  <a href="%s" style="background-color: %s; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; font-size: 14px; display: inline-block;">%s</a>
</div>`, url, bgColor, html.EscapeString(label))
}

func renderIncidentOpened(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "Unknown Client"
	}
	monitorName := ctx.MonitorName
	if monitorName == "" {
		monitorName = "Website Monitor"
	}

	detectedAt := "Just now"
	if ctx.FirstFailureAt != nil {
		detectedAt = notification.FormatTimestamp(*ctx.FirstFailureAt, ctx.Timezone, isFA)
	} else if ctx.ConfirmedAt != nil {
		detectedAt = notification.FormatTimestamp(*ctx.ConfirmedAt, ctx.Timezone, isFA)
	}

	durationStr := notification.FormatDuration(ctx.DurationSeconds, isFA)
	intervalStr := notification.FormatDuration(int64(ctx.CheckIntervalSeconds), isFA)

	var subject, heading, resolvedBanner string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("🚨 بحرانی: [%s] سرویس %s از دسترس خارج شد (%s)", clientName, monitorName, ctx.IncidentRef)
		heading = fmt.Sprintf(`<h2 style="color: #dc2626; margin-top: 0; font-size: 20px;">🚨 حادثه جدید: سرویس از دسترس خارج شد</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">پایش خودکار عدم پاسخ‌دهی سرویس را تایید کرده و حادثه جدیدی ایجاد شده است.</p>`)

		if ctx.IsAlreadyResolved {
			resolvedBanner = fmt.Sprintf(`<div style="background: #f0fdf4; border: 1px solid #86efac; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #166534; font-size: 13px;">
  <strong>توجه:</strong> این حادثه پیش از ارسال اعلان برطرف شد (مدت کل اختلال: %s).
</div>`, durationStr)
		}

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("دارایی:", html.EscapeString(ctx.AssetName), true))
		rows = append(rows, tableRow("مانیتور:", fmt.Sprintf("%s (%s)", html.EscapeString(monitorName), html.EscapeString(ctx.MonitorType)), true))
		if ctx.SanitizedTarget != "" {
			rows = append(rows, tableRow("آدرس مقصد:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.SanitizedTarget)), true))
		}
		rows = append(rows, tableRow("زمان شناسایی:", detectedAt, true))
		if ctx.IsAlreadyResolved {
			rows = append(rows, tableRow("وضعیت فعلی:", fmt.Sprintf("<span style=\"color: #16a34a;\">برطرف شده (مدت قطعی: %s)</span>", durationStr), true))
		} else {
			rows = append(rows, tableRow("وضعیت فعلی:", fmt.Sprintf("<span style=\"color: #dc2626;\">ادامه‌دار (مدت: %s)</span>", durationStr), true))
		}
		if ctx.HTTPStatusCode > 0 {
			rows = append(rows, tableRow("کد وضعیت HTTP:", notification.ToPersianDigits(fmt.Sprintf("%d", ctx.HTTPStatusCode)), true))
		}
		rows = append(rows, tableRow("دسته خطا:", html.EscapeString(ctx.FailureCategory), true))
		if ctx.FailureReason != "" {
			rows = append(rows, tableRow("پیام خطا:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.FailureReason)), true))
		}
		rows = append(rows, tableRow("بازه بررسی:", intervalStr, true))
		rows = append(rows, tableRow("شناسه مرجع:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), true))

		btn := actionButton(ctx.ViewURL, "مشاهده و مدیریت حادثه در ClientPulse", "#dc2626")
		htmlBody := baseHTML("rtl", "fa", subject, heading+resolvedBanner+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("🚨 بحرانی: [%s] سرویس %s از دسترس خارج شد\n", clientName, monitorName))
		textBody.WriteString(fmt.Sprintf("مشتری: %s\nدارایی: %s\nمانیتور: %s\nآدرس: %s\nزمان: %s\nمدت اختلال: %s\nمرجع: %s\nبررسی: %s\n",
			clientName, ctx.AssetName, monitorName, ctx.SanitizedTarget, detectedAt, durationStr, ctx.IncidentRef, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("🚨 CRITICAL: [%s] %s is DOWN (%s)", clientName, monitorName, ctx.IncidentRef)
	heading = fmt.Sprintf(`<h2 style="color: #dc2626; margin-top: 0; font-size: 20px;">🚨 Incident Opened: Service Down</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">Automated health checks detected an outage and confirmed this incident.</p>`)

	if ctx.IsAlreadyResolved {
		resolvedBanner = fmt.Sprintf(`<div style="background: #f0fdf4; border: 1px solid #86efac; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #166534; font-size: 13px;">
  <strong>Notice:</strong> This incident was resolved before notification delivery (Total downtime: %s).
</div>`, durationStr)
	}

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Asset:", html.EscapeString(ctx.AssetName), false))
	rows = append(rows, tableRow("Monitor:", fmt.Sprintf("%s (%s)", html.EscapeString(monitorName), html.EscapeString(ctx.MonitorType)), false))
	if ctx.SanitizedTarget != "" {
		rows = append(rows, tableRow("Target URL:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.SanitizedTarget)), false))
	}
	rows = append(rows, tableRow("Detected At:", detectedAt, false))
	if ctx.IsAlreadyResolved {
		rows = append(rows, tableRow("Current Status:", fmt.Sprintf("<span style=\"color: #16a34a;\">RESOLVED (Total: %s)</span>", durationStr), false))
	} else {
		rows = append(rows, tableRow("Current Status:", fmt.Sprintf("<span style=\"color: #dc2626;\">DOWN (Ongoing for %s)</span>", durationStr), false))
	}
	if ctx.HTTPStatusCode > 0 {
		rows = append(rows, tableRow("HTTP Status:", fmt.Sprintf("%d", ctx.HTTPStatusCode), false))
	}
	rows = append(rows, tableRow("Category:", html.EscapeString(ctx.FailureCategory), false))
	if ctx.FailureReason != "" {
		rows = append(rows, tableRow("Reason:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.FailureReason)), false))
	}
	rows = append(rows, tableRow("Check Interval:", intervalStr, false))
	rows = append(rows, tableRow("Incident Ref:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), false))

	btn := actionButton(ctx.ViewURL, "View Incident in ClientPulse", "#dc2626")
	htmlBody := baseHTML("ltr", "en", subject, heading+resolvedBanner+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("🚨 CRITICAL: [%s] %s is DOWN\n", clientName, monitorName))
	textBody.WriteString(fmt.Sprintf("Client: %s\nAsset: %s\nMonitor: %s\nTarget: %s\nDetected: %s\nDowntime: %s\nRef: %s\nView: %s\n",
		clientName, ctx.AssetName, monitorName, ctx.SanitizedTarget, detectedAt, durationStr, ctx.IncidentRef, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderIncidentResolved(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "Unknown Client"
	}
	monitorName := ctx.MonitorName
	if monitorName == "" {
		monitorName = "Website Monitor"
	}

	downSince := "Unknown"
	if ctx.FirstFailureAt != nil {
		downSince = notification.FormatTimestamp(*ctx.FirstFailureAt, ctx.Timezone, isFA)
	}
	recoveredAt := "Just now"
	if ctx.ResolvedAt != nil {
		recoveredAt = notification.FormatTimestamp(*ctx.ResolvedAt, ctx.Timezone, isFA)
	}
	totalDowntime := notification.FormatDuration(ctx.DurationSeconds, isFA)

	var subject, heading string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("✅ رفع مشکل: [%s] سرویس %s به حالت عادی بازگشت (%s)", clientName, monitorName, ctx.IncidentRef)
		heading = fmt.Sprintf(`<h2 style="color: #16a34a; margin-top: 0; font-size: 20px;">✅ رفع مشکل سرویس: بازگشت به حالت عادی</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">بررسی‌های خودکار تایید کردند که سرویس مجدداً فعال و پاسخگو شده است.</p>`)

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("دارایی:", html.EscapeString(ctx.AssetName), true))
		rows = append(rows, tableRow("مانیتور:", html.EscapeString(monitorName), true))
		if ctx.SanitizedTarget != "" {
			rows = append(rows, tableRow("آدرس مقصد:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.SanitizedTarget)), true))
		}
		rows = append(rows, tableRow("شروع اختلال:", downSince, true))
		rows = append(rows, tableRow("زمان رفع مشکل:", recoveredAt, true))
		rows = append(rows, tableRow("مجموع مدت قطعی:", fmt.Sprintf("<strong>%s</strong>", totalDowntime), true))
		rows = append(rows, tableRow("وضعیت بازیابی:", html.EscapeString(ctx.RecoveryStatus), true))
		rows = append(rows, tableRow("شناسه مرجع:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), true))

		btn := actionButton(ctx.ViewURL, "مشاهده جزئیات رفع حادثه", "#16a34a")
		htmlBody := baseHTML("rtl", "fa", subject, heading+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("✅ رفع مشکل: [%s] %s به حالت عادی بازگشت\n", clientName, monitorName))
		textBody.WriteString(fmt.Sprintf("مشتری: %s\nمدت قطعی: %s\nزمان رفع: %s\nمرجع: %s\nبررسی: %s\n",
			clientName, totalDowntime, recoveredAt, ctx.IncidentRef, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("✅ RESOLVED: [%s] %s is back UP (%s)", clientName, monitorName, ctx.IncidentRef)
	heading = fmt.Sprintf(`<h2 style="color: #16a34a; margin-top: 0; font-size: 20px;">✅ Incident Resolved: Service Recovered</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">Automated health checks confirmed that the service has fully recovered.</p>`)

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Asset:", html.EscapeString(ctx.AssetName), false))
	rows = append(rows, tableRow("Monitor:", html.EscapeString(monitorName), false))
	if ctx.SanitizedTarget != "" {
		rows = append(rows, tableRow("Target URL:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.SanitizedTarget)), false))
	}
	rows = append(rows, tableRow("Down Since:", downSince, false))
	rows = append(rows, tableRow("Recovered At:", recoveredAt, false))
	rows = append(rows, tableRow("Total Downtime:", fmt.Sprintf("<strong>%s</strong>", totalDowntime), false))
	rows = append(rows, tableRow("Recovery Status:", html.EscapeString(ctx.RecoveryStatus), false))
	rows = append(rows, tableRow("Incident Ref:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), false))

	btn := actionButton(ctx.ViewURL, "View Incident Details", "#16a34a")
	htmlBody := baseHTML("ltr", "en", subject, heading+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("✅ RESOLVED: [%s] %s is back UP\n", clientName, monitorName))
	textBody.WriteString(fmt.Sprintf("Client: %s\nDowntime: %s\nRecovered: %s\nRef: %s\nView: %s\n",
		clientName, totalDowntime, recoveredAt, ctx.IncidentRef, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderIncidentAcknowledged(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "Unknown Client"
	}
	serviceName := ctx.MonitorName
	if serviceName == "" {
		serviceName = "Service"
	}

	ackAt := "Just now"
	if ctx.AcknowledgedAt != nil {
		ackAt = notification.FormatTimestamp(*ctx.AcknowledgedAt, ctx.Timezone, isFA)
	}
	tta := notification.FormatDuration(ctx.TimeToAcknowledgeSeconds, isFA)
	downtimeSoFar := notification.FormatDuration(ctx.DurationSeconds, isFA)

	ackBy := ctx.AcknowledgedByName
	if ctx.AcknowledgedByEmail != "" {
		ackBy = fmt.Sprintf("%s (%s)", ctx.AcknowledgedByName, ctx.AcknowledgedByEmail)
	}

	var subject, heading, banner string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("👁️ تایید بررسی: [%s] حادثه %s در دست بررسی است", clientName, ctx.IncidentRef)
		heading = fmt.Sprintf(`<h2 style="color: #d97706; margin-top: 0; font-size: 20px;">👁️ تایید و آغاز بررسی حادثه</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">یکی از اعضای تیم پشتیبانی مسئولیت رسیدگی به این حادثه را پذیرفت.</p>`)

		banner = `<div style="background: #fffbeb; border: 1px solid #fde68a; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #92400e; font-size: 13px;">
  ⚠️ <strong>توجه:</strong> حادثه در حال بررسی است اما سرویس هنوز به طور کامل رفع مشکل نشده است.
</div>`

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("سرویس:", html.EscapeString(serviceName), true))
		rows = append(rows, tableRow("بررسی‌کننده:", html.EscapeString(ackBy), true))
		rows = append(rows, tableRow("زمان تایید:", ackAt, true))
		rows = append(rows, tableRow("زمان تا پاسخ (TTA):", tta, true))
		rows = append(rows, tableRow("مدت قطعی تا این لحظه:", downtimeSoFar, true))
		rows = append(rows, tableRow("شناسه مرجع:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), true))

		btn := actionButton(ctx.ViewURL, "مشاهده جزئیات و پیگیری حادثه", "#d97706")
		htmlBody := baseHTML("rtl", "fa", subject, heading+banner+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("👁️ تایید بررسی: [%s] حادثه %s\n", clientName, ctx.IncidentRef))
		textBody.WriteString(fmt.Sprintf("تاییدکننده: %s\nزمان تایید: %s\nمدت قطعی تا کنون: %s\nپیگیری: %s\n",
			ackBy, ackAt, downtimeSoFar, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("👁️ ACKNOWLEDGED: [%s] Incident %s is under investigation", clientName, ctx.IncidentRef)
	heading = fmt.Sprintf(`<h2 style="color: #d97706; margin-top: 0; font-size: 20px;">👁️ Incident Acknowledged</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">An on-call engineer has acknowledged this incident and started triage.</p>`)

	banner = `<div style="background: #fffbeb; border: 1px solid #fde68a; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #92400e; font-size: 13px;">
  ⚠️ <strong>Notice:</strong> The service is currently under active investigation and is <strong>NOT</strong> yet resolved.
</div>`

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Service:", html.EscapeString(serviceName), false))
	rows = append(rows, tableRow("Acknowledged By:", html.EscapeString(ackBy), false))
	rows = append(rows, tableRow("Acknowledged At:", ackAt, false))
	rows = append(rows, tableRow("Time to Acknowledge:", tta, false))
	rows = append(rows, tableRow("Downtime So Far:", downtimeSoFar, false))
	rows = append(rows, tableRow("Incident Ref:", fmt.Sprintf("<code>%s</code>", html.EscapeString(ctx.IncidentRef)), false))

	btn := actionButton(ctx.ViewURL, "View Incident in ClientPulse", "#d97706")
	htmlBody := baseHTML("ltr", "en", subject, heading+banner+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("👁️ ACKNOWLEDGED: [%s] Incident %s\n", clientName, ctx.IncidentRef))
	textBody.WriteString(fmt.Sprintf("Acknowledged By: %s\nTime: %s\nDowntime So Far: %s\nView: %s\n",
		ackBy, ackAt, downtimeSoFar, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderHeartbeatMissed(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "Unknown Client"
	}
	hbName := ctx.HeartbeatName
	if hbName == "" {
		hbName = "Heartbeat Monitor"
	}

	lastPingStr := "Never received"
	if isFA {
		lastPingStr = "تاکنون دریافت نشده"
	}
	if ctx.LastPingAt != nil {
		lastPingStr = notification.FormatTimestamp(*ctx.LastPingAt, ctx.Timezone, isFA)
	}

	expInterval := notification.FormatDuration(int64(ctx.ExpectedIntervalSeconds), isFA)
	gracePeriod := notification.FormatDuration(int64(ctx.GracePeriodSeconds), isFA)
	overdueStr := notification.FormatDuration(ctx.OverdueSeconds, isFA)

	var subject, heading, banner string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("⚠️ عدم دریافت پالس: [%s] کار %s با تاخیر مواجه شد", clientName, hbName)
		heading = fmt.Sprintf(`<h2 style="color: #ea580c; margin-top: 0; font-size: 20px;">⚠️ عدم دریافت پالس سیستم در موعد مقرر</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">پالس زمان‌بندی شده برای کار پس‌زمینه تا پایان مهلت مجاز دریافت نشد.</p>`)

		banner = `<div style="background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #1e40af; font-size: 13px;">
  ℹ️ <strong>راهنما:</strong> این هشدار مربوط به وظایف زمان‌بندی‌شده، پشتیبان‌گیری، یا پردازش‌های پس‌زمینه است و نشان‌دهنده قطعی وب‌سایت نیست.
</div>`

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("دارایی:", html.EscapeString(ctx.AssetName), true))
		rows = append(rows, tableRow("نام پالس:", html.EscapeString(hbName), true))
		rows = append(rows, tableRow("آخرین دریافت موفق:", lastPingStr, true))
		rows = append(rows, tableRow("بازه مورد انتظار:", expInterval, true))
		rows = append(rows, tableRow("مهلت ارفاقی:", gracePeriod, true))
		rows = append(rows, tableRow("میزان تاخیر:", fmt.Sprintf("<span style=\"color: #ea580c;\"><strong>%s</strong></span>", overdueStr), true))

		btn := actionButton(ctx.ViewURL, "بررسی وضعیت پالس‌ها در پنل", "#ea580c")
		htmlBody := baseHTML("rtl", "fa", subject, heading+banner+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("⚠️ عدم دریافت پالس: [%s] %s\n", clientName, hbName))
		textBody.WriteString(fmt.Sprintf("مشتری: %s\nمیزان تاخیر: %s\nآخرین پالس: %s\nبررسی: %s\n",
			clientName, overdueStr, lastPingStr, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("⚠️ HEARTBEAT MISSED: [%s] %s is overdue", clientName, hbName)
	heading = fmt.Sprintf(`<h2 style="color: #ea580c; margin-top: 0; font-size: 20px;">⚠️ Expected Heartbeat Missed</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">A scheduled background task did not ping before its deadline.</p>`)

	banner = `<div style="background: #eff6ff; border: 1px solid #bfdbfe; border-radius: 6px; padding: 12px; margin-bottom: 16px; color: #1e40af; font-size: 13px;">
  ℹ️ <strong>Clarification:</strong> This alert indicates a scheduled cron job, background worker, or backup task failed to ping before deadline. It does not represent an HTTP website downtime.
</div>`

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Asset:", html.EscapeString(ctx.AssetName), false))
	rows = append(rows, tableRow("Heartbeat Name:", html.EscapeString(hbName), false))
	rows = append(rows, tableRow("Last Received Ping:", lastPingStr, false))
	rows = append(rows, tableRow("Expected Interval:", expInterval, false))
	rows = append(rows, tableRow("Grace Period:", gracePeriod, false))
	rows = append(rows, tableRow("Overdue By:", fmt.Sprintf("<span style=\"color: #ea580c;\"><strong>%s</strong></span>", overdueStr), false))

	btn := actionButton(ctx.ViewURL, "View Heartbeats in ClientPulse", "#ea580c")
	htmlBody := baseHTML("ltr", "en", subject, heading+banner+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("⚠️ HEARTBEAT MISSED: [%s] %s is overdue\n", clientName, hbName))
	textBody.WriteString(fmt.Sprintf("Client: %s\nOverdue By: %s\nLast Ping: %s\nView: %s\n",
		clientName, overdueStr, lastPingStr, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderHeartbeatRecovered(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "Unknown Client"
	}
	hbName := ctx.HeartbeatName
	if hbName == "" {
		hbName = "Heartbeat Monitor"
	}

	recAt := "Just now"
	if ctx.ResolvedAt != nil {
		recAt = notification.FormatTimestamp(*ctx.ResolvedAt, ctx.Timezone, isFA)
	}
	overdueTotal := notification.FormatDuration(ctx.OverdueSeconds, isFA)

	var subject, heading string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("✅ بازگشت پالس: [%s] پالس %s مجدداً دریافت شد", clientName, hbName)
		heading = fmt.Sprintf(`<h2 style="color: #16a34a; margin-top: 0; font-size: 20px;">✅ پالس سیستم مجدداً دریافت شد</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">پالس وظیفه پس‌زمینه دریافت شده و وضعیت به حالت سالم و فعال بازگشت.</p>`)

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("دارایی:", html.EscapeString(ctx.AssetName), true))
		rows = append(rows, tableRow("نام پالس:", html.EscapeString(hbName), true))
		rows = append(rows, tableRow("زمان بازیابی:", recAt, true))
		if ctx.OverdueSeconds > 0 {
			rows = append(rows, tableRow("مدت تاخیر:", overdueTotal, true))
		}
		rows = append(rows, tableRow("وضعیت:", "<span style=\"color: #16a34a;\">سالم و فعال</span>", true))

		btn := actionButton(ctx.ViewURL, "بررسی وضعیت پالس‌ها", "#16a34a")
		htmlBody := baseHTML("rtl", "fa", subject, heading+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("✅ بازگشت پالس: [%s] %s\n", clientName, hbName))
		textBody.WriteString(fmt.Sprintf("مشتری: %s\nزمان: %s\nوضعیت: سالم\nبررسی: %s\n", clientName, recAt, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("✅ HEARTBEAT RECOVERED: [%s] %s ping received", clientName, hbName)
	heading = fmt.Sprintf(`<h2 style="color: #16a34a; margin-top: 0; font-size: 20px;">✅ Heartbeat Ping Received</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">A ping was received and this background task has returned to a healthy state.</p>`)

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Asset:", html.EscapeString(ctx.AssetName), false))
	rows = append(rows, tableRow("Heartbeat Name:", html.EscapeString(hbName), false))
	rows = append(rows, tableRow("Recovered At:", recAt, false))
	if ctx.OverdueSeconds > 0 {
		rows = append(rows, tableRow("Total Overdue:", overdueTotal, false))
	}
	rows = append(rows, tableRow("Status:", "<span style=\"color: #16a34a;\">Healthy / Operational</span>", false))

	btn := actionButton(ctx.ViewURL, "View Heartbeats", "#16a34a")
	htmlBody := baseHTML("ltr", "en", subject, heading+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("✅ HEARTBEAT RECOVERED: [%s] %s\n", clientName, hbName))
	textBody.WriteString(fmt.Sprintf("Client: %s\nRecovered: %s\nStatus: Healthy\nView: %s\n", clientName, recAt, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderReportReady(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	clientName := ctx.ClientName
	if clientName == "" {
		clientName = "All Clients"
	}
	reportTitle := ctx.ReportTitle
	if reportTitle == "" {
		reportTitle = "Care & Maintenance Report"
	}

	var subject, heading string
	var rows []string
	var textBody strings.Builder

	if isFA {
		subject = fmt.Sprintf("📊 گزارش جدید: [%s] %s آماده شد", clientName, reportTitle)
		heading = fmt.Sprintf(`<h2 style="color: #2563eb; margin-top: 0; font-size: 20px;">📊 گزارش نگهداری و پایش آماده مشاهده است</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">گزارش جدید با موفقیت تولید شد و آماده دانلود و بررسی است.</p>`)

		rows = append(rows, tableRow("مشتری:", html.EscapeString(clientName), true))
		rows = append(rows, tableRow("عنوان گزارش:", html.EscapeString(reportTitle), true))
		if ctx.PeriodStart != "" && ctx.PeriodEnd != "" {
			rows = append(rows, tableRow("دوره گزارش:", fmt.Sprintf("%s تا %s", ctx.PeriodStart, ctx.PeriodEnd), true))
		}
		rows = append(rows, tableRow("سازمان:", html.EscapeString(ctx.OrgName), true))

		btn := actionButton(ctx.ViewURL, "مشاهده و دانلود گزارش", "#2563eb")
		htmlBody := baseHTML("rtl", "fa", subject, heading+formatTable(rows)+btn)

		textBody.WriteString(fmt.Sprintf("📊 گزارش آماده شد: [%s] %s\n", clientName, reportTitle))
		textBody.WriteString(fmt.Sprintf("مشتری: %s\nگزارش: %s\nمشاهده: %s\n", clientName, reportTitle, ctx.ViewURL))

		return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
	}

	subject = fmt.Sprintf("📊 Report Ready: [%s] %s", clientName, reportTitle)
	heading = fmt.Sprintf(`<h2 style="color: #2563eb; margin-top: 0; font-size: 20px;">📊 Maintenance Report Generated</h2>
<p style="margin: 4px 0 16px 0; color: #334155;">Your care and maintenance report is ready for review and download.</p>`)

	rows = append(rows, tableRow("Client:", html.EscapeString(clientName), false))
	rows = append(rows, tableRow("Report Title:", html.EscapeString(reportTitle), false))
	if ctx.PeriodStart != "" && ctx.PeriodEnd != "" {
		rows = append(rows, tableRow("Report Period:", fmt.Sprintf("%s to %s", ctx.PeriodStart, ctx.PeriodEnd), false))
	}
	rows = append(rows, tableRow("Organization:", html.EscapeString(ctx.OrgName), false))

	btn := actionButton(ctx.ViewURL, "View / Download Report", "#2563eb")
	htmlBody := baseHTML("ltr", "en", subject, heading+formatTable(rows)+btn)

	textBody.WriteString(fmt.Sprintf("📊 Report Ready: [%s] %s\n", clientName, reportTitle))
	textBody.WriteString(fmt.Sprintf("Client: %s\nReport: %s\nView: %s\n", clientName, reportTitle, ctx.ViewURL))

	return RenderedEmail{Subject: subject, HTMLBody: htmlBody, TextBody: textBody.String()}
}

func renderGenericTest(ctx *notification.NotificationContext, isFA bool) RenderedEmail {
	if isFA {
		subject := "🔔 پیام آزمایشی اعلان ClientPulse"
		content := `
      <h2 style="color: #0284c7; margin-top: 0; font-size: 20px;">🔔 تست دریافت اعلان</h2>
      <p style="color: #334155;">این یک پیام آزمایشی برای تایید صحت اتصال ایمیل به سامانه <strong>ClientPulse</strong> است.</p>
      <p style="color: #334155;">کانال اعلان شما به درستی متصل شده و آماده دریافت هشدارهای خرابی و پایش است.</p>
    `
		btn := actionButton(ctx.ViewURL, "ورود به داشبورد ClientPulse", "#0284c7")
		return RenderedEmail{
			Subject:  subject,
			HTMLBody: baseHTML("rtl", "fa", subject, content+btn),
			TextBody: "پیام آزمایشی اعلان ClientPulse با موفقیت تحویل شد.",
		}
	}

	subject := "🔔 ClientPulse Notification Test"
	content := `
    <h2 style="color: #0284c7; margin-top: 0; font-size: 20px;">🔔 Notification Channel Test</h2>
    <p style="color: #334155;">This is a test notification confirming that your email channel is properly configured with <strong>ClientPulse</strong>.</p>
    <p style="color: #334155;">Your destination is active and ready to deliver real-time incident and monitoring alerts.</p>
  `
	btn := actionButton(ctx.ViewURL, "Go to ClientPulse Dashboard", "#0284c7")
	return RenderedEmail{
		Subject:  subject,
		HTMLBody: baseHTML("ltr", "en", subject, content+btn),
		TextBody: "ClientPulse test notification delivered successfully.",
	}
}

// RenderEmailVerification renders the ownership verification email for an alert destination.
func RenderEmailVerification(token, appBaseURL, locale string) RenderedEmail {
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	appBaseURL = strings.TrimRight(appBaseURL, "/")
	verifyURL := fmt.Sprintf("%s/settings/notifications?verify_email=%s", appBaseURL, token)

	isFA := strings.ToLower(locale) == "fa"
	if isFA {
		subject := "🔐 تایید آدرس ایمیل برای هشدارهای ClientPulse"
		content := fmt.Sprintf(`
      <h2 style="color: #0284c7; margin-top: 0; font-size: 20px;">🔐 تایید مالکیت آدرس ایمیل</h2>
      <p style="color: #334155;">درخواستی برای دریافت هشدارهای نظارتی سازمان شما به این نشانی ایمیل در سامانه <strong>ClientPulse</strong> ثبت شده است.</p>
      <p style="color: #334155;">جهت فعال‌سازی این مقصد و شروع دریافت اعلان‌ها، روی دکمه زیر کلیک کنید (این پیوند تا ۲۴ ساعت معتبر است):</p>
      <div style="margin: 24px 0;">
        <a href="%s" style="background-color: #0284c7; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">تایید و فعال‌سازی ایمیل</a>
      </div>
      <p style="font-size: 13px; color: #64748b;">اگر دکمه بالا کار نمی‌کند، پیوند زیر را در مرورگر کپی کنید:<br><code style="word-break: break-all;">%s</code></p>
      <p style="font-size: 12px; color: #94a3b8; margin-top: 24px;">اگر شما این درخواست را ارسال نکرده‌اید، نیازی به انجام کاری نیست؛ این ایمیل فعال نخواهد شد.</p>
    `, verifyURL, verifyURL)
		return RenderedEmail{
			Subject:  subject,
			HTMLBody: baseHTML("rtl", "fa", subject, content),
			TextBody: fmt.Sprintf("تایید ایمیل ClientPulse: برای فعال‌سازی نشانی ایمیل به این پیوند مراجعه نمایید: %s", verifyURL),
		}
	}

	subject := "🔐 Verify your email address for ClientPulse alerts"
	content := fmt.Sprintf(`
      <h2 style="color: #0284c7; margin-top: 0; font-size: 20px;">🔐 Verify Alert Destination Email</h2>
      <p style="color: #334155;">A request was made to add this email address as an alert destination in <strong>ClientPulse</strong>.</p>
      <p style="color: #334155;">To confirm ownership and activate alert delivery, click the button below (link expires in 24 hours):</p>
      <div style="margin: 24px 0;">
        <a href="%s" style="background-color: #0284c7; color: #ffffff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 600; display: inline-block;">Verify & Activate Email</a>
      </div>
      <p style="font-size: 13px; color: #64748b;">If the button above does not work, copy and paste this link into your browser:<br><code style="word-break: break-all;">%s</code></p>
      <p style="font-size: 12px; color: #94a3b8; margin-top: 24px;">If you did not request this notification destination, you can safely ignore this email; no alerts will be sent.</p>
    `, verifyURL, verifyURL)
	return RenderedEmail{
		Subject:  subject,
		HTMLBody: baseHTML("ltr", "en", subject, content),
		TextBody: fmt.Sprintf("Verify ClientPulse Email: Visit %s to activate your alert destination.", verifyURL),
	}
}
