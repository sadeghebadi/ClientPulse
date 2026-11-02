package telegram

import (
	"fmt"
	"html"
	"strings"

	"clientpulse/services/api/internal/notification"
)

// RenderTelegramMessage formats a rich, localized message for Telegram (HTML mode).
func RenderTelegramMessage(ctx *notification.NotificationContext) string {
	if ctx == nil {
		return "🔔 <b>ClientPulse Notification</b>"
	}

	isFA := strings.ToLower(ctx.Locale) == "fa"

	var msg string
	switch ctx.EventType {
	case "incident.opened":
		msg = renderIncidentOpened(ctx, isFA)
	case "incident.resolved":
		msg = renderIncidentResolved(ctx, isFA)
	case "incident.acknowledged":
		msg = renderIncidentAcknowledged(ctx, isFA)
	case "heartbeat.missed":
		msg = renderHeartbeatMissed(ctx, isFA)
	case "heartbeat.recovered":
		msg = renderHeartbeatRecovered(ctx, isFA)
	case "report.ready":
		msg = renderReportReady(ctx, isFA)
	default:
		msg = renderTestNotification(ctx, isFA)
	}

	// Telegram message length limit safeguard (4096 bytes max)
	if len(msg) > 4000 {
		msg = msg[:3990] + "\n..."
	}
	return msg
}

func renderIncidentOpened(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	asset := html.EscapeString(ctx.AssetName)
	monitor := html.EscapeString(ctx.MonitorName)
	target := html.EscapeString(ctx.SanitizedTarget)
	incRef := html.EscapeString(ctx.IncidentRef)
	reason := html.EscapeString(ctx.FailureReason)
	if reason == "" {
		reason = html.EscapeString(ctx.FailureCategory)
	}

	detectedAt := ""
	if ctx.FirstFailureAt != nil {
		detectedAt = notification.FormatTimestamp(*ctx.FirstFailureAt, ctx.Timezone, isFA)
	} else if ctx.ConfirmedAt != nil {
		detectedAt = notification.FormatTimestamp(*ctx.ConfirmedAt, ctx.Timezone, isFA)
	}

	durationStr := notification.FormatDuration(ctx.DurationSeconds, isFA)
	intervalStr := notification.FormatDuration(int64(ctx.CheckIntervalSeconds), isFA)

	statusLine := ""
	if ctx.HTTPStatusCode > 0 {
		if isFA {
			statusLine = fmt.Sprintf("وضعیت HTTP: <code>%s</code>", notification.ToPersianDigits(fmt.Sprintf("%d", ctx.HTTPStatusCode)))
		} else {
			statusLine = fmt.Sprintf("HTTP Status: <code>%d</code>", ctx.HTTPStatusCode)
		}
		if reason != "" {
			statusLine += fmt.Sprintf(" (%s)", reason)
		}
	} else if reason != "" {
		if isFA {
			statusLine = fmt.Sprintf("علت خطا: <code>%s</code>", reason)
		} else {
			statusLine = fmt.Sprintf("Error: <code>%s</code>", reason)
		}
	}

	if isFA {
		downtimeLabel := "مدت قطعی جاری"
		ongoingBadge := "در حال پیگیری"
		if ctx.IsAlreadyResolved {
			downtimeLabel = "مجموع مدت قطعی تشخیصدادهشده"
			ongoingBadge = "بازیابی‌شده"
		}

		lines := []string{
			"🚨 <b>وبسایت از دسترس خارج شد</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("دارایی: <b>%s</b>", asset),
			fmt.Sprintf("مانیتور: <b>%s</b>", monitor),
		}
		if target != "" {
			lines = append(lines, fmt.Sprintf("آدرس هدف: <code>%s</code>", target))
		}
		lines = append(lines, "")
		if detectedAt != "" {
			lines = append(lines, fmt.Sprintf("زمان تشخیص: %s", detectedAt))
		}
		lines = append(lines, fmt.Sprintf("%s: <b>%s (%s)</b>", downtimeLabel, durationStr, ongoingBadge))
		lines = append(lines, "")
		if statusLine != "" {
			lines = append(lines, statusLine)
		}
		lines = append(lines, fmt.Sprintf("بازه بررسی: %s", intervalStr))
		if incRef != "" {
			lines = append(lines, fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده و بررسی در کلاینت‌پالس</a>", ctx.ViewURL))
		} else {
			lines = append(lines, "کلاینت‌پالس • سامانه پایش پایداری و نگهداری")
		}
		return strings.Join(lines, "\n")
	}

	// English
	downtimeLabel := "Downtime"
	ongoingBadge := "ONGOING"
	if ctx.IsAlreadyResolved {
		downtimeLabel = "Total Detected Downtime"
		ongoingBadge = "RESOLVED"
	}

	lines := []string{
		"🚨 <b>WEBSITE DOWN</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Asset: <b>%s</b>", asset),
		fmt.Sprintf("Monitor: <b>%s</b>", monitor),
	}
	if target != "" {
		lines = append(lines, fmt.Sprintf("Target: <code>%s</code>", target))
	}
	lines = append(lines, "")
	if detectedAt != "" {
		lines = append(lines, fmt.Sprintf("Detected at: %s", detectedAt))
	}
	lines = append(lines, fmt.Sprintf("%s: <b>%s (%s)</b>", downtimeLabel, durationStr, ongoingBadge))
	lines = append(lines, "")
	if statusLine != "" {
		lines = append(lines, statusLine)
	}
	lines = append(lines, fmt.Sprintf("Check Interval: %s", intervalStr))
	if incRef != "" {
		lines = append(lines, fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Incident in ClientPulse</a>", ctx.ViewURL))
	} else {
		lines = append(lines, "ClientPulse")
	}
	return strings.Join(lines, "\n")
}

func renderIncidentResolved(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	asset := html.EscapeString(ctx.AssetName)
	monitor := html.EscapeString(ctx.MonitorName)
	target := html.EscapeString(ctx.SanitizedTarget)
	incRef := html.EscapeString(ctx.IncidentRef)
	recStatus := html.EscapeString(ctx.RecoveryStatus)
	if recStatus == "" {
		recStatus = "HTTP 200 OK"
	}

	downSince := ""
	if ctx.FirstFailureAt != nil {
		downSince = notification.FormatTimestamp(*ctx.FirstFailureAt, ctx.Timezone, isFA)
	}
	recoveredAt := ""
	if ctx.ResolvedAt != nil {
		recoveredAt = notification.FormatTimestamp(*ctx.ResolvedAt, ctx.Timezone, isFA)
	}
	totalDowntime := notification.FormatDuration(ctx.DurationSeconds, isFA)

	if isFA {
		lines := []string{
			"✅ <b>وبسایت بازیابی شد</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("دارایی: <b>%s</b>", asset),
			fmt.Sprintf("مانیتور: <b>%s</b>", monitor),
		}
		if target != "" {
			lines = append(lines, fmt.Sprintf("آدرس هدف: <code>%s</code>", target))
		}
		lines = append(lines, "")
		if downSince != "" {
			lines = append(lines, fmt.Sprintf("شروع قطعی از: %s", downSince))
		}
		if recoveredAt != "" {
			lines = append(lines, fmt.Sprintf("زمان بازیابی: %s", recoveredAt))
		}
		lines = append(lines, fmt.Sprintf("مجموع مدت قطعی تشخیصدادهشده: <b>%s</b>", totalDowntime))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("وضعیت بازیابی: <code>%s</code>", recStatus))
		if incRef != "" {
			lines = append(lines, fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده گزارش رفع حادثه در کلاینت‌پالس</a>", ctx.ViewURL))
		} else {
			lines = append(lines, "کلاینت‌پالس")
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		"✅ <b>WEBSITE RECOVERED</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Asset: <b>%s</b>", asset),
		fmt.Sprintf("Monitor: <b>%s</b>", monitor),
	}
	if target != "" {
		lines = append(lines, fmt.Sprintf("Target: <code>%s</code>", target))
	}
	lines = append(lines, "")
	if downSince != "" {
		lines = append(lines, fmt.Sprintf("Down since: %s", downSince))
	}
	if recoveredAt != "" {
		lines = append(lines, fmt.Sprintf("Recovered at: %s", recoveredAt))
	}
	lines = append(lines, fmt.Sprintf("Total Detected Downtime: <b>%s</b>", totalDowntime))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Recovery Status: <code>%s</code>", recStatus))
	if incRef != "" {
		lines = append(lines, fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Resolution in ClientPulse</a>", ctx.ViewURL))
	} else {
		lines = append(lines, "ClientPulse")
	}
	return strings.Join(lines, "\n")
}

func renderIncidentAcknowledged(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	asset := html.EscapeString(ctx.AssetName)
	monitor := html.EscapeString(ctx.MonitorName)
	incRef := html.EscapeString(ctx.IncidentRef)
	ackBy := html.EscapeString(ctx.AcknowledgedByName)

	ackAt := ""
	if ctx.AcknowledgedAt != nil {
		ackAt = notification.FormatTimestamp(*ctx.AcknowledgedAt, ctx.Timezone, isFA)
	}
	tta := notification.FormatDuration(ctx.TimeToAcknowledgeSeconds, isFA)
	downtimeSoFar := notification.FormatDuration(ctx.DurationSeconds, isFA)

	if isFA {
		lines := []string{
			"👁️ <b>تایید مشاهده حادثه</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("دارایی: <b>%s</b>", asset),
			fmt.Sprintf("مانیتور: <b>%s</b>", monitor),
			"",
			fmt.Sprintf("تایید شده توسط: <b>%s</b>", ackBy),
			fmt.Sprintf("زمان تایید: %s", ackAt),
			fmt.Sprintf("زمان تا تایید: %s", tta),
			"وضعیت حادثه: <b>در حال پیگیری (تایید شده توسط تیم پشتیبانی)</b>",
			fmt.Sprintf("مدت قطعی تا این لحظه: %s", downtimeSoFar),
		}
		if incRef != "" {
			lines = append(lines, "", fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده جزئیات حادثه</a>", ctx.ViewURL))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		"👁️ <b>INCIDENT ACKNOWLEDGED</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Asset: <b>%s</b>", asset),
		fmt.Sprintf("Monitor: <b>%s</b>", monitor),
		"",
		fmt.Sprintf("Acknowledged by: <b>%s</b>", ackBy),
		fmt.Sprintf("Acknowledged at: %s", ackAt),
		fmt.Sprintf("Time to Acknowledge: %s", tta),
		"Current Status: <b>Ongoing (Acknowledged by On-Call)</b>",
		fmt.Sprintf("Downtime so far: %s", downtimeSoFar),
	}
	if incRef != "" {
		lines = append(lines, "", fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Incident Details</a>", ctx.ViewURL))
	}
	return strings.Join(lines, "\n")
}

func renderHeartbeatMissed(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	asset := html.EscapeString(ctx.AssetName)
	hbName := html.EscapeString(ctx.HeartbeatName)
	incRef := html.EscapeString(ctx.IncidentRef)

	lastPingStr := "Never received"
	if isFA {
		lastPingStr = "هیچ سیگنالی دریافت نشده است"
	}
	if ctx.LastPingAt != nil {
		lastPingStr = notification.FormatTimestamp(*ctx.LastPingAt, ctx.Timezone, isFA)
	}

	expInterval := notification.FormatDuration(int64(ctx.ExpectedIntervalSeconds), isFA)
	gracePeriod := notification.FormatDuration(int64(ctx.GracePeriodSeconds), isFA)
	overdueStr := notification.FormatDuration(ctx.OverdueSeconds, isFA)

	if isFA {
		lines := []string{
			"⚠️ <b>عدم دریافت سیگنال سلامت (Heartbeat)</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("دارایی: <b>%s</b>", asset),
			fmt.Sprintf("سیگنال سلامت: <b>%s</b>", hbName),
			"",
			fmt.Sprintf("آخرین دریافت سیگنال: %s", lastPingStr),
			fmt.Sprintf("بازه مورد انتظار: %s", expInterval),
			fmt.Sprintf("مهلت فرجه: %s", gracePeriod),
			fmt.Sprintf("مدت تأخیر: <b>%s</b>", overdueStr),
			"",
			"وضعیت: <b>هیچ سیگنال جدیدی در موعد مقرر دریافت نشد.</b>",
			"<i>نکته: این مورد لزوماً به معنای قطعی وب‌سایت نیست.</i>",
		}
		if incRef != "" {
			lines = append(lines, "", fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">بررسی سیگنال‌های سلامت در کلاینت‌پالس</a>", ctx.ViewURL))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		"⚠️ <b>HEARTBEAT MISSED</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Asset: <b>%s</b>", asset),
		fmt.Sprintf("Heartbeat: <b>%s</b>", hbName),
		"",
		fmt.Sprintf("Last Ping: %s", lastPingStr),
		fmt.Sprintf("Expected Interval: %s", expInterval),
		fmt.Sprintf("Grace Period: %s", gracePeriod),
		fmt.Sprintf("Overdue By: <b>%s</b>", overdueStr),
		"",
		"Status: <b>No recent heartbeat received.</b>",
		"<i>Note: This does not necessarily indicate website downtime.</i>",
	}
	if incRef != "" {
		lines = append(lines, "", fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">Check Heartbeats in ClientPulse</a>", ctx.ViewURL))
	}
	return strings.Join(lines, "\n")
}

func renderHeartbeatRecovered(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	asset := html.EscapeString(ctx.AssetName)
	hbName := html.EscapeString(ctx.HeartbeatName)
	incRef := html.EscapeString(ctx.IncidentRef)

	recAt := ""
	if ctx.ResolvedAt != nil {
		recAt = notification.FormatTimestamp(*ctx.ResolvedAt, ctx.Timezone, isFA)
	}
	overdueTotal := notification.FormatDuration(ctx.OverdueSeconds, isFA)

	if isFA {
		lines := []string{
			"✅ <b>دریافت سیگنال مجدداً برقرار شد</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("دارایی: <b>%s</b>", asset),
			fmt.Sprintf("سیگنال سلامت: <b>%s</b>", hbName),
			"",
			fmt.Sprintf("زمان برقراری مجدد: %s", recAt),
			fmt.Sprintf("مدت زمان تأخیر سیگنال: <b>%s</b>", overdueTotal),
			"",
			"وضعیت: <b>سیگنال سلامت دریافت شد و سرویس در وضعیت پایدار است.</b>",
		}
		if incRef != "" {
			lines = append(lines, "", fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده وضعیت در کلاینت‌پالس</a>", ctx.ViewURL))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		"✅ <b>HEARTBEAT RECOVERED</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Asset: <b>%s</b>", asset),
		fmt.Sprintf("Heartbeat: <b>%s</b>", hbName),
		"",
		fmt.Sprintf("Recovered at: %s", recAt),
		fmt.Sprintf("Heartbeat Overdue Duration: <b>%s</b>", overdueTotal),
		"",
		"Status: <b>Heartbeat healthy and restored.</b>",
	}
	if incRef != "" {
		lines = append(lines, "", fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">Check Heartbeats in ClientPulse</a>", ctx.ViewURL))
	}
	return strings.Join(lines, "\n")
}

func renderReportReady(ctx *notification.NotificationContext, isFA bool) string {
	client := html.EscapeString(ctx.ClientName)
	title := html.EscapeString(ctx.ReportTitle)

	if isFA {
		lines := []string{
			"📊 <b>گزارش نگهداری و پایش آماده است</b>",
			"",
			fmt.Sprintf("مشتری: <b>%s</b>", client),
			fmt.Sprintf("عنوان گزارش: <b>%s</b>", title),
		}
		if ctx.PeriodStart != "" && ctx.PeriodEnd != "" {
			lines = append(lines, fmt.Sprintf("بازه زمانی: %s تا %s", notification.ToPersianDigits(ctx.PeriodStart), notification.ToPersianDigits(ctx.PeriodEnd)))
		}
		lines = append(lines, "", "گزارش ماهانه پایش و عملکرد سیستم آماده دانلود است.")
		if ctx.ViewURL != "" {
			lines = append(lines, "", fmt.Sprintf("👉 <a href=\"%s\">مشاهده و دانلود گزارش</a>", ctx.ViewURL))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{
		"📊 <b>CARE & MAINTENANCE REPORT READY</b>",
		"",
		fmt.Sprintf("Client: <b>%s</b>", client),
		fmt.Sprintf("Report: <b>%s</b>", title),
	}
	if ctx.PeriodStart != "" && ctx.PeriodEnd != "" {
		lines = append(lines, fmt.Sprintf("Period: %s to %s", ctx.PeriodStart, ctx.PeriodEnd))
	}
	lines = append(lines, "", "Your care and maintenance report is ready for download.")
	if ctx.ViewURL != "" {
		lines = append(lines, "", fmt.Sprintf("👉 <a href=\"%s\">View Report in ClientPulse</a>", ctx.ViewURL))
	}
	return strings.Join(lines, "\n")
}

func renderTestNotification(ctx *notification.NotificationContext, isFA bool) string {
	org := html.EscapeString(ctx.OrgName)

	if isFA {
		return fmt.Sprintf("🔔 <b>تست اعلانات تلگرام کلاینت‌پالس</b>\n\nسازمان: <b>%s</b>\n\nاین یک پیام آزمایشی است که تایید می‌کند کانال تلگرام شما فعال است و هشدارهای سیستم را با موفقیت دریافت می‌کند.\n\nکلاینت‌پالس • سامانه پایش پایداری و نگهداری", org)
	}
	return fmt.Sprintf("🔔 <b>ClientPulse Telegram Notification Test</b>\n\nOrganization: <b>%s</b>\n\nThis is a test notification confirming that your Telegram channel is active and receiving alerts.\n\nClientPulse • Agency Reliability & Maintenance Platform", org)
}

