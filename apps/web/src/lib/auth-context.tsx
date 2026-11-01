"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { useRouter, usePathname } from "next/navigation";
import { apiFetch } from "./api";

export interface User {
  id: string;
  email: string;
  display_name: string;
  email_verified_at?: string;
  status: string;
}

export interface Organization {
  id: string;
  name: string;
  slug: string;
  locale: string;
  timezone: string;
  status: string;
}

export interface Membership {
  id: string;
  organization_id: string;
  user_id: string;
  role: string;
  org_name?: string;
  org_slug?: string;
}

interface AuthContextType {
  user: User | null;
  activeOrg: Organization | null;
  role: string | null;
  memberships: Membership[];
  loading: boolean;
  refreshUser: () => Promise<void>;
  logout: () => Promise<void>;
  switchOrganization: (orgID: string) => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<User | null>(null);
  const [activeOrg, setActiveOrg] = useState<Organization | null>(null);
  const [role, setRole] = useState<string | null>(null);
  const [memberships, setMemberships] = useState<Membership[]>([]);
  const [loading, setLoading] = useState(true);

  const refreshUser = useCallback(async () => {
    try {
      const data = await apiFetch<{
        user: User;
        active_organization?: Organization;
        role?: string;
        memberships: Membership[];
      }>("/api/v1/auth/me");

      setUser(data.user);
      setActiveOrg(data.active_organization || null);
      setRole(data.role || null);
      setMemberships(data.memberships || []);
    } catch {
      setUser(null);
      setActiveOrg(null);
      setRole(null);
      setMemberships([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    let isMounted = true;
    apiFetch<{
      user: User;
      active_organization?: Organization;
      role?: string;
      memberships: Membership[];
    }>("/api/v1/auth/me")
      .then((data) => {
        if (!isMounted) return;
        setUser(data.user);
        setActiveOrg(data.active_organization || null);
        setRole(data.role || null);
        setMemberships(data.memberships || []);
      })
      .catch(() => {
        if (!isMounted) return;
        setUser(null);
        setActiveOrg(null);
        setRole(null);
        setMemberships([]);
      })
      .finally(() => {
        if (isMounted) setLoading(false);
      });

    // Listen for session expired events
    const handleSessionExpired = () => {
      setUser(null);
      setActiveOrg(null);
      setRole(null);
      setMemberships([]);
      router.push("/login?session_expired=true");
    };

    window.addEventListener("clientpulse:session_expired", handleSessionExpired);

    return () => {
      isMounted = false;
      window.removeEventListener("clientpulse:session_expired", handleSessionExpired);
    };
  }, [router]);

  const logout = async () => {
    try {
      await apiFetch("/api/v1/auth/logout", { method: "POST" });
    } finally {
      setUser(null);
      setActiveOrg(null);
      setRole(null);
      setMemberships([]);
      if (typeof window !== "undefined") {
        window.location.replace(window.location.origin + "/login");
      }
    }
  };

  const switchOrganization = async (orgID: string) => {
    await apiFetch("/api/v1/organizations/select", {
      method: "POST",
      body: JSON.stringify({ organization_id: orgID }),
    });
    await refreshUser();
    // If user is currently on an ID-specific detail route, redirect to root or list to avoid cross-tenant 404s
    if (pathname && /\/(clients|assets|monitors)\/[a-zA-Z0-9_-]+/.test(pathname)) {
      router.push("/");
    }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        activeOrg,
        role,
        memberships,
        loading,
        refreshUser,
        logout,
        switchOrganization,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
