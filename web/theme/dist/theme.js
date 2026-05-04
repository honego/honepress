(() => {
  const t = "blog-theme",
    e = ["auto", "light", "dark"],
    n = { auto: "主题：自动", light: "主题：亮色", dark: "主题：暗色" };
  function o() {
    (d(),
      r(i()),
      u(i()),
      document.querySelectorAll("[data-theme-toggle]").forEach((a) => {
        a.addEventListener("click", () => {
          const c = s(i());
          (l(c), r(c), u(c));
        });
      }));
  }
  function d() {
    window.lucide?.createIcons({ nameAttr: "data-lucide" });
  }
  function r(a) {
    document.documentElement.setAttribute("data-theme", a);
  }
  function u(a) {
    document.querySelectorAll("[data-theme-toggle]").forEach((c) => {
      (c.setAttribute("aria-label", n[a]), c.setAttribute("title", n[a]));
    });
  }
  function s(a) {
    const c = e.indexOf(a);
    return e[(c + 1) % e.length];
  }
  function i() {
    try {
      const a = window.localStorage.getItem(t);
      if (a === "auto" || a === "light" || a === "dark") return a;
    } catch {
      return m();
    }
    return m();
  }
  function l(a) {
    try {
      window.localStorage.setItem(t, a);
    } catch {
      return;
    }
  }
  function m() {
    const a = document.documentElement.dataset.theme;
    return a === "light" || a === "dark" || a === "auto" ? a : "auto";
  }
  (r(i()), document.readyState === "loading" ? document.addEventListener("DOMContentLoaded", o) : o());
})();
