type ThemeMode = "auto" | "light" | "dark";

const storageKey = "blog-theme";
const themeModes: ThemeMode[] = ["auto", "light", "dark"];
const themeLabels: Record<ThemeMode, string> = {
  auto: "主题：自动",
  light: "主题：亮色",
  dark: "主题：暗色",
};

applyTheme(readStoredTheme());

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initializeThemeToggle);
} else {
  initializeThemeToggle();
}

function initializeThemeToggle(): void {
  updateToggleButtons(readStoredTheme());
  const toggleButtons = document.querySelectorAll<HTMLButtonElement>("[data-theme-toggle]");
  toggleButtons.forEach((toggleButton) => {
    toggleButton.addEventListener("click", () => {
      const nextTheme = nextThemeMode(readStoredTheme());
      saveTheme(nextTheme);
      applyTheme(nextTheme);
      updateToggleButtons(nextTheme);
    });
  });
}

// 只写 data-theme，让 CSS 同时处理 auto、light、dark 三种状态。
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
