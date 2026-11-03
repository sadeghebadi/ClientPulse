import type { Metadata } from "next";
import "./globals.css";
import { I18nProvider } from "@/lib/i18n";
import { AuthProvider } from "@/lib/auth-context";
import { ToastProvider } from "@/components/ui/Toast";
import { inter, vazirmatn } from "@/lib/fonts";

export const metadata: Metadata = {
  metadataBase: new URL("https://mindu.ir"),
  title: {
    default: "ClientPulse — Website Monitoring & Reliability Platform for Web Agencies | mindu.ir",
    template: "%s | ClientPulse",
  },
  description:
    "The dedicated website monitoring and client care platform for digital agencies. Monitor website availability, track cron heartbeats, manage incidents, and document client maintenance from one workspace.",
  keywords: [
    "website monitoring",
    "agency monitoring",
    "uptime monitoring",
    "cron heartbeat",
    "incident management",
    "client care reports",
    "SaaS for agencies",
    "mindu.ir",
  ],
  authors: [{ name: "ClientPulse" }],
  creator: "ClientPulse",
  publisher: "ClientPulse",
  alternates: {
    canonical: "https://mindu.ir",
  },
  openGraph: {
    title: "ClientPulse — Website Monitoring & Reliability Platform for Web Agencies",
    description:
      "Stay ahead of every client website. Monitor availability, catch issues early, and keep maintenance organized from one centralized workspace.",
    url: "https://mindu.ir",
    siteName: "ClientPulse",
    locale: "en_US",
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "ClientPulse — Website Monitoring & Reliability Platform for Web Agencies",
    description:
      "Stay ahead of every client website. Purpose-built for digital agencies managing client retainers.",
  },
  robots: {
    index: true,
    follow: true,
    googleBot: {
      index: true,
      follow: true,
      "max-video-preview": -1,
      "max-image-preview": "large",
      "max-snippet": -1,
    },
  },
};

const dirScript = `(function() {
  try {
    var saved = localStorage.getItem('clientpulse_locale');
    if (saved === 'fa') {
      document.documentElement.lang = 'fa';
      document.documentElement.dir = 'rtl';
    } else {
      document.documentElement.lang = 'en';
      document.documentElement.dir = 'ltr';
    }
  } catch(e) {}
})();`;

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html
      lang="en"
      dir="ltr"
      suppressHydrationWarning
      className={`${inter.variable} ${vazirmatn.variable}`}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: dirScript }} />
      </head>
      <body className="antialiased flex flex-col min-h-screen bg-slate-50 text-slate-900">
        <I18nProvider>
          <AuthProvider>
            <ToastProvider>
              {children}
            </ToastProvider>
          </AuthProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
