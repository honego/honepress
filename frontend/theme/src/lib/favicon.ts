export const defaultFaviconHref = "/honepress-black.svg";

export function faviconHrefFromIcon(icon?: string | null, fallbackIconUrl?: string | null): string {
  const trimmedIcon = icon?.trim() ?? "";
  if (isIconURL(trimmedIcon)) return trimmedIcon;
  if (trimmedIcon) return emojiFaviconHref(trimmedIcon);
  return fallbackIconUrl?.trim() || defaultFaviconHref;
}

export function readCurrentFavicon(): string | null {
  if (typeof document === "undefined") return null;
  return document.querySelector<HTMLLinkElement>('link[rel="icon"]')?.getAttribute("href") || null;
}

export function syncDocumentFavicon(href: string) {
  if (typeof document === "undefined" || !href) return;
  let favicon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
  if (!favicon) {
    favicon = document.createElement("link");
    favicon.rel = "icon";
    document.head.appendChild(favicon);
  }
  if (favicon.getAttribute("href") !== href) {
    favicon.setAttribute("href", href);
  }
}

function isIconURL(icon: string): boolean {
  return icon.startsWith("http://") || icon.startsWith("https://") || icon.startsWith("/");
}

function emojiFaviconHref(emoji: string): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text x="50%" y="50%" style="dominant-baseline:central;text-anchor:middle;font-size:86px;">${escapeHTML(
    emoji,
  )}</text></svg>`;
  return `data:image/svg+xml,${encodeURIComponent(svg)}`;
}

function escapeHTML(value: string): string {
  return value.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
