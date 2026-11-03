import { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: ["/", "/login", "/signup"],
        disallow: [
          "/public/reports",
          "/public/reports/",
          "/dashboard",
          "/dashboard/",
          "/clients",
          "/clients/",
          "/monitors",
          "/monitors/",
          "/heartbeats",
          "/heartbeats/",
          "/incidents",
          "/incidents/",
          "/maintenance",
          "/maintenance/",
          "/reports",
          "/reports/",
          "/assets",
          "/assets/",
          "/settings",
          "/settings/",
          "/organizations",
          "/organizations/",
          "/onboarding",
          "/onboarding/",
        ],
      },
    ],
    sitemap: "https://mindu.ir/sitemap.xml",
  };
}

