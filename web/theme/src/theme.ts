type ThemeMode = "auto" | "light" | "dark";

type LucideWindow = Window & {
  lucide?: {
    createIcons: (options?: { nameAttr?: string }) => void;
  };
};

const storageKey = "blog-theme";
const themeModes: ThemeMode[] = ["auto", "light", "dark"];
const themeLabels: Record<ThemeMode, string> = {
  auto: "主题：自动",
  light: "主题：亮色",
  dark: "主题：暗色",
};

applyTheme(readStoredTheme());

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializePage);
} else {
  initializePage();
}

function initializePage(): void {
  initializeIcons();
  initializeGiscusComments();
  updateToggleButtons(readStoredTheme());

  const toggleButtons = document.querySelectorAll<HTMLButtonElement>("[data-theme-toggle]");
  toggleButtons.forEach((toggleButton) => {
    toggleButton.addEventListener("click", () => {
      const nextTheme = nextThemeMode(readStoredTheme());
      saveTheme(nextTheme);
      applyTheme(nextTheme);
      updateToggleButtons(nextTheme);
      syncGiscusTheme(nextTheme);
    });
  });
}

function initializeIcons(): void {
  const lucide = (window as LucideWindow).lucide;
  lucide?.createIcons({ nameAttr: "data-lucide" });
}

function initializeGiscusComments(): void {
  const commentContainers = document.querySelectorAll<HTMLElement>("[data-giscus-comments]");
  commentContainers.forEach((commentContainer) => {
    if (!hasRequiredGiscusConfig(commentContainer) || hasMountedGiscus(commentContainer)) {
      return;
    }

    const giscusScript = document.createElement("script");
    giscusScript.src = "https://giscus.app/client.js";
    giscusScript.async = true;
    giscusScript.crossOrigin = "anonymous";
    giscusScript.setAttribute("data-giscus-script", "true");
    setGiscusAttributes(giscusScript, commentContainer);
    commentContainer.appendChild(giscusScript);
  });
}

function hasRequiredGiscusConfig(commentContainer: HTMLElement): boolean {
  return ["repo", "repoId", "category", "categoryId"].every((datasetKey) => {
    const datasetValue = commentContainer.dataset[datasetKey];
    return typeof datasetValue === "string" && datasetValue.trim() !== "";
  });
}

function hasMountedGiscus(commentContainer: HTMLElement): boolean {
  return Boolean(
    commentContainer.querySelector("iframe.giscus-frame") ||
    commentContainer.querySelector("script[data-giscus-script]"),
  );
}

function setGiscusAttributes(giscusScript: HTMLScriptElement, commentContainer: HTMLElement): void {
  const giscusAttributes: Array<[string, string]> = [
    ["data-repo", commentContainer.dataset.repo ?? ""],
    ["data-repo-id", commentContainer.dataset.repoId ?? ""],
    ["data-category", commentContainer.dataset.category ?? ""],
    ["data-category-id", commentContainer.dataset.categoryId ?? ""],
    ["data-mapping", commentContainer.dataset.mapping ?? "pathname"],
    ["data-strict", commentContainer.dataset.strict ?? "0"],
    ["data-reactions-enabled", commentContainer.dataset.reactionsEnabled ?? "1"],
    ["data-emit-metadata", commentContainer.dataset.emitMetadata ?? "0"],
    ["data-input-position", commentContainer.dataset.inputPosition ?? "bottom"],
    ["data-theme", giscusThemeFor(readStoredTheme(), commentContainer.dataset.theme)],
    ["data-lang", commentContainer.dataset.lang ?? document.documentElement.lang ?? "zh-CN"],
  ];

  giscusAttributes.forEach(([attributeName, attributeValue]) => {
    giscusScript.setAttribute(attributeName, attributeValue);
  });
}

function syncGiscusTheme(themeMode: ThemeMode): void {
  const commentContainers = document.querySelectorAll<HTMLElement>("[data-giscus-comments]");
  commentContainers.forEach((commentContainer) => {
    const giscusFrame = commentContainer.querySelector<HTMLIFrameElement>("iframe.giscus-frame");
    if (!giscusFrame?.contentWindow) {
      return;
    }

    giscusFrame.contentWindow.postMessage(
      {
        giscus: {
          setConfig: {
            theme: giscusThemeFor(themeMode, commentContainer.dataset.theme),
          },
        },
      },
      "https://giscus.app",
    );
  });
}

function giscusThemeFor(themeMode: ThemeMode, configuredTheme?: string): string {
  const normalizedTheme = configuredTheme?.trim();
  if (normalizedTheme && normalizedTheme !== "preferred_color_scheme") {
    return normalizedTheme;
  }
  if (themeMode === "light" || themeMode === "dark") {
    return themeMode;
  }
  return "preferred_color_scheme";
}

function applyTheme(themeMode: ThemeMode): void {
  document.documentElement.setAttribute("data-theme", themeMode);
}

function updateToggleButtons(themeMode: ThemeMode): void {
  const toggleButtons = document.querySelectorAll<HTMLButtonElement>("[data-theme-toggle]");
  toggleButtons.forEach((toggleButton) => {
    toggleButton.setAttribute("aria-label", themeLabels[themeMode]);
    toggleButton.setAttribute("title", themeLabels[themeMode]);
  });
}

function nextThemeMode(themeMode: ThemeMode): ThemeMode {
  const currentThemeIndex = themeModes.indexOf(themeMode);
  const nextThemeIndex = (currentThemeIndex + 1) % themeModes.length;
  return themeModes[nextThemeIndex];
}

function readStoredTheme(): ThemeMode {
  try {
    const storedTheme = window.localStorage.getItem(storageKey);
    if (storedTheme === "auto" || storedTheme === "light" || storedTheme === "dark") {
      return storedTheme;
    }
  } catch {
    return readDocumentDefaultTheme();
  }
  return readDocumentDefaultTheme();
}

function saveTheme(themeMode: ThemeMode): void {
  try {
    window.localStorage.setItem(storageKey, themeMode);
  } catch {
    return;
  }
}

function readDocumentDefaultTheme(): ThemeMode {
  const defaultTheme = document.documentElement.dataset.theme;
  if (defaultTheme === "light" || defaultTheme === "dark" || defaultTheme === "auto") {
    return defaultTheme;
  }
  return "auto";
}
