export const adminFaviconChangeEventName = "honepress-admin-favicon-change";
export const defaultFaviconHref = "/honepress-black.svg";

interface PublicSiteResponse {
  site?: {
    iconUrl?: string;
  };
}

export async function fetchSiteFaviconHref(): Promise<string> {
  const response = await fetch("/api/site", { credentials: "same-origin" });
  if (!response.ok) {
    return defaultFaviconHref;
  }
  const responseBody = (await response.json()) as PublicSiteResponse;
  return faviconHrefFromIconURL(responseBody.site?.iconUrl);
}

export function faviconHrefFromChangeEvent(event: Event): string {
  const iconUrl = (event as CustomEvent<string>).detail;
  return faviconHrefFromIconURL(typeof iconUrl === "string" ? iconUrl : "");
}

export function faviconHrefFromIconURL(iconUrl?: string | null): string {
  return iconUrl?.trim() || defaultFaviconHref;
}

export function notifyAdminFaviconChange(iconUrl: string) {
  window.dispatchEvent(new CustomEvent(adminFaviconChangeEventName, { detail: iconUrl }));
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
