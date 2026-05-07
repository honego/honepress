<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";

import {
  UnauthorizedError,
  checkAdminSession,
  createPost,
  deletePost,
  fetchPost,
  fetchPosts,
  fetchSettings,
  loginAdmin,
  logoutAdmin,
  previewMarkdown,
  updatePost,
  updateSettings,
} from "./api/posts";
import type { PostDetail, PostSummary, SavePostRequest, SiteSettings } from "./types/posts";

type AdminView = "dashboard" | "posts" | "editor" | "settings";
type EditorMode = "create" | "edit";
type AdminThemeMode = "auto" | "light" | "dark";
type LucideWindow = Window & {
  lucide?: {
    createIcons: (options?: { nameAttr?: string }) => void;
  };
};

const MarkdownWorkspace = defineAsyncComponent(() => import("./components/MarkdownWorkspace.vue"));

const themeStorageKey = "honepress-theme";
const themeModes: AdminThemeMode[] = ["auto", "light", "dark"];
const themeLabels: Record<AdminThemeMode, string> = {
  auto: "跟随系统",
  light: "明亮主题",
  dark: "暗色主题",
};
const emojiOptions = [
  { value: "", label: "默认网站 icon" },
  { value: "☘️", label: "☘️ 日常" },
  { value: "🌱", label: "🌱 记录" },
  { value: "✨", label: "✨ 灵感" },
  { value: "📑", label: "📑 笔记" },
  { value: "💡", label: "💡 想法" },
  { value: "🚀", label: "🚀 项目" },
  { value: "📌", label: "📌 摘录" },
  { value: "🌙", label: "🌙 夜读" },
  { value: "🧠", label: "🧠 技术" },
];

const activeView = ref<AdminView>("dashboard");
const posts = ref<PostSummary[]>([]);
const editorMode = ref<EditorMode>("create");
const editorForm = ref<PostDetail>(createEmptyPost());
const aliasesText = ref("");
const tagDraft = ref("");
const previewHTML = ref("");
const statusMessage = ref("");
const errorMessage = ref("");
const isSaving = ref(false);
const isPreviewLoading = ref(false);
const isAuthenticated = ref(false);
const isLoggingIn = ref(false);
const isDeleteDialogOpen = ref(false);
const adminTheme = ref<AdminThemeMode>(readStoredTheme());
const siteSettings = ref<SiteSettings>(createEmptySiteSettings());
const loginForm = ref({ username: "admin", password: "" });
const loginError = ref("");

let previewTimerID: number | undefined;

const publishedPosts = computed(() => posts.value.filter((post) => !post.draft));
const draftPosts = computed(() => posts.value.filter((post) => post.draft));
const recentPosts = computed(() => posts.value.slice(0, 5));
const canEditExistingPost = computed(() => editorMode.value === "edit" && editorForm.value.id !== "");
const adminThemeLabel = computed(() => `主题：${themeLabels[adminTheme.value]}`);
const editorTitle = computed(() => editorForm.value.title.trim() || "未命名文章");
const pageTitle = computed(() => {
  if (activeView.value === "dashboard") return "首页";
  if (activeView.value === "posts") return "文章列表";
  if (activeView.value === "settings") return "站点配置";
  return editorMode.value === "create" ? "新建文章" : "编辑文章";
});
const pageDescription = computed(() => {
  if (activeView.value === "dashboard") return "轻量、安静、专注写作的 HonePress 后台。";
  if (activeView.value === "posts") return "管理已发布文章和草稿。";
  if (activeView.value === "settings") return "站点基础配置。";
  return "左侧编写 Markdown，右侧维护文章元信息。";
});

onMounted(() => {
  applyTheme(adminTheme.value);
  initializeIcons();
  window.addEventListener("load", initializeIcons);
  window.addEventListener("keydown", handleGlobalKeydown);
  void bootstrapAdmin();
});

onBeforeUnmount(() => {
  window.removeEventListener("load", initializeIcons);
  window.removeEventListener("keydown", handleGlobalKeydown);
  if (previewTimerID !== undefined) window.clearTimeout(previewTimerID);
});

watch(
  () => [activeView.value, isAuthenticated.value, isDeleteDialogOpen.value, statusMessage.value, errorMessage.value],
  () => void nextTick(initializeIcons),
);

watch(
  () => editorForm.value.body,
  () => {
    if (activeView.value === "editor") schedulePreview();
  },
);
async function bootstrapAdmin(): Promise<void> {
  errorMessage.value = "";
  try {
    await checkAdminSession();
    isAuthenticated.value = true;
    await loadInitialData();
  } catch (error) {
    if (error instanceof UnauthorizedError) {
      isAuthenticated.value = false;
      loginError.value = "";
    } else {
      isAuthenticated.value = true;
      errorMessage.value = readError(error);
    }
  } finally {
    void nextTick(initializeIcons);
  }
}

async function loadInitialData(): Promise<void> {
  const [loadedPosts, loadedSettings] = await Promise.all([fetchPosts(), fetchSettings()]);
  posts.value = loadedPosts;
  siteSettings.value = loadedSettings;
}

async function handleLogin(): Promise<void> {
  isLoggingIn.value = true;
  loginError.value = "";
  try {
    await loginAdmin(loginForm.value.username.trim(), loginForm.value.password);
    isAuthenticated.value = true;
    statusMessage.value = "登录成功。";
    loginForm.value.password = "";
    await loadInitialData();
    activeView.value = "dashboard";
  } catch (error) {
    loginError.value = readError(error);
  } finally {
    isLoggingIn.value = false;
    void nextTick(initializeIcons);
  }
}

async function handleLogout(): Promise<void> {
  try {
    await logoutAdmin();
  } finally {
    isAuthenticated.value = false;
    activeView.value = "dashboard";
    statusMessage.value = "";
    errorMessage.value = "";
    posts.value = [];
  }
}

async function loadPosts(): Promise<void> {
  errorMessage.value = "";
  try {
    posts.value = await fetchPosts();
  } catch (error) {
    handleRequestError(error);
  }
}

async function loadSettings(): Promise<void> {
  errorMessage.value = "";
  try {
    siteSettings.value = await fetchSettings();
  } catch (error) {
    handleRequestError(error);
  }
}

function switchView(nextView: AdminView): void {
  activeView.value = nextView;
  errorMessage.value = "";
  statusMessage.value = "";
  if (nextView === "settings") void loadSettings();
  if (nextView === "posts" || nextView === "dashboard") void loadPosts();
  if (nextView === "editor") schedulePreview();
}

async function openEditorForPost(postID: string): Promise<void> {
  errorMessage.value = "";
  try {
    const loadedPost = await fetchPost(postID);
    editorMode.value = "edit";
    editorForm.value = normalizePostDetail(loadedPost);
    aliasesText.value = loadedPost.aliases.join("\n");
    tagDraft.value = "";
    activeView.value = "editor";
    statusMessage.value = `已打开：${loadedPost.title}`;
    schedulePreview();
  } catch (error) {
    handleRequestError(error);
  }
}

function startNewPost(): void {
  editorMode.value = "create";
  editorForm.value = createEmptyPost();
  aliasesText.value = "";
  tagDraft.value = "";
  previewHTML.value = "";
  errorMessage.value = "";
  statusMessage.value = "正在新建文章。";
  activeView.value = "editor";
  schedulePreview();
}

async function saveCurrentPost(): Promise<void> {
  if (activeView.value !== "editor") return;
  addTagFromDraft();
  isSaving.value = true;
  errorMessage.value = "";
  try {
    const savePostRequest = buildSavePostRequest();
    const postDetailResponse =
      editorMode.value === "edit" && editorForm.value.id !== ""
        ? await updatePost(editorForm.value.id, savePostRequest)
        : await createPost(savePostRequest);
    editorMode.value = "edit";
    editorForm.value = normalizePostDetail(postDetailResponse.post);
    aliasesText.value = postDetailResponse.post.aliases.join("\n");
    tagDraft.value = "";
    statusMessage.value = savedPostMessage(postDetailResponse.post);
    await loadPosts();
  } catch (error) {
    handleRequestError(error);
  } finally {
    isSaving.value = false;
  }
}

function requestDeleteCurrentPost(): void {
  if (!canEditExistingPost.value) return;
  isDeleteDialogOpen.value = true;
}

async function confirmDeleteCurrentPost(): Promise<void> {
  if (!canEditExistingPost.value) return;
  isSaving.value = true;
  errorMessage.value = "";
  try {
    await deletePost(editorForm.value.id);
    statusMessage.value = "文章已删除，站点已自动更新。";
    isDeleteDialogOpen.value = false;
    activeView.value = "posts";
    editorMode.value = "create";
    editorForm.value = createEmptyPost();
    await loadPosts();
  } catch (error) {
    handleRequestError(error);
  } finally {
    isSaving.value = false;
  }
}

async function saveSettings(): Promise<void> {
  isSaving.value = true;
  errorMessage.value = "";
  try {
    const settingsResponse = await updateSettings(siteSettings.value);
    siteSettings.value = settingsResponse.settings;
    statusMessage.value = "站点设置已保存，站点已自动更新。";
    await loadPosts();
  } catch (error) {
    handleRequestError(error);
  } finally {
    isSaving.value = false;
  }
}
function schedulePreview(): void {
  if (previewTimerID !== undefined) window.clearTimeout(previewTimerID);
  previewTimerID = window.setTimeout(() => void refreshPreview(), 300);
}

async function refreshPreview(): Promise<void> {
  if (activeView.value !== "editor") return;
  isPreviewLoading.value = true;
  try {
    previewHTML.value = await previewMarkdown(editorForm.value.body);
  } catch (error) {
    previewHTML.value = `<p>${escapeHTML(readError(error))}</p>`;
  } finally {
    isPreviewLoading.value = false;
  }
}

function handleTagKeydown(event: KeyboardEvent): void {
  if (event.key === "Enter" || event.key === "," || event.key === "，") {
    event.preventDefault();
    addTagFromDraft();
    return;
  }
  if (event.key === "Backspace" && tagDraft.value === "" && editorForm.value.tags.length > 0) {
    editorForm.value.tags = editorForm.value.tags.slice(0, -1);
  }
}

function addTagFromDraft(): void {
  const rawTags = tagDraft.value
    .split(/[\n,，]/)
    .map((tagText) => tagText.trim())
    .filter((tagText) => tagText !== "");
  rawTags.forEach((tagText) => addTag(tagText));
  tagDraft.value = "";
}

function addTag(tagText: string): void {
  const normalizedTag = tagText.trim();
  if (normalizedTag === "" || editorForm.value.tags.includes(normalizedTag)) return;
  editorForm.value.tags = [...editorForm.value.tags, normalizedTag];
}

function removeTag(tagIndex: number): void {
  editorForm.value.tags = editorForm.value.tags.filter((_, currentIndex) => currentIndex !== tagIndex);
}

function handleGlobalKeydown(event: KeyboardEvent): void {
  const isSaveShortcut = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s";
  if (!isSaveShortcut || !isAuthenticated.value) return;
  event.preventDefault();
  if (activeView.value === "settings") {
    void saveSettings();
    return;
  }
  if (activeView.value === "editor") void saveCurrentPost();
}

function cycleAdminTheme(): void {
  const nextTheme = nextThemeMode(adminTheme.value);
  adminTheme.value = nextTheme;
  saveTheme(nextTheme);
  applyTheme(nextTheme);
  initializeIcons();
}

function initializeIcons(): void {
  const lucide = (window as LucideWindow).lucide;
  lucide?.createIcons({ nameAttr: "data-lucide" });
}

function applyTheme(themeMode: AdminThemeMode): void {
  document.documentElement.dataset.theme = themeMode;
}

function nextThemeMode(themeMode: AdminThemeMode): AdminThemeMode {
  const currentThemeIndex = themeModes.indexOf(themeMode);
  return themeModes[(currentThemeIndex + 1) % themeModes.length];
}

function readStoredTheme(): AdminThemeMode {
  try {
    const storedTheme = window.localStorage.getItem(themeStorageKey);
    if (storedTheme === "light" || storedTheme === "dark" || storedTheme === "auto") return storedTheme;
  } catch {
    return readDocumentDefaultTheme();
  }
  return readDocumentDefaultTheme();
}

function saveTheme(themeMode: AdminThemeMode): void {
  try {
    window.localStorage.setItem(themeStorageKey, themeMode);
  } catch {
    return;
  }
}

function readDocumentDefaultTheme(): AdminThemeMode {
  const defaultTheme = document.documentElement.dataset.theme;
  if (defaultTheme === "light" || defaultTheme === "dark" || defaultTheme === "auto") return defaultTheme;
  return "auto";
}

function buildSavePostRequest(): SavePostRequest {
  return {
    id: editorMode.value === "create" ? "" : editorForm.value.id,
    title: editorForm.value.title,
    icon: editorForm.value.icon,
    date: editorForm.value.date,
    description: editorForm.value.description,
    seoTitle: editorForm.value.seoTitle,
    seoDescription: editorForm.value.seoDescription,
    draft: editorForm.value.draft,
    url: editorForm.value.url,
    aliases: aliasesText.value
      .split("\n")
      .map((aliasText) => aliasText.trim())
      .filter((aliasText) => aliasText !== ""),
    tags: editorForm.value.tags,
    body: editorForm.value.body,
  };
}

function createEmptyPost(): PostDetail {
  return {
    id: "",
    title: "未命名文章",
    icon: "",
    date: formatCurrentDate(),
    description: "",
    seoTitle: "",
    seoDescription: "",
    draft: false,
    url: "new-post.html",
    aliases: [],
    tags: [],
    body: "这里写 Markdown 正文。",
  };
}

function normalizePostDetail(postDetail: PostDetail): PostDetail {
  return {
    ...postDetail,
    icon: postDetail.icon ?? "",
    seoTitle: postDetail.seoTitle ?? "",
    seoDescription: postDetail.seoDescription ?? "",
    aliases: postDetail.aliases ?? [],
    tags: postDetail.tags ?? [],
  };
}

function setPostDateToNow(): void {
  editorForm.value.date = formatCurrentDate();
}

function createEmptySiteSettings(): SiteSettings {
  return {
    title: "",
    description: "",
    iconUrl: "",
    commentEnabled: false,
    giscusRepo: "",
    giscusRepoId: "",
    giscusCategory: "",
    giscusCategoryId: "",
    themeDefault: "auto",
    font: "default",
  };
}

function formatCurrentDate(): string {
  const currentDate = new Date();
  const year = currentDate.getFullYear();
  const month = String(currentDate.getMonth() + 1).padStart(2, "0");
  const day = String(currentDate.getDate()).padStart(2, "0");
  const hour = String(currentDate.getHours()).padStart(2, "0");
  const minute = String(currentDate.getMinutes()).padStart(2, "0");
  const second = String(currentDate.getSeconds()).padStart(2, "0");
  return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
}

function handleRequestError(error: unknown): void {
  if (error instanceof UnauthorizedError) {
    isAuthenticated.value = false;
    loginError.value = "登录状态已失效，请重新登录。";
    return;
  }
  errorMessage.value = readError(error);
}

function readError(error: unknown): string {
  if (error instanceof UnauthorizedError) return error.message;
  if (error instanceof Error) return userMessageFromBackendError(error.message);
  return "发生未知错误。";
}

function savedPostMessage(postDetail: PostDetail): string {
  if (postDetail.draft) return "草稿已保存，未生成公开页面。";
  return "文章已保存，站点已自动更新。";
}

function userMessageFromBackendError(errorText: string): string {
  const normalizedError = errorText.toLowerCase();
  if (normalizedError.includes("decode request json")) return "请求内容格式不正确。";
  if (normalizedError.includes("authentication required") || normalizedError.includes("unauthorized")) return "请先登录后台。";
  if (normalizedError.includes("invalid username or password")) return "账号或密码不正确。";
  if (normalizedError.includes("title is empty")) return "请输入文章标题。";
  if (normalizedError.includes("date is empty")) return "请输入发布时间。";
  if (normalizedError.includes("date must use")) return "发布时间格式必须是 YYYY-MM-DD HH:mm:ss。";
  if (normalizedError.includes("permalink is empty")) return "请输入固定链接。";
  if (normalizedError.includes("permalink conflict") || normalizedError.includes("alias conflict")) return "固定链接或别名已被使用，请换一个链接。";
  if (normalizedError.includes("permalink")) return "固定链接格式不正确。";
  if (normalizedError.includes("markdown file name")) return "文章文件名不正确。";
  if (normalizedError.includes("default theme")) return "默认主题只能是自动、明亮或暗色。";
  if (normalizedError.includes("site font")) return "站点字体配置不正确。";
  if (normalizedError.includes("site icon")) return "网站 Icon 只支持 http(s) 链接或 / 开头的站内路径。";
  if (normalizedError.includes("read post") || normalizedError.includes("read posts")) return "读取文章失败。";
  if (normalizedError.includes("write") || normalizedError.includes("save")) return "保存失败，请稍后重试。";
  if (normalizedError.includes("render")) return "站点渲染失败，请检查文章内容或配置。";
  return "请求失败，请稍后重试。";
}

function escapeHTML(rawText: string): string {
  return rawText
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}
</script>
<template>
  <main v-if="!isAuthenticated" class="login-page">
    <section class="login-card card">
      <div class="login-brand">
        <img class="brand-logo" src="/honepress-black.svg" alt="" />
        <div>
          <p class="eyebrow">HonePress</p>
          <h1>登录后台</h1>
        </div>
      </div>
      <p class="login-copy">继续管理文章、草稿和站点配置。</p>
      <form class="login-form" @submit.prevent="handleLogin">
        <label class="form-field">
          <span>用户名</span>
          <input v-model="loginForm.username" autocomplete="username" placeholder="请输入后台用户名" type="text" />
        </label>
        <label class="form-field">
          <span>密码</span>
          <input v-model="loginForm.password" autocomplete="current-password" placeholder="请输入后台密码" type="password" />
        </label>
        <p v-if="loginError" class="form-error">{{ loginError }}</p>
        <button class="button button-primary" type="submit" :disabled="isLoggingIn">
          <i data-lucide="log-in" aria-hidden="true"></i>
          {{ isLoggingIn ? "正在登录" : "登录" }}
        </button>
      </form>
    </section>
  </main>

  <div v-else class="admin-app">
    <aside class="app-sidebar">
      <a class="sidebar-brand" href="/" target="_blank" rel="noreferrer">
        <img class="brand-logo" src="/honepress-black.svg" alt="" />
        <span>HonePress</span>
      </a>

      <nav class="sidebar-nav" aria-label="后台导航">
        <button type="button" class="nav-item" :class="{ active: activeView === 'dashboard' }"
          @click="switchView('dashboard')">
          <i data-lucide="layout-dashboard" aria-hidden="true"></i>
          <span>首页</span>
        </button>
        <button type="button" class="nav-item" :class="{ active: activeView === 'posts' }" @click="switchView('posts')">
          <i data-lucide="file-text" aria-hidden="true"></i>
          <span>文章列表</span>
        </button>
        <button type="button" class="nav-item" :class="{ active: activeView === 'editor' && editorMode === 'create' }"
          @click="startNewPost">
          <i data-lucide="square-pen" aria-hidden="true"></i>
          <span>新建文章</span>
        </button>
        <button type="button" class="nav-item" :class="{ active: activeView === 'settings' }"
          @click="switchView('settings')">
          <i data-lucide="settings" aria-hidden="true"></i>
          <span>站点配置</span>
        </button>
      </nav>

      <div class="sidebar-footer">
        <button type="button" class="button button-ghost compact" :title="adminThemeLabel" @click="cycleAdminTheme">
          <i class="theme-icon theme-icon-sun" data-lucide="sun" aria-hidden="true"></i>
          <i class="theme-icon theme-icon-moon" data-lucide="moon" aria-hidden="true"></i>
          <span>{{ themeLabels[adminTheme] }}</span>
        </button>
      </div>
    </aside>

    <main class="app-main">
      <header class="app-header">
        <div>
          <p class="eyebrow">HonePress</p>
          <h1>{{ pageTitle }}</h1>
          <p>{{ pageDescription }}</p>
        </div>
        <div class="header-actions">
          <button type="button" class="button button-outline" @click="handleLogout">
            <i data-lucide="log-out" aria-hidden="true"></i>
            退出
          </button>
          <button v-if="activeView !== 'editor'" type="button" class="button button-primary" @click="startNewPost">
            <i data-lucide="plus" aria-hidden="true"></i>
            新建文章
          </button>
          <button v-else type="button" class="button button-primary" :disabled="isSaving" @click="saveCurrentPost">
            <i data-lucide="save" aria-hidden="true"></i>
            {{ isSaving ? "正在保存" : "保存文章" }}
          </button>
        </div>
      </header>

      <div v-if="statusMessage || errorMessage" class="toast" :class="{ error: errorMessage }" role="status">
        <i :data-lucide="errorMessage ? 'circle-alert' : 'circle-check'" aria-hidden="true"></i>
        <span>{{ errorMessage || statusMessage }}</span>
      </div>

      <section v-if="activeView === 'dashboard'" class="dashboard-grid">
        <article class="metric-card card">
          <span class="metric-icon"><i data-lucide="file-text" aria-hidden="true"></i></span>
          <p>全部文章</p>
          <strong>{{ posts.length }}</strong>
        </article>
        <article class="metric-card card">
          <span class="metric-icon"><i data-lucide="send" aria-hidden="true"></i></span>
          <p>已发布</p>
          <strong>{{ publishedPosts.length }}</strong>
        </article>
        <article class="metric-card card">
          <span class="metric-icon"><i data-lucide="file-clock" aria-hidden="true"></i></span>
          <p>草稿</p>
          <strong>{{ draftPosts.length }}</strong>
        </article>

        <section class="card dashboard-panel recent-panel">
          <div class="card-heading">
            <div>
              <h2>最近文章</h2>
              <p>继续写作或查看刚发布的内容。</p>
            </div>
            <button class="button button-outline" type="button" @click="switchView('posts')">查看全部</button>
          </div>
          <div class="simple-list">
            <button v-for="post in recentPosts" :key="post.id" type="button" class="simple-list-row"
              @click="openEditorForPost(post.id)">
              <span>
                <strong>{{ post.title }}</strong>
                <small>{{ post.date }}</small>
              </span>
              <span class="badge" :class="post.draft ? 'badge-muted' : 'badge-success'">
                {{ post.draft ? "草稿" : "已发布" }}
              </span>
            </button>
            <p v-if="recentPosts.length === 0" class="empty-state">还没有文章，先写第一篇吧。</p>
          </div>
        </section>

        <section class="card dashboard-panel quick-panel">
          <div class="card-heading">
            <div>
              <h2>快捷入口</h2>
            </div>
          </div>
          <div class="quick-actions">
            <button type="button" class="quick-action" @click="startNewPost">
              <i data-lucide="square-pen" aria-hidden="true"></i>
              <span>写新文章</span>
            </button>
            <button type="button" class="quick-action" @click="switchView('settings')">
              <i data-lucide="sliders-horizontal" aria-hidden="true"></i>
              <span>站点配置</span>
            </button>
          </div>
        </section>
      </section>
      <section v-else-if="activeView === 'posts'" class="card table-card">
        <div class="card-heading">
          <div>
            <h2>文章列表</h2>
            <p>{{ posts.length }} 篇文章，{{ draftPosts.length }} 篇草稿。</p>
          </div>
          <button class="button button-primary" type="button" @click="startNewPost">
            <i data-lucide="plus" aria-hidden="true"></i>
            新建文章
          </button>
        </div>
        <div class="table-wrap">
          <table class="data-table">
            <thead>
              <tr>
                <th>标题</th>
                <th>状态</th>
                <th>发布时间</th>
                <th>固定链接</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="post in posts" :key="post.id">
                <td>
                  <button type="button" class="title-button" @click="openEditorForPost(post.id)">{{ post.title
                    }}</button>
                  <p>{{ post.description || "没有文章摘要" }}</p>
                </td>
                <td>
                  <span class="badge" :class="post.draft ? 'badge-muted' : 'badge-success'">
                    {{ post.draft ? "草稿" : "已发布" }}
                  </span>
                </td>
                <td>{{ post.date }}</td>
                <td><code>{{ post.url }}</code></td>
                <td class="table-actions">
                  <button class="button button-ghost icon-only" type="button" title="编辑"
                    @click="openEditorForPost(post.id)">
                    <i data-lucide="pencil" aria-hidden="true"></i>
                  </button>
                  <a v-if="!post.draft" class="button button-ghost icon-only" :href="`/${post.url}`" target="_blank"
                    rel="noreferrer" title="查看">
                    <i data-lucide="external-link" aria-hidden="true"></i>
                  </a>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-if="posts.length === 0" class="empty-state">还没有文章，点击右上角新建文章。</p>
        </div>
      </section>

      <section v-else-if="activeView === 'editor'" class="editor-layout">
        <section class="editor-main card">
          <div class="card-heading editor-heading">
            <div>
              <p class="eyebrow">{{ editorMode === "create" ? "新建文章" : "编辑文章" }}</p>
              <h2>{{ editorTitle }}</h2>
            </div>
            <div class="editor-actions">
              <a v-if="canEditExistingPost && !editorForm.draft" class="button button-outline"
                :href="`/${editorForm.url}`" target="_blank" rel="noreferrer">
                <i data-lucide="external-link" aria-hidden="true"></i>
                查看
              </a>
              <button type="button" class="button button-outline" :disabled="!canEditExistingPost || isSaving"
                @click="requestDeleteCurrentPost">
                <i data-lucide="trash-2" aria-hidden="true"></i>
                删除
              </button>
            </div>
          </div>
          <Suspense>
            <MarkdownWorkspace v-model="editorForm.body" :preview-html="previewHTML"
              :is-preview-loading="isPreviewLoading" />
            <template #fallback>
              <div class="editor-loading">正在加载 Markdown 编辑器</div>
            </template>
          </Suspense>
        </section>

        <aside class="meta-column">
          <section class="card meta-card">
            <div class="card-heading compact-heading">
              <h2>文章设置</h2>
            </div>
            <div class="form-stack">
              <label class="form-field">
                <span>标题</span>
                <input v-model="editorForm.title" type="text" placeholder="输入文章标题" />
              </label>
              <label class="form-field">
                <span>发布时间</span>
                <div class="input-button-row">
                  <input v-model="editorForm.date" type="text" placeholder="YYYY-MM-DD HH:mm:ss" />
                  <button type="button" class="button button-outline" @click="setPostDateToNow">生成</button>
                </div>
              </label>
              <label class="form-field">
                <span>固定链接</span>
                <input v-model="editorForm.url" type="text" placeholder="example-post.html" />
              </label>
              <label class="form-field">
                <span>摘要</span>
                <textarea v-model="editorForm.description" rows="3" placeholder="用于首页、归档和 SEO 的简短摘要"></textarea>
              </label>
              <label class="switch-field">
                <input v-model="editorForm.draft" type="checkbox" />
                <span>保存为草稿</span>
              </label>
            </div>
          </section>

          <section class="card meta-card">
            <div class="card-heading compact-heading">
              <h2>补充信息</h2>
            </div>
            <div class="form-stack">
              <label class="form-field">
                <span>Emoji</span>
                <select v-model="editorForm.icon">
                  <option v-for="emojiOption in emojiOptions" :key="emojiOption.label" :value="emojiOption.value">
                    {{ emojiOption.label }}
                  </option>
                </select>
              </label>
              <label class="form-field">
                <span>标签</span>
                <div class="tag-editor">
                  <span v-for="(tag, tagIndex) in editorForm.tags" :key="tag" class="tag-chip">
                    {{ tag }}
                    <button type="button" :aria-label="`删除标签 ${tag}`" @click="removeTag(tagIndex)">
                      <span aria-hidden="true">×</span>
                    </button>
                  </span>
                  <input v-model="tagDraft" type="text" placeholder="输入标签后回车" @keydown="handleTagKeydown"
                    @blur="addTagFromDraft" />
                </div>
              </label>
              <label class="form-field">
                <span>别名链接</span>
                <textarea v-model="aliasesText" rows="2" placeholder="每行一个别名链接"></textarea>
              </label>
            </div>
          </section>

          <section class="card meta-card">
            <div class="card-heading compact-heading">
              <h2>SEO 设置</h2>
            </div>
            <div class="form-stack">
              <label class="form-field">
                <span>SEO 标题</span>
                <input v-model="editorForm.seoTitle" type="text" placeholder="留空则使用文章标题" />
              </label>
              <label class="form-field">
                <span>SEO 描述</span>
                <textarea v-model="editorForm.seoDescription" rows="3" placeholder="留空则使用文章摘要"></textarea>
              </label>
            </div>
          </section>
        </aside>
      </section>
      <section v-else class="settings-layout">
        <section class="card settings-card">
          <div class="card-heading">
            <div>
              <h2>基础信息</h2>
            </div>
            <button type="button" class="button button-primary" :disabled="isSaving" @click="saveSettings">
              <i data-lucide="save" aria-hidden="true"></i>
              {{ isSaving ? "正在保存" : "保存设置" }}
            </button>
          </div>
          <div class="settings-grid">
            <label class="form-field">
              <span>站点标题</span>
              <input v-model="siteSettings.title" type="text" placeholder="请输入站点标题" />
            </label>
            <label class="form-field">
              <span>站点描述</span>
              <input v-model="siteSettings.description" type="text" placeholder="请输入一句简短的站点描述" />
            </label>
            <label class="form-field">
              <span>默认主题</span>
              <select v-model="siteSettings.themeDefault">
                <option value="auto">自动</option>
                <option value="light">明亮</option>
                <option value="dark">暗色</option>
              </select>
            </label>
            <label class="form-field">
              <span>字体</span>
              <select v-model="siteSettings.font">
                <option value="default">默认字体</option>
                <option value="douyin-sans">抖音美好体</option>
              </select>
            </label>
            <label class="form-field wide-field">
              <span>网站 icon URL</span>
              <input v-model="siteSettings.iconUrl" type="text"
                placeholder="填写网站 Icon URL，例如 https://example.com/favicon.png" />
            </label>
          </div>
        </section>

        <section class="card settings-card">
          <div class="card-heading compact-heading">
            <h2>评论设置</h2>
          </div>
          <div class="settings-grid">
            <label class="switch-field wide-field">
              <input v-model="siteSettings.commentEnabled" type="checkbox" />
              <span>开启 giscus 评论</span>
            </label>
            <template v-if="siteSettings.commentEnabled">
              <label class="form-field">
                <span>GitHub 仓库</span>
                <input v-model="siteSettings.giscusRepo" type="text" placeholder="owner/repo" />
              </label>
              <label class="form-field">
                <span>仓库 ID</span>
                <input v-model="siteSettings.giscusRepoId" type="text" />
              </label>
              <label class="form-field">
                <span>讨论分类</span>
                <input v-model="siteSettings.giscusCategory" type="text" />
              </label>
              <label class="form-field">
                <span>分类 ID</span>
                <input v-model="siteSettings.giscusCategoryId" type="text" />
              </label>
            </template>
          </div>
        </section>
      </section>
    </main>

    <div v-if="isDeleteDialogOpen" class="dialog-layer" role="dialog" aria-modal="true"
      aria-labelledby="delete-dialog-title">
      <section class="dialog-card card">
        <div class="dialog-icon"><i data-lucide="trash-2" aria-hidden="true"></i></div>
        <h2 id="delete-dialog-title">删除文章</h2>
        <p>确定删除「{{ editorTitle }}」吗？这个操作会同步更新静态站点。</p>
        <div class="dialog-actions">
          <button type="button" class="button button-outline" @click="isDeleteDialogOpen = false">取消</button>
          <button type="button" class="button button-danger" :disabled="isSaving"
            @click="confirmDeleteCurrentPost">删除</button>
        </div>
      </section>
    </div>
  </div>
</template>
