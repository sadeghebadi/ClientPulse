"use client";

import React, { useState, useEffect } from "react";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import {
  Bell,
  Send,
  Mail,
  CheckCircle2,
  AlertTriangle,
  RefreshCw,
  Plus,
  Trash2,
  Sliders,
  ExternalLink,
  Copy,
  Check,
  Clock,
  Users,
} from "lucide-react";

interface Destination {
  id: string;
  organization_id: string;
  destination_type: "telegram_private" | "telegram_group" | "email";
  name: string;
  telegram_chat_id?: number;
  telegram_user_id?: number;
  telegram_username?: string;
  email_address?: string;
  status: "active" | "disconnected" | "pending" | "pending_approval";
  is_enabled: boolean;
  created_at: string;
  last_verified_at?: string;
  verified_at?: string;
  verification_sent_at?: string;
  pending_group_approval?: boolean;
  pending_group_title?: string;
  preferences?: Record<string, boolean>;
}

interface DeliveryRecord {
  id: string;
  destination_id: string;
  destination_type: string;
  recipient: string;
  event_type: string;
  status: "queued" | "sending" | "sent" | "retrying" | "failed" | "cancelled" | "uncertain";
  retry_count: number;
  max_retries: number;
  error_message?: string;
  sent_at?: string;
  created_at: string;
}

export default function NotificationSettingsPage() {
  const { role, activeOrg } = useAuth();
  const { t } = useI18n();

  const [activeTab, setActiveTab] = useState<"destinations" | "history">("destinations");
  const [destinations, setDestinations] = useState<Destination[]>([]);
  const [botUsername, setBotUsername] = useState<string>("");
  const [loading, setLoading] = useState<boolean>(true);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Delivery History
  const [history, setHistory] = useState<DeliveryRecord[]>([]);
  const [loadingHistory, setLoadingHistory] = useState<boolean>(false);

  // Connect Telegram Private Modal
  const [connectModalOpen, setConnectModalOpen] = useState(false);
  const [deepLink, setDeepLink] = useState<string>("");

  // Connect Telegram Group Modal
  const [groupModalOpen, setGroupModalOpen] = useState(false);
  const [groupCode, setGroupCode] = useState<string>("");
  const [codeCopied, setCodeCopied] = useState(false);

  // Add Email Modal
  const [emailModalOpen, setEmailModalOpen] = useState(false);
  const [emailInput, setEmailInput] = useState("");
  const [emailNameInput, setEmailNameInput] = useState("");
  const [savingEmail, setSavingEmail] = useState(false);

  // Manual Verify Email Token Modal
  const [verifyTokenModalOpen, setVerifyTokenModalOpen] = useState(false);
  const [manualTokenInput, setManualTokenInput] = useState("");
  const [verifyingToken, setVerifyingToken] = useState(false);

  // Preferences Modal
  const [prefModalOpen, setPrefModalOpen] = useState(false);
  const [editingDest, setEditingDest] = useState<Destination | null>(null);
  const [editPrefs, setEditPrefs] = useState<Record<string, boolean>>({});
  const [savingPrefs, setSavingPrefs] = useState(false);

  // Test notification progress
  const [testingDestId, setTestingDestId] = useState<string | null>(null);

  const canManage = role === "owner" || role === "admin";

  const fetchDestinations = async () => {
    try {
      setLoading(true);
      const data = await apiFetch<{
        destinations?: Destination[];
        bot_username?: string;
      }>("/api/v1/notifications/destinations");

      setDestinations(data?.destinations || []);
      if (data?.bot_username) {
        setBotUsername(data.bot_username);
      }
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    } finally {
      setLoading(false);
    }
  };

  const fetchHistory = async () => {
    try {
      setLoadingHistory(true);
      const data = await apiFetch<{
        deliveries?: DeliveryRecord[];
      }>("/api/v1/notifications/history?limit=30");

      setHistory(data?.deliveries || []);
    } catch (err: any) {
      console.error("Failed fetching history:", err);
    } finally {
      setLoadingHistory(false);
    }
  };

  // Auto-verify if url contains verify_email token
  useEffect(() => {
    if (typeof window !== "undefined") {
      const params = new URLSearchParams(window.location.search);
      const verifyToken = params.get("verify_email");
      if (verifyToken) {
        apiFetch("/api/v1/notifications/destinations/verify-email", {
          method: "POST",
          body: JSON.stringify({ token: verifyToken }),
        })
          .then(() => {
            setMessage({ type: "success", text: "Email destination successfully verified and activated!" });
            fetchDestinations();
          })
          .catch((err) => {
            setMessage({ type: "error", text: humanizeApiError(err) });
          });
      }
    }
  }, []);

  useEffect(() => {
    fetchDestinations();
    if (activeTab === "history") {
      fetchHistory();
    }
  }, [activeOrg, activeTab]);

  // Connect Private Telegram
  const handleOpenConnectTelegram = async () => {
    try {
      setConnectModalOpen(true);
      const data = await apiFetch<{
        token: string;
        deep_link: string;
        bot_username: string;
        expires_at: string;
      }>("/api/v1/notifications/destinations/telegram/connect-token", {
        method: "POST",
        body: JSON.stringify({ label: "Telegram User" }),
      });

      if (data) {
        setDeepLink(data.deep_link);
        if (data.bot_username) setBotUsername(data.bot_username);
      }
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
      setConnectModalOpen(false);
    }
  };

  // Connect Group Telegram
  const handleOpenConnectGroup = async () => {
    try {
      setGroupModalOpen(true);
      setCodeCopied(false);
      const data = await apiFetch<{
        code: string;
        bot_username: string;
        expires_at: string;
      }>("/api/v1/notifications/destinations/telegram/group-token", {
        method: "POST",
        body: JSON.stringify({ label: "Alerts Group" }),
      });

      if (data) {
        setGroupCode(data.code);
        if (data.bot_username) setBotUsername(data.bot_username);
      }
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
      setGroupModalOpen(false);
    }
  };

  // Add Email Destination
  const handleCreateEmail = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!emailInput) return;
    try {
      setSavingEmail(true);
      await apiFetch("/api/v1/notifications/destinations/email", {
        method: "POST",
        body: JSON.stringify({
          email: emailInput,
          name: emailNameInput || emailInput,
          preferences: {
            "incident.opened": true,
            "incident.acknowledged": true,
            "incident.resolved": true,
            "heartbeat.missed": true,
            "heartbeat.recovered": true,
            "report.ready": true,
          },
        }),
      });

      setMessage({ type: "success", text: "Email destination added. A verification email has been sent to confirm ownership." });
      setEmailModalOpen(false);
      setEmailInput("");
      setEmailNameInput("");
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    } finally {
      setSavingEmail(false);
    }
  };

  // Toggle Destination Enabled State
  const handleToggleEnabled = async (dest: Destination) => {
    try {
      const nextState = !dest.is_enabled;
      await apiFetch(`/api/v1/notifications/destinations/${dest.id}`, {
        method: "PATCH",
        body: JSON.stringify({ is_enabled: nextState }),
      });

      setDestinations((prev) =>
        prev.map((d) => (d.id === dest.id ? { ...d, is_enabled: nextState } : d))
      );
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    }
  };

  // Open Preferences Modal
  const handleOpenPreferences = (dest: Destination) => {
    setEditingDest(dest);
    setEditPrefs(dest.preferences || {});
    setPrefModalOpen(true);
  };

  // Save Preferences
  const handleSavePreferences = async () => {
    if (!editingDest) return;
    try {
      setSavingPrefs(true);
      await apiFetch(`/api/v1/notifications/destinations/${editingDest.id}`, {
        method: "PATCH",
        body: JSON.stringify({ preferences: editPrefs }),
      });

      setMessage({ type: "success", text: t.savePreferences + " ✓" });
      setPrefModalOpen(false);
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    } finally {
      setSavingPrefs(false);
    }
  };

  // Send Test Notification
  const handleTestNotification = async (destId: string) => {
    try {
      setTestingDestId(destId);
      setMessage(null);
      const data = await apiFetch<{ success: boolean; message: string }>(
        `/api/v1/notifications/destinations/${destId}/test`,
        { method: "POST" }
      );

      if (data?.success) {
        setMessage({ type: "success", text: t.testSentSuccess });
      } else {
        setMessage({ type: "error", text: data?.message || t.testSentFailed });
      }
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    } finally {
      setTestingDestId(null);
    }
  };

  // Disconnect Destination
  const handleDisconnect = async (destId: string) => {
    if (!confirm(t.disconnectConfirm)) return;
    try {
      await apiFetch(`/api/v1/notifications/destinations/${destId}`, {
        method: "DELETE",
      });
      setMessage({ type: "success", text: "Destination disconnected successfully" });
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    }
  };

  // Approve Pending Telegram Group Connection
  const handleApproveGroup = async (destId: string) => {
    try {
      await apiFetch(`/api/v1/notifications/destinations/${destId}/approve-group`, {
        method: "POST",
      });
      setMessage({ type: "success", text: "Telegram group connection approved and activated!" });
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    }
  };

  // Cancel Pending Telegram Group Connection
  const handleCancelGroup = async (destId: string) => {
    try {
      await apiFetch(`/api/v1/notifications/destinations/${destId}/cancel-group`, {
        method: "POST",
      });
      setMessage({ type: "success", text: "Pending group connection cancelled." });
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    }
  };

  // Resend Email Ownership Verification
  const handleResendEmailVerification = async (destId: string) => {
    try {
      await apiFetch(`/api/v1/notifications/destinations/${destId}/resend-verification`, {
        method: "POST",
      });
      setMessage({ type: "success", text: "Verification email resent successfully. Please check your inbox." });
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    }
  };

  // Manual Verify Email Token
  const handleManualVerifyToken = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!manualTokenInput.trim()) return;
    try {
      setVerifyingToken(true);
      await apiFetch("/api/v1/notifications/destinations/verify-email", {
        method: "POST",
        body: JSON.stringify({ token: manualTokenInput.trim() }),
      });
      setMessage({ type: "success", text: "Email destination verified and activated successfully!" });
      setVerifyTokenModalOpen(false);
      setManualTokenInput("");
      fetchDestinations();
    } catch (err: any) {
      setMessage({ type: "error", text: humanizeApiError(err) });
    } finally {
      setVerifyingToken(false);
    }
  };

  const telegramDestinations = destinations.filter(
    (d) => d.destination_type === "telegram_private" || d.destination_type === "telegram_group"
  );
  const emailDestinations = destinations.filter((d) => d.destination_type === "email");

  const eventCategoryList = [
    { key: "incident.opened", label: t.incidentOpened },
    { key: "incident.acknowledged", label: t.incidentAcknowledged },
    { key: "incident.resolved", label: t.incidentResolved },
    { key: "heartbeat.missed", label: t.heartbeatMissed },
    { key: "heartbeat.recovered", label: t.heartbeatRecovered },
    { key: "report.ready", label: t.reportReady },
  ];

  return (
    <AppLayout>
      <div className="max-w-6xl mx-auto space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-slate-200 pb-5">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
              <Bell className="w-6 h-6 text-blue-600" />
              {t.notificationSettings}
            </h1>
            <p className="text-slate-500 text-sm mt-1">{t.notificationSettingsDesc}</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => setActiveTab("destinations")}
              className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${activeTab === "destinations"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 hover:bg-slate-100 border border-slate-200"
                }`}
            >
              {t.telegramDestinations} & {t.emailDestinations}
            </button>
            <button
              onClick={() => setActiveTab("history")}
              className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${activeTab === "history"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-slate-700 hover:bg-slate-100 border border-slate-200"
                }`}
            >
              {t.deliveryHistory}
            </button>
          </div>
        </div>

        {/* Global Feedback Alert */}
        {message && (
          <div
            className={`p-4 rounded-lg flex items-center justify-between text-sm ${message.type === "success"
                ? "bg-emerald-50 text-emerald-800 border border-emerald-200"
                : "bg-rose-50 text-rose-800 border border-rose-200"
              }`}
          >
            <div className="flex items-center gap-2">
              {message.type === "success" ? (
                <CheckCircle2 className="w-5 h-5 text-emerald-600 flex-shrink-0" />
              ) : (
                <AlertTriangle className="w-5 h-5 text-rose-600 flex-shrink-0" />
              )}
              <span>{message.text}</span>
            </div>
            <button
              onClick={() => setMessage(null)}
              className="text-slate-400 hover:text-slate-600 text-xs font-semibold"
            >
              ✕
            </button>
          </div>
        )}

        {/* Tab 1: Notification Destinations */}
        {activeTab === "destinations" && (
          <div className="space-y-8">
            {/* Telegram Section */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
              <div className="p-5 border-b border-slate-200 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 bg-slate-50">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-blue-100 text-blue-600 rounded-lg">
                    <Send className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-base font-semibold text-slate-900">
                      {t.telegramDestinations}
                    </h2>
                    <p className="text-xs text-slate-500">
                      {botUsername ? `@${botUsername}` : "Central ClientPulse Bot"}
                    </p>
                  </div>
                </div>

                {canManage && (
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleOpenConnectGroup}
                      className="flex items-center gap-1.5"
                    >
                      <Users className="w-4 h-4" />
                      {t.addTelegramGroup}
                    </Button>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={handleOpenConnectTelegram}
                      className="flex items-center gap-1.5 bg-blue-600 hover:bg-blue-700 text-white"
                    >
                      <Plus className="w-4 h-4" />
                      {t.connectTelegram}
                    </Button>
                  </div>
                )}
              </div>

              {loading ? (
                <div className="p-8 text-center text-slate-400 text-sm">
                  <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2" />
                  Loading destinations...
                </div>
              ) : telegramDestinations.length === 0 ? (
                <div className="p-8 text-center">
                  <Send className="w-10 h-10 text-slate-300 mx-auto mb-2" />
                  <p className="text-slate-500 text-sm">{t.noDestinationsFound}</p>
                </div>
              ) : (
                <div className="divide-y divide-slate-100">
                  {telegramDestinations.map((d) => (
                    <div
                      key={d.id}
                      className="p-4 sm:p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:bg-slate-50/50 transition-colors"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold text-slate-900 text-sm sm:text-base">
                            {d.name}
                          </span>
                          <Badge variant="outline" className="text-xs">
                            {d.destination_type === "telegram_private" ? "Private Chat" : "Group"}
                          </Badge>
                          {d.telegram_username && (
                            <span className="text-xs text-blue-600 font-mono">
                              @{d.telegram_username}
                            </span>
                          )}
                          {d.status === "pending_approval" ? (
                            <Badge variant="warning" className="text-xs">
                              Pending Approval
                            </Badge>
                          ) : (
                            <Badge
                              variant={d.is_enabled ? "success" : "secondary"}
                              className="text-xs"
                            >
                              {d.is_enabled ? t.enabled : t.disabled}
                            </Badge>
                          )}
                        </div>
                        <div className="text-xs text-slate-400 font-mono">
                          {t.chatId}: {d.telegram_chat_id || "N/A"}
                        </div>

                        {d.status === "pending_approval" && (
                          <div className="mt-2 p-2.5 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                            <span className="flex items-center gap-1.5">
                              <AlertTriangle className="w-4 h-4 text-amber-600 flex-shrink-0" />
                              Group connected by unlinked account. Admin approval required.
                            </span>
                            {canManage && (
                              <div className="flex items-center gap-1.5 flex-shrink-0">
                                <button
                                  onClick={() => handleApproveGroup(d.id)}
                                  className="px-2.5 py-1 bg-emerald-600 hover:bg-emerald-700 text-white rounded text-xs font-semibold shadow-xs"
                                >
                                  Approve Group
                                </button>
                                <button
                                  onClick={() => handleCancelGroup(d.id)}
                                  className="px-2 py-1 bg-slate-200 hover:bg-slate-300 text-slate-700 rounded text-xs"
                                >
                                  Cancel
                                </button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>

                      <div className="flex items-center gap-2 flex-wrap">
                        {canManage && (
                          <>
                            <button
                              onClick={() => handleToggleEnabled(d)}
                              className={`px-3 py-1.5 rounded text-xs font-medium border transition-colors ${d.is_enabled
                                  ? "bg-slate-100 text-slate-700 hover:bg-slate-200 border-slate-300"
                                  : "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border-emerald-300"
                                }`}
                            >
                              {d.is_enabled ? t.disabled : t.enabled}
                            </button>

                            <button
                              onClick={() => handleOpenPreferences(d)}
                              className="px-3 py-1.5 rounded text-xs font-medium border border-slate-300 hover:bg-slate-100 flex items-center gap-1 text-slate-700"
                            >
                              <Sliders className="w-3.5 h-3.5" />
                              {t.eventPreferences}
                            </button>
                          </>
                        )}

                        <button
                          onClick={() => handleTestNotification(d.id)}
                          disabled={testingDestId === d.id}
                          className="px-3 py-1.5 rounded text-xs font-medium bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200 flex items-center gap-1 transition-colors"
                        >
                          <Send className="w-3.5 h-3.5" />
                          {testingDestId === d.id ? "Sending..." : t.testNotification}
                        </button>

                        {canManage && (
                          <button
                            onClick={() => handleDisconnect(d.id)}
                            className="p-1.5 text-rose-600 hover:bg-rose-50 rounded transition-colors"
                            title={t.disconnect}
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Email Section */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
              <div className="p-5 border-b border-slate-200 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 bg-slate-50">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-purple-100 text-purple-600 rounded-lg">
                    <Mail className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-base font-semibold text-slate-900">
                      {t.emailDestinations}
                    </h2>
                    <p className="text-xs text-slate-500">SMTP Notification Delivery</p>
                  </div>
                </div>

                {canManage && (
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => setEmailModalOpen(true)}
                    className="flex items-center gap-1.5 bg-purple-600 hover:bg-purple-700 text-white"
                  >
                    <Plus className="w-4 h-4" />
                    {t.addEmailDestination}
                  </Button>
                )}
              </div>

              {loading ? (
                <div className="p-8 text-center text-slate-400 text-sm">Loading...</div>
              ) : emailDestinations.length === 0 ? (
                <div className="p-8 text-center">
                  <Mail className="w-10 h-10 text-slate-300 mx-auto mb-2" />
                  <p className="text-slate-500 text-sm">{t.noDestinationsFound}</p>
                </div>
              ) : (
                <div className="divide-y divide-slate-100">
                  {emailDestinations.map((d) => (
                    <div
                      key={d.id}
                      className="p-4 sm:p-5 flex flex-col sm:flex-row sm:items-center justify-between gap-4 hover:bg-slate-50/50 transition-colors"
                    >
                      <div className="space-y-1">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="font-semibold text-slate-900 text-sm sm:text-base">
                            {d.name}
                          </span>
                          <span className="text-xs text-slate-500 font-mono">
                            {d.email_address}
                          </span>
                          {d.status === "pending" ? (
                            <Badge variant="warning" className="text-xs">
                              Pending Verification
                            </Badge>
                          ) : (
                            <Badge
                              variant={d.is_enabled ? "success" : "secondary"}
                              className="text-xs"
                            >
                              {d.is_enabled ? t.enabled : t.disabled}
                            </Badge>
                          )}
                        </div>

                        {d.status === "pending" && (
                          <div className="mt-2 p-2.5 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                            <span className="flex items-center gap-1.5">
                              <AlertTriangle className="w-4 h-4 text-amber-600 flex-shrink-0" />
                              Ownership verification required. Check inbox or enter token.
                            </span>
                            {canManage && (
                              <div className="flex items-center gap-1.5 flex-shrink-0">
                                <button
                                  onClick={() => handleResendEmailVerification(d.id)}
                                  className="px-2.5 py-1 bg-purple-600 hover:bg-purple-700 text-white rounded text-xs font-semibold shadow-xs"
                                >
                                  Resend Email
                                </button>
                                <button
                                  onClick={() => {
                                    setManualTokenInput("");
                                    setVerifyTokenModalOpen(true);
                                  }}
                                  className="px-2 py-1 bg-slate-200 hover:bg-slate-300 text-slate-700 rounded text-xs"
                                >
                                  Enter Token
                                </button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>

                      <div className="flex items-center gap-2 flex-wrap">
                        {canManage && (
                          <>
                            <button
                              onClick={() => handleToggleEnabled(d)}
                              className={`px-3 py-1.5 rounded text-xs font-medium border transition-colors ${d.is_enabled
                                  ? "bg-slate-100 text-slate-700 hover:bg-slate-200 border-slate-300"
                                  : "bg-emerald-50 text-emerald-700 hover:bg-emerald-100 border-emerald-300"
                                }`}
                            >
                              {d.is_enabled ? t.disabled : t.enabled}
                            </button>

                            <button
                              onClick={() => handleOpenPreferences(d)}
                              className="px-3 py-1.5 rounded text-xs font-medium border border-slate-300 hover:bg-slate-100 flex items-center gap-1 text-slate-700"
                            >
                              <Sliders className="w-3.5 h-3.5" />
                              {t.eventPreferences}
                            </button>
                          </>
                        )}

                        <button
                          onClick={() => handleTestNotification(d.id)}
                          disabled={testingDestId === d.id}
                          className="px-3 py-1.5 rounded text-xs font-medium bg-purple-50 text-purple-700 hover:bg-purple-100 border border-purple-200 flex items-center gap-1 transition-colors"
                        >
                          <Send className="w-3.5 h-3.5" />
                          {testingDestId === d.id ? "Sending..." : t.testNotification}
                        </button>

                        {canManage && (
                          <button
                            onClick={() => handleDisconnect(d.id)}
                            className="p-1.5 text-rose-600 hover:bg-rose-50 rounded transition-colors"
                            title={t.disconnect}
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tab 2: Delivery History */}
        {activeTab === "history" && (
          <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="p-5 border-b border-slate-200 flex flex-col sm:flex-row sm:items-center justify-between gap-2 bg-slate-50">
              <div>
                <h2 className="text-base font-semibold text-slate-900 flex items-center gap-2">
                  <Clock className="w-5 h-5 text-slate-600" />
                  {t.deliveryHistory}
                </h2>
                <p className="text-xs text-slate-400 mt-0.5">
                  Tracks external provider acceptance (Telegram Bot API / SMTP Relay). External acceptance does not guarantee device receipt.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={fetchHistory}
                disabled={loadingHistory}
                className="flex items-center gap-1 text-xs"
              >
                <RefreshCw className={`w-3.5 h-3.5 ${loadingHistory ? "animate-spin" : ""}`} />
                Refresh
              </Button>
            </div>

            {loadingHistory ? (
              <div className="p-8 text-center text-slate-400 text-sm">
                <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2" />
                Loading history...
              </div>
            ) : history.length === 0 ? (
              <div className="p-8 text-center text-slate-400 text-sm">
                <Clock className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                {t.noHistoryFound}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-slate-600">
                  <thead className="bg-slate-50 text-slate-500 font-medium text-xs border-b border-slate-200">
                    <tr>
                      <th className="py-3 px-4">{t.attemptTime}</th>
                      <th className="py-3 px-4">Channel</th>
                      <th className="py-3 px-4">{t.recipient}</th>
                      <th className="py-3 px-4">{t.eventType}</th>
                      <th className="py-3 px-4">{t.status}</th>
                      <th className="py-3 px-4">{t.retryCount}</th>
                      <th className="py-3 px-4">{t.errorMessage}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {history.map((h) => (
                      <tr key={h.id} className="hover:bg-slate-50/50 transition-colors">
                        <td className="py-3 px-4 whitespace-nowrap text-xs text-slate-500">
                          {new Date(h.created_at).toLocaleString()}
                        </td>
                        <td className="py-3 px-4">
                          <span className="inline-flex items-center gap-1 text-xs font-medium text-slate-700">
                            {h.destination_type.startsWith("telegram") ? (
                              <Send className="w-3.5 h-3.5 text-blue-500" />
                            ) : (
                              <Mail className="w-3.5 h-3.5 text-purple-500" />
                            )}
                            {h.destination_type}
                          </span>
                        </td>
                        <td className="py-3 px-4 font-mono text-xs text-slate-700">{h.recipient}</td>
                        <td className="py-3 px-4">
                          <code className="text-xs bg-slate-100 text-slate-800 px-1.5 py-0.5 rounded">
                            {h.event_type}
                          </code>
                        </td>
                        <td className="py-3 px-4">
                          <Badge
                            variant={
                              h.status === "sent"
                                ? "success"
                                : h.status === "retrying" || h.status === "uncertain"
                                  ? "warning"
                                  : h.status === "failed"
                                    ? "destructive"
                                    : "secondary"
                            }
                            className="text-xs uppercase font-mono"
                          >
                            {h.status === "sent"
                              ? "ACCEPTED BY PROVIDER"
                              : h.status === "uncertain"
                                ? "UNCERTAIN (CRASH)"
                                : h.status}
                          </Badge>
                        </td>
                        <td className="py-3 px-4 text-xs font-mono">{h.retry_count}</td>
                        <td className="py-3 px-4 text-xs text-rose-600 max-w-xs truncate font-mono">
                          {h.error_message || "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Modal: Connect Private Telegram */}
        {connectModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-5 animate-in fade-in-50 zoom-in-95">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="font-semibold text-slate-900 text-base flex items-center gap-2">
                  <Send className="w-5 h-5 text-blue-600" />
                  {t.telegramConnectTitle}
                </h3>
                <button
                  onClick={() => setConnectModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 text-sm font-bold"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-4 text-sm text-slate-600">
                <p>{t.telegramConnectInstructions}</p>

                {deepLink ? (
                  <div className="space-y-3 pt-2">
                    <a
                      href={deepLink}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="w-full py-3 px-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-medium flex items-center justify-center gap-2 shadow-sm transition-colors text-center"
                    >
                      <Send className="w-4 h-4" />
                      {t.openTelegram}
                      <ExternalLink className="w-4 h-4" />
                    </a>

                    <div className="p-3 bg-amber-50 rounded-lg border border-amber-200 text-xs text-amber-800 flex items-start gap-2">
                      <Clock className="w-4 h-4 text-amber-600 flex-shrink-0 mt-0.5" />
                      <span>Single-use link valid for 15 minutes. Automatically bound upon start.</span>
                    </div>
                  </div>
                ) : (
                  <div className="py-6 text-center text-slate-400">
                    <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2" />
                    Generating single-use link...
                  </div>
                )}
              </div>

              <div className="flex justify-end pt-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setConnectModalOpen(false);
                    fetchDestinations();
                  }}
                >
                  Done
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Connect Group Telegram */}
        {groupModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-5 animate-in fade-in-50 zoom-in-95">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="font-semibold text-slate-900 text-base flex items-center gap-2">
                  <Users className="w-5 h-5 text-blue-600" />
                  {t.telegramGroupTitle}
                </h3>
                <button
                  onClick={() => setGroupModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 text-sm font-bold"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-4 text-sm text-slate-600">
                <div className="p-3 bg-slate-50 rounded-lg border border-slate-200 space-y-1">
                  <p className="font-medium text-slate-800">{t.telegramGroupStep1}</p>
                  <code className="text-blue-600 font-mono font-semibold">
                    @{botUsername || "ClientPulseAlertsBot"}
                  </code>
                </div>

                <div className="p-3 bg-slate-50 rounded-lg border border-slate-200 space-y-2">
                  <p className="font-medium text-slate-800">{t.telegramGroupStep2}</p>
                  <div className="flex items-center gap-2">
                    <code className="flex-1 bg-white p-2 rounded border border-slate-300 font-mono text-xs text-slate-900 font-bold overflow-x-auto">
                      /connect@{botUsername} {groupCode}
                    </code>
                    <button
                      onClick={() => {
                        navigator.clipboard.writeText(`/connect@${botUsername} ${groupCode}`);
                        setCodeCopied(true);
                        setTimeout(() => setCodeCopied(false), 2000);
                      }}
                      className="p-2 border border-slate-300 rounded hover:bg-slate-100 flex items-center gap-1 text-xs"
                      title={t.copyCode}
                    >
                      {codeCopied ? (
                        <Check className="w-4 h-4 text-emerald-600" />
                      ) : (
                        <Copy className="w-4 h-4" />
                      )}
                    </button>
                  </div>
                </div>

                <div className="p-3 bg-amber-50 rounded-lg border border-amber-200 text-xs text-amber-800 flex items-start gap-2">
                  <Clock className="w-4 h-4 text-amber-600 flex-shrink-0 mt-0.5" />
                  <span>Code is single-use and expires in 15 minutes.</span>
                </div>
              </div>

              <div className="flex justify-end pt-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setGroupModalOpen(false);
                    fetchDestinations();
                  }}
                >
                  Done
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Add Email */}
        {emailModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-5 animate-in fade-in-50 zoom-in-95">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="font-semibold text-slate-900 text-base flex items-center gap-2">
                  <Mail className="w-5 h-5 text-purple-600" />
                  {t.addEmailTitle}
                </h3>
                <button
                  onClick={() => setEmailModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 text-sm font-bold"
                >
                  ✕
                </button>
              </div>

              <form onSubmit={handleCreateEmail} className="space-y-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    {t.recipient} (Email)
                  </label>
                  <Input
                    type="email"
                    required
                    placeholder={t.emailPlaceholder}
                    value={emailInput}
                    onChange={(e) => setEmailInput(e.target.value)}
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    {t.destinationName}
                  </label>
                  <Input
                    type="text"
                    placeholder={t.labelPlaceholder}
                    value={emailNameInput}
                    onChange={(e) => setEmailNameInput(e.target.value)}
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setEmailModalOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={savingEmail}>
                    {savingEmail ? "Saving..." : t.addEmailDestination}
                  </Button>
                </div>
              </form>
            </div>
          </div>
        )}

        {/* Modal: Event Preferences */}
        {prefModalOpen && editingDest && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-5 animate-in fade-in-50 zoom-in-95">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="font-semibold text-slate-900 text-base flex items-center gap-2">
                  <Sliders className="w-5 h-5 text-blue-600" />
                  {t.eventPreferences}: {editingDest.name}
                </h3>
                <button
                  onClick={() => setPrefModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 text-sm font-bold"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-3">
                <p className="text-xs text-slate-500 mb-2">{t.eventTypes}</p>
                {eventCategoryList.map((cat) => {
                  const isChecked = editPrefs[cat.key] !== false;
                  return (
                    <label
                      key={cat.key}
                      className="flex items-center justify-between p-2.5 rounded-lg border border-slate-200 hover:bg-slate-50 cursor-pointer transition-colors"
                    >
                      <span className="text-sm font-medium text-slate-800">{cat.label}</span>
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={(e) =>
                          setEditPrefs((prev) => ({
                            ...prev,
                            [cat.key]: e.target.checked,
                          }))
                        }
                        className="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
                      />
                    </label>
                  );
                })}
              </div>

              <div className="flex justify-end gap-2 pt-3 border-t">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setPrefModalOpen(false)}
                >
                  Cancel
                </Button>
                <Button onClick={handleSavePreferences} disabled={savingPrefs}>
                  {savingPrefs ? "Saving..." : t.savePreferences}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Modal: Enter Verification Token */}
        {verifyTokenModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-sm">
            <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6 space-y-5 animate-in fade-in-50 zoom-in-95">
              <div className="flex items-center justify-between border-b pb-3">
                <h3 className="font-semibold text-slate-900 text-base flex items-center gap-2">
                  <Mail className="w-5 h-5 text-purple-600" />
                  Verify Email Destination
                </h3>
                <button
                  onClick={() => setVerifyTokenModalOpen(false)}
                  className="text-slate-400 hover:text-slate-600 text-sm font-bold"
                >
                  ✕
                </button>
              </div>

              <form onSubmit={handleManualVerifyToken} className="space-y-4">
                <p className="text-xs text-slate-600">
                  Paste the verification token received in your confirmation email to activate this destination.
                </p>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Verification Token
                  </label>
                  <Input
                    type="text"
                    required
                    placeholder="e.g. 48-64 hex character token"
                    value={manualTokenInput}
                    onChange={(e) => setManualTokenInput(e.target.value)}
                  />
                </div>

                <div className="flex justify-end gap-2 pt-3 border-t">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setVerifyTokenModalOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" disabled={verifyingToken}>
                    {verifyingToken ? "Verifying..." : "Verify & Activate"}
                  </Button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
