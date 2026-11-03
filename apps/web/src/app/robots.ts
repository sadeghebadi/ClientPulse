import { MetadataRoute } from "next";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: ["/", "/login", "/signup", "/public/reports/"],
        disallow: [
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

