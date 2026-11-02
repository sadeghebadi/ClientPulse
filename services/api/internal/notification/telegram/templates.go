package telegram

import (
	"fmt"
	"html"
	"strings"

	"clientpulse/services/api/internal/notification"
)

type formatDetailLevel int

const (
	formatFull formatDetailLevel = iota
	formatCompact
	formatMinimal
)

// RenderTelegramMessage formats a rich, localized message for Telegram (HTML mode).
// It validates length against Telegram Bot API limits, preserves valid HTML formatting,
// and gracefully degrades through compact and minimal fallbacks if content is long.
func RenderTelegramMessage(ctx *notification.NotificationContext) string {
	if ctx == nil {
		return "🔔 <b>ClientPulse Notification</b>"
	}

	isFA := strings.ToLower(ctx.Locale) == "fa"

	// 1. Attempt full detailed message
	fullMsg := renderEvent(ctx, isFA, formatFull)
	if TelegramMessageLength(fullMsg) <= TelegramSafeMarginChars {
		return EnsureValidTelegramHTML(fullMsg, TelegramMaxMessageChars)
	}

	// 2. Attempt compact message (omits low-priority optional fields: target URL, failure reason, check interval)
	compactMsg := renderEvent(ctx, isFA, formatCompact)
	if TelegramMessageLength(compactMsg) <= TelegramSafeMarginChars {
		return EnsureValidTelegramHTML(compactMsg, TelegramMaxMessageChars)
	}

	// 3. Fallback to minimal message preserving critical context only
	minimalMsg := renderEvent(ctx, isFA, formatMinimal)
	return EnsureValidTelegramHTML(minimalMsg, TelegramMaxMessageChars)
}

// RenderTelegramMessagePlainText formats an alert in plain text mode (no HTML entities),
// suitable for environments or fallbacks where Telegram HTML parse mode is disabled.
func RenderTelegramMessagePlainText(ctx *notification.NotificationContext) string {
	htmlMsg := RenderTelegramMessage(ctx)
	plain := StripTelegramHTML(htmlMsg)
	return SafeTruncateRunes(plain, TelegramMaxMessageChars, "...")
}

func renderEvent(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	switch ctx.EventType {
	case "incident.opened":
		return renderIncidentOpened(ctx, isFA, level)
	case "incident.resolved":
		return renderIncidentResolved(ctx, isFA, level)
	case "incident.acknowledged":
		return renderIncidentAcknowledged(ctx, isFA, level)
	case "heartbeat.missed":
		return renderHeartbeatMissed(ctx, isFA, level)
	case "heartbeat.recovered":
		return renderHeartbeatRecovered(ctx, isFA, level)
	case "report.ready":
		return renderReportReady(ctx, isFA, level)
	default:
		return renderTestNotification(ctx, isFA)
	}
}

func getSafeNames(ctx *notification.NotificationContext, level formatDetailLevel) (client, asset, monitor, target, incRef string) {
	maxName := 70
	maxTarget := 100
	if level == formatCompact {
		maxName = 50
		maxTarget = 70
	} else if level == formatMinimal {
		maxName = 35
		maxTarget = 50
	}

	client = html.EscapeString(SafeTruncateRunes(ctx.ClientName, maxName, "..."))
	if client == "" {
		client = "Unknown Client"
	}
	asset = html.EscapeString(SafeTruncateRunes(ctx.AssetName, maxName, "..."))
	monitor = html.EscapeString(SafeTruncateRunes(ctx.MonitorName, maxName, "..."))
	if monitor == "" {
		monitor = "Website Monitor"
	}
	target = html.EscapeString(TruncateTargetURL(ctx.SanitizedTarget, maxTarget))
	incRef = html.EscapeString(SafeTruncateRunes(ctx.IncidentRef, 30, ""))
	return
}

func renderIncidentOpened(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	client, asset, monitor, target, incRef := getSafeNames(ctx, level)

	reason := html.EscapeString(SafeTruncateRunes(ctx.FailureReason, 200, "..."))
	if reason == "" {
		reason = html.EscapeString(SafeTruncateRunes(ctx.FailureCategory, 50, "..."))
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
	if level == formatFull {
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
		if level == formatFull && target != "" {
			lines = append(lines, fmt.Sprintf("آدرس هدف: <code>%s</code>", target))
		}
		lines = append(lines, "")
		if detectedAt != "" {
			lines = append(lines, fmt.Sprintf("زمان تشخیص: %s", detectedAt))
		}
		lines = append(lines, fmt.Sprintf("%s: <b>%s (%s)</b>", downtimeLabel, durationStr, ongoingBadge))
		if statusLine != "" {
			lines = append(lines, "", statusLine)
		}
		if level == formatFull {
			lines = append(lines, fmt.Sprintf("بازه بررسی: %s", intervalStr))
		}
		if incRef != "" {
			lines = append(lines, fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده و بررسی در کلاینت‌پالس</a>", html.EscapeString(ctx.ViewURL)))
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
	if level == formatFull && target != "" {
		lines = append(lines, fmt.Sprintf("Target: <code>%s</code>", target))
	}
	lines = append(lines, "")
	if detectedAt != "" {
		lines = append(lines, fmt.Sprintf("Detected at: %s", detectedAt))
	}
	lines = append(lines, fmt.Sprintf("%s: <b>%s (%s)</b>", downtimeLabel, durationStr, ongoingBadge))
	if statusLine != "" {
		lines = append(lines, "", statusLine)
	}
	if level == formatFull {
		lines = append(lines, fmt.Sprintf("Check Interval: %s", intervalStr))
	}
	if incRef != "" {
		lines = append(lines, fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Incident in ClientPulse</a>", html.EscapeString(ctx.ViewURL)))
	} else {
		lines = append(lines, "ClientPulse")
	}
	return strings.Join(lines, "\n")
}

func renderIncidentResolved(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	client, asset, monitor, target, incRef := getSafeNames(ctx, level)

	recStatus := html.EscapeString(SafeTruncateRunes(ctx.RecoveryStatus, 100, "..."))
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
		if level == formatFull && target != "" {
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
		if level == formatFull {
			lines = append(lines, "", fmt.Sprintf("وضعیت بازیابی: <code>%s</code>", recStatus))
		}
		if incRef != "" {
			lines = append(lines, fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده گزارش رفع حادثه در کلاینت‌پالس</a>", html.EscapeString(ctx.ViewURL)))
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
	if level == formatFull && target != "" {
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
	if level == formatFull {
		lines = append(lines, "", fmt.Sprintf("Recovery Status: <code>%s</code>", recStatus))
	}
	if incRef != "" {
		lines = append(lines, fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Resolution in ClientPulse</a>", html.EscapeString(ctx.ViewURL)))
	} else {
		lines = append(lines, "ClientPulse")
	}
	return strings.Join(lines, "\n")
}

func renderIncidentAcknowledged(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	client, asset, monitor, _, incRef := getSafeNames(ctx, level)
	ackBy := html.EscapeString(SafeTruncateRunes(ctx.AcknowledgedByName, 50, "..."))
	if ackBy == "" {
		ackBy = "On-call Engineer"
	}

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
		}
		if level == formatFull {
			lines = append(lines, fmt.Sprintf("زمان تا تایید: %s", tta))
		}
		lines = append(lines,
			"وضعیت حادثه: <b>در حال پیگیری (تایید شده توسط تیم پشتیبانی)</b>",
			fmt.Sprintf("مدت قطعی تا این لحظه: %s", downtimeSoFar),
		)
		if incRef != "" {
			lines = append(lines, "", fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده جزئیات حادثه</a>", html.EscapeString(ctx.ViewURL)))
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
	}
	if level == formatFull {
		lines = append(lines, fmt.Sprintf("Time to Acknowledge: %s", tta))
	}
	lines = append(lines,
		"Current Status: <b>Ongoing (Acknowledged by On-Call)</b>",
		fmt.Sprintf("Downtime so far: %s", downtimeSoFar),
	)
	if incRef != "" {
		lines = append(lines, "", fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">View Incident Details</a>", html.EscapeString(ctx.ViewURL)))
	}
	return strings.Join(lines, "\n")
}

func renderHeartbeatMissed(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	client, asset, _, _, incRef := getSafeNames(ctx, level)
	hbName := html.EscapeString(SafeTruncateRunes(ctx.HeartbeatName, 70, "..."))
	if hbName == "" {
		hbName = "Heartbeat Monitor"
	}

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
		}
		if level == formatFull {
			lines = append(lines,
				fmt.Sprintf("بازه مورد انتظار: %s", expInterval),
				fmt.Sprintf("مهلت فرجه: %s", gracePeriod),
			)
		}
		lines = append(lines,
			fmt.Sprintf("مدت تأخیر: <b>%s</b>", overdueStr),
			"",
			"وضعیت: <b>هیچ سیگنال جدیدی در موعد مقرر دریافت نشد.</b>",
		)
		if level == formatFull {
			lines = append(lines, "<i>نکته: این مورد لزوماً به معنای قطعی وب‌سایت نیست.</i>")
		}
		if incRef != "" {
			lines = append(lines, "", fmt.Sprintf("شناسه حادثه: <code>%s</code>", incRef))
		}
		lines = append(lines, "")
		if ctx.ViewURL != "" {
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">بررسی سیگنال‌های سلامت در کلاینت‌پالس</a>", html.EscapeString(ctx.ViewURL)))
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
	}
	if level == formatFull {
		lines = append(lines,
			fmt.Sprintf("Expected Interval: %s", expInterval),
			fmt.Sprintf("Grace Period: %s", gracePeriod),
		)
	}
	lines = append(lines,
		fmt.Sprintf("Overdue By: <b>%s</b>", overdueStr),
		"",
		"Status: <b>No recent heartbeat received.</b>",
	)
	if level == formatFull {
		lines = append(lines, "<i>Note: This does not necessarily indicate website downtime.</i>")
	}
	if incRef != "" {
		lines = append(lines, "", fmt.Sprintf("Incident: <code>%s</code>", incRef))
	}
	lines = append(lines, "")
	if ctx.ViewURL != "" {
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">Check Heartbeats in ClientPulse</a>", html.EscapeString(ctx.ViewURL)))
	}
	return strings.Join(lines, "\n")
}

func renderHeartbeatRecovered(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	client, asset, _, _, incRef := getSafeNames(ctx, level)
	hbName := html.EscapeString(SafeTruncateRunes(ctx.HeartbeatName, 70, "..."))
	if hbName == "" {
		hbName = "Heartbeat Monitor"
	}

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
			lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">مشاهده وضعیت در کلاینت‌پالس</a>", html.EscapeString(ctx.ViewURL)))
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
		lines = append(lines, fmt.Sprintf("👉 <a href=\"%s\">Check Heartbeats in ClientPulse</a>", html.EscapeString(ctx.ViewURL)))
	}
	return strings.Join(lines, "\n")
}

func renderReportReady(ctx *notification.NotificationContext, isFA bool, level formatDetailLevel) string {
	maxName := 70
	if level == formatMinimal {
		maxName = 35
	}
	client := html.EscapeString(SafeTruncateRunes(ctx.ClientName, maxName, "..."))
	title := html.EscapeString(SafeTruncateRunes(ctx.ReportTitle, maxName, "..."))

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
			lines = append(lines, "", fmt.Sprintf("👉 <a href=\"%s\">مشاهده و دانلود گزارش</a>", html.EscapeString(ctx.ViewURL)))
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
		lines = append(lines, "", fmt.Sprintf("👉 <a href=\"%s\">View Report in ClientPulse</a>", html.EscapeString(ctx.ViewURL)))
	}
	return strings.Join(lines, "\n")
}

func renderTestNotification(ctx *notification.NotificationContext, isFA bool) string {
	org := html.EscapeString(SafeTruncateRunes(ctx.OrgName, 60, "..."))

	if isFA {
		return fmt.Sprintf("🔔 <b>تست اعلانات تلگرام کلاینت‌پالس</b>\n\nسازمان: <b>%s</b>\n\nاین یک پیام آزمایشی است که تایید می‌کند کانال تلگرام شما فعال است و هشدارهای سیستم را با موفقیت دریافت می‌کند.\n\nکلاینت‌پالس • سامانه پایش پایداری و نگهداری", org)
	}
	return fmt.Sprintf("🔔 <b>ClientPulse Telegram Notification Test</b>\n\nOrganization: <b>%s</b>\n\nThis is a test notification confirming that your Telegram channel is active and receiving alerts.\n\nClientPulse • Agency Reliability & Maintenance Platform", org)
}
