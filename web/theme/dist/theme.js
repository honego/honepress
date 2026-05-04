(() => {
  const storageKey = "blog-theme";
  const themeModes = ["auto", "light", "dark"];
  const themeLabels = {
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

  function initializePage() {
    initializeIcons();
    initializeGiscusComments();
    updateToggleButtons(readStoredTheme());

    document.querySelectorAll("[data-theme-toggle]").forEach((toggleButton) => {
      toggleButton.addEventListener("click", () => {
        const nextTheme = nextThemeMode(readStoredTheme());
        saveTheme(nextTheme);
        applyTheme(nextTheme);
        updateToggleButtons(nextTheme);
        syncGiscusTheme(nextTheme);
      });
    });
  }

  function initializeIcons() {
    window.lucide?.createIcons({ nameAttr: "data-lucide" });
  }

  function initializeGiscusComments() {
    document.querySelectorAll("[data-giscus-comments]").forEach((commentContainer) => {
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

  function hasRequiredGiscusConfig(commentContainer) {
    return ["repo", "repoId", "category", "categoryId"].every((datasetKey) => {
      const datasetValue = commentContainer.dataset[datasetKey];
      return typeof datasetValue === "string" && datasetValue.trim() !== "";
    });
  }

  function hasMountedGiscus(commentContainer) {
    return Boolean(
      commentContainer.querySelector("iframe.giscus-frame") ||
      commentContainer.querySelector("script[data-giscus-script]"),
    );
  }

  function setGiscusAttributes(giscusScript, commentContainer) {
    const giscusAttributes = [
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

  function syncGiscusTheme(themeMode) {
    document.querySelectorAll("[data-giscus-comments]").forEach((commentContainer) => {
      const giscusFrame = commentContainer.querySelector("iframe.giscus-frame");
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

  function giscusThemeFor(themeMode, configuredTheme) {
    const normalizedTheme = configuredTheme?.trim();
    if (normalizedTheme && normalizedTheme !== "preferred_color_scheme") {
      return normalizedTheme;
    }
    if (themeMode === "light" || themeMode === "dark") {
      return themeMode;
    }
    return "preferred_color_scheme";
  }

  function applyTheme(themeMode) {
    document.documentElement.setAttribute("data-theme", themeMode);
  }

  function updateToggleButtons(themeMode) {
    document.querySelectorAll("[data-theme-toggle]").forEach((toggleButton) => {
      toggleButton.setAttribute("aria-label", themeLabels[themeMode]);
      toggleButton.setAttribute("title", themeLabels[themeMode]);
    });
  }

  function nextThemeMode(themeMode) {
    const currentThemeIndex = themeModes.indexOf(themeMode);
    const nextThemeIndex = (currentThemeIndex + 1) % themeModes.length;
    return themeModes[nextThemeIndex];
  }

  function readStoredTheme() {
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

  function saveTheme(themeMode) {
    try {
      window.localStorage.setItem(storageKey, themeMode);
    } catch {
      return;
    }
  }

  function readDocumentDefaultTheme() {
    const defaultTheme = document.documentElement.dataset.theme;
    if (defaultTheme === "light" || defaultTheme === "dark" || defaultTheme === "auto") {
      return defaultTheme;
    }
    return "auto";
  }
})();
