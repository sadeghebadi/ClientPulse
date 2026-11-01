// ClientPulse API Client with CSRF token support and cookie-based authentication

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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
    throw err;
  }

  return data as T;
}

