import Head from "next/head";
import Link from "next/link";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import type { PublicSiteSettings } from "../lib/api";
import { defaultFaviconHref } from "../lib/favicon";

type ThemeMode = "auto" | "light" | "dark";
type IconName = "house" | "log-in" | "map" | "moon" | "rss" | "sun";

const storageKey = "honepress-theme";
const themeModes: ThemeMode[] = ["auto", "light", "dark"];

const themeLabels: Record<ThemeMode, string> = {
  auto: "主题：自动",
  light: "主题：明亮",
  dark: "主题：暗黑",
};

export function SiteHead({
  site,
  title,
  description,
  canonicalPath,
  faviconHref,
  type = "website",
}: {
  site: PublicSiteSettings | null;
  title?: string;
  description?: string;
  canonicalPath?: string;
  faviconHref?: string | null;
  type?: "website" | "article";
}) {
  const siteTitle = siteName(site);
  const siteDescription = site?.description?.trim() ?? "";
  const pageTitle = title ?? (siteDescription ? `${siteTitle} - ${siteDescription}` : siteTitle);
  const pageDescription = description ?? siteDescription;
  const resolvedFaviconHref =
    faviconHref === null ? "" : faviconHref?.trim() || site?.iconUrl?.trim() || defaultFaviconHref;

  return (
    <Head>
      <title>{pageTitle}</title>
      {pageDescription ? <meta name="description" content={pageDescription} /> : null}
      {canonicalPath ? <link rel="canonical" href={canonicalPath} /> : null}
      <meta property="og:type" content={type} />
      <meta property="og:title" content={pageTitle} />
      {pageDescription ? <meta property="og:description" content={pageDescription} /> : null}
      {canonicalPath ? <meta property="og:url" content={canonicalPath} /> : null}
      <meta property="og:site_name" content={siteTitle} />
      {site?.iconUrl ? <meta property="og:image" content={site.iconUrl} /> : null}
      <meta name="twitter:card" content={site?.iconUrl ? "summary_large_image" : "summary"} />
      <meta name="twitter:title" content={pageTitle} />
      {pageDescription ? <meta name="twitter:description" content={pageDescription} /> : null}
      {site?.iconUrl ? <meta name="twitter:image" content={site.iconUrl} /> : null}
      {resolvedFaviconHref ? <link rel="icon" href={resolvedFaviconHref} /> : null}
      <link rel="alternate" type="application/rss+xml" title={siteTitle} href="/rss.xml" />
      {site?.font === "douyin-sans" ? (
        <link rel="preload" href="/fonts/DouyinSansBold.ttf" as="font" type="font/ttf" crossOrigin="anonymous" />
      ) : null}
    </Head>
  );
}

export function SiteLayout({
  site,
  children,
  pageClassName,
}: {
  site: PublicSiteSettings | null;
  children: ReactNode;
  pageClassName?: string;
}) {
  const { theme, toggleTheme } = useTheme(site);
  const siteTitle = siteName(site);
  const siteIcon = site?.iconUrl?.trim() || defaultFaviconHref;
  const themeIcon = useThemeIcon(theme);

  return (
    <>
      <header className="site-header">
        <nav className="nav">
          <Link className="brand" href="/">
            <img className="brand-icon" src={siteIcon} alt="" />
            <span>{siteTitle}</span>
          </Link>
          <div className="nav-links">
            <Link className="icon-link" href="/" aria-label="首页" title="首页">
              <LucideIcon name="house" />
            </Link>
            <Link href="/archive.html">归档</Link>
            <button
              type="button"
              className="theme-toggle icon-button"
              aria-label={themeLabels[theme]}
              title={themeLabels[theme]}
              onClick={toggleTheme}
            >
              <LucideIcon name={themeIcon} />
            </button>
            <a className="icon-link" href="/admin/" aria-label="后台登录" title="后台登录">
              <LucideIcon name="log-in" />
            </a>
          </div>
        </nav>
      </header>

      <main className={`page${pageClassName ? ` ${pageClassName}` : ""}`}>{children}</main>

      <footer className="site-footer">
        <div className="footer-inner">
          <p>
            Powered by{" "}
            <a href="https://github.com/honeok/honepress" target="_blank" rel="noopener noreferrer">
              HonePress
            </a>
          </p>
          <div className="footer-links">
            <a className="footer-sitemap icon-link" href="/sitemap.xml" aria-label="站点地图" title="站点地图">
              <LucideIcon name="map" />
            </a>
            <a className="footer-rss icon-link" href="/rss.xml" aria-label="RSS" title="RSS">
              <LucideIcon name="rss" />
            </a>
          </div>
        </div>
      </footer>
    </>
  );
}

export function siteName(site: PublicSiteSettings | null): string {
  const title = site?.title?.trim();
  return title || "HonePress";
}

function useTheme(site: PublicSiteSettings | null) {
  const [theme, setTheme] = useState<ThemeMode>("auto");

  useEffect(() => {
    const initialTheme = readStoredTheme(site?.themeDefault);
    setTheme(initialTheme);
    applyTheme(initialTheme, site?.font);
  }, [site?.themeDefault, site?.font]);

  function toggleTheme() {
    const nextTheme = nextThemeMode(theme);
    saveTheme(nextTheme);
    setTheme(nextTheme);
    applyTheme(nextTheme, site?.font);
    syncGiscusTheme(nextTheme);
  }

  return { theme, toggleTheme };
}

function useThemeIcon(theme: ThemeMode): IconName {
  const [systemDark, setSystemDark] = useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    setSystemDark(mediaQuery.matches);
    const updateSystemTheme = () => setSystemDark(mediaQuery.matches);
    mediaQuery.addEventListener("change", updateSystemTheme);
    return () => mediaQuery.removeEventListener("change", updateSystemTheme);
  }, []);

  return theme === "dark" || (theme === "auto" && systemDark) ? "moon" : "sun";
}

function applyTheme(theme: ThemeMode, font?: string) {
  document.documentElement.dataset.theme = theme;
  document.documentElement.dataset.font = font || "default";
}

function readStoredTheme(defaultTheme?: string): ThemeMode {
  try {
    const storedTheme = window.localStorage.getItem(storageKey);
    if (isThemeMode(storedTheme)) return storedTheme;
  } catch {
    return normalizeTheme(defaultTheme);
  }
  return normalizeTheme(defaultTheme);
}

function saveTheme(theme: ThemeMode) {
  try {
    window.localStorage.setItem(storageKey, theme);
  } catch {
    return;
  }
}

function normalizeTheme(theme?: string): ThemeMode {
  return isThemeMode(theme) ? theme : "auto";
}

function isThemeMode(theme: unknown): theme is ThemeMode {
  return theme === "auto" || theme === "light" || theme === "dark";
}

function nextThemeMode(theme: ThemeMode): ThemeMode {
  const currentIndex = themeModes.indexOf(theme);
  return themeModes[(currentIndex + 1) % themeModes.length];
}

function syncGiscusTheme(theme: ThemeMode) {
  const giscusTheme = theme === "light" || theme === "dark" ? theme : "preferred_color_scheme";
  document.querySelectorAll<HTMLIFrameElement>("iframe.giscus-frame").forEach((frame) => {
    frame.contentWindow?.postMessage(
      {
        giscus: {
          setConfig: { theme: giscusTheme },
        },
      },
      "https://giscus.app",
    );
  });
}

function LucideIcon({ name }: { name: IconName }) {
  const paths = iconPaths[name];
  return (
    <svg className="nav-icon" viewBox="0 0 24 24" aria-hidden="true">
      {paths.map((pathData, index) => (
        <path d={pathData} key={index} />
      ))}
    </svg>
  );
}

const iconPaths: Record<IconName, string[]> = {
  house: ["M3 11l9-8 9 8", "M5 10v10h5v-6h4v6h5V10"],
  "log-in": ["M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4", "M10 17l5-5-5-5", "M15 12H3"],
  map: ["M14 6l-4-2-6 3v13l6-3 4 2 6-3V3z", "M10 4v13", "M14 6v13"],
  moon: ["M21 12.8A9 9 0 1 1 11.2 3 7 7 0 0 0 21 12.8z"],
  rss: ["M4 11a9 9 0 0 1 9 9", "M4 4a16 16 0 0 1 16 16", "M5 19h.01"],
  sun: [
    "M12 4V2",
    "M12 22v-2",
    "M4.93 4.93 3.52 3.52",
    "M20.48 20.48l-1.41-1.41",
    "M2 12h2",
    "M20 12h2",
    "M4.93 19.07l-1.41 1.41",
    "M20.48 3.52l-1.41 1.41",
    "M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8",
  ],
};
