import type { Metadata } from "next";
import React from "react";

export const metadata: Metadata = {
  title: "Client Care Report · ClientPulse",
  robots: {
    index: false,
    follow: false,
    nocache: true,
    googleBot: {
      index: false,
      follow: false,
      noarchive: true,
    },
  },
  other: {
    robots: "noindex, nofollow, noarchive",
  },
};

export default function PublicReportsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <>{children}</>;
}
