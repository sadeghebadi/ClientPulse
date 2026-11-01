// ClientPulse API Client with CSRF token support and cookie-based authentication

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ||
  process.env.NEXT_PUBLIC_API_URL ||
  "http://localhost:8080";

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp("(^|;\\s*)(" + name + ")=([^;]*)"));
  return match ? decodeURIComponent(match[3]) : null;
}

export interface ApiError {
  code: string;
  message: string;
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = path.startsWith("http") ? path : `${API_BASE_URL}${path}`;

  const headers = new Headers(options.headers || {});
  headers.set("Content-Type", "application/json");

  // Mutating methods (POST, PUT, PATCH, DELETE) need CSRF token header
  const method = (options.method || "GET").toUpperCase();
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
    const csrfToken = getCookie("clientpulse_csrf");
    if (csrfToken) {
      headers.set("X-CSRF-Token", csrfToken);
    }
  }

  const response = await fetch(url, {
    ...options,
    headers,
    credentials: "include", // Ensure session cookies are sent
  });

  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    const errorMsg = data?.error?.message || response.statusText || "An unexpected error occurred.";
    const errorCode = data?.error?.code || `HTTP_${response.status}`;
    const err = new Error(errorMsg) as Error & { code?: string; status?: number };
    err.code = errorCode;
    err.status = response.status;

    // Handle session expiration (401) cleanly on protected routes
    if (response.status === 401 && typeof window !== "undefined") {
      const publicPaths = ["/login", "/signup", "/forgot-password", "/reset-password", "/verify-email"];
      if (!publicPaths.some((p) => window.location.pathname.startsWith(p))) {
        // Dispatch custom event for auth context to react
        window.dispatchEvent(new CustomEvent("clientpulse:session_expired"));
      }
    }

    throw err;
  }

  return data as T;
}

/**
 * Humanizes raw API errors (such as SSRF or authorization blocks)
 * into friendly, understandable explanations without weakening security.
 */
export function humanizeApiError(err: unknown, isRTL: boolean = false): string {
  if (!err) return isRTL ? "خطایی رخ داد" : "An unexpected error occurred";

  const message = err instanceof Error ? err.message : String(err);
  const code = (err as { code?: string })?.code || "";

  // SSRF Protection Humanized Messages
  if (
    code.includes("SSRF") ||
    message.toLowerCase().includes("restricted internal or private ip") ||
    message.toLowerCase().includes("loopback") ||
    message.toLowerCase().includes("private ip") ||
    message.toLowerCase().includes("ssrf") ||
    message.toLowerCase().includes("blocked by security policy")
  ) {
    return isRTL
      ? "به دلایل امنیتی، پایش آدرس‌های شبکه داخلی، لوکال‌هاست (localhost) و سرویس‌های محلی ابر مجاز نیست. لطفاً یک دامنه عمومی و معتبر (مانند https://example.com) وارد نمایید."
      : "For security and infrastructure protection, monitoring internal networks, private IP addresses, localhost, or cloud metadata endpoints is not permitted. Please enter a publicly accessible web address (e.g. https://example.com).";
  }

  // Duplicate URL Warning / Error
  if (code === "DUPLICATE_URL" || message.toLowerCase().includes("duplicate url")) {
    return isRTL
      ? "توجه: این آدرس پیش‌تر برای وب‌سایت دیگری در سازمان شما ثبت شده است."
      : "Note: Another asset in this workspace is already configured with this target URL.";
  }

  // Tenant / Resource Not Found
  if (code === "NOT_FOUND" || code === "CLIENT_NOT_FOUND" || code === "ASSET_NOT_FOUND" || code === "MONITOR_NOT_FOUND") {
    return isRTL
      ? "منبع درخواستی یافت نشد یا به فضای کاری فعال شما تعلق ندارد."
      : "The requested resource was not found or does not belong to your active workspace.";
  }

  // Rate Limiting
  if (code === "RATE_LIMITED" || (err as { status?: number })?.status === 429) {
    return isRTL
      ? "تعداد درخواست‌ها بیش از حد مجاز است. لطفاً چند لحظه صبر کنید و دوباره تلاش کنید."
      : "Too many requests. Please wait a moment before trying again.";
  }

  // Email Unverified
  if (code === "EMAIL_NOT_VERIFIED" || message.toLowerCase().includes("verify your email")) {
    return isRTL
      ? "جهت انجام این عملیات ابتدا باید ایمیل خود را تأیید نمایید."
      : "Please verify your email address before performing this action.";
  }

  return message;
}
