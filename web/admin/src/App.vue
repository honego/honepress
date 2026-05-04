<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";

import {
  createPost,
  deletePost,
  fetchPost,
  fetchPosts,
  fetchSettings,
  previewMarkdown,
  updatePost,
  updateSettings,
} from "./api/post";
import type { PostDetail, PostSummary, SavePostRequest, SiteSettings } from "./types/post";

type EditorMode = "create" | "edit";
type AdminSection = "posts" | "settings";
type AdminThemeMode = "auto" | "light" | "dark";

const adminThemeModes: AdminThemeMode[] = ["auto", "light", "dark"];
const activeSection = ref<AdminSection>("posts");
const posts = ref<PostSummary[]>([]);
const editorMode = ref<EditorMode>("create");
const editorForm = ref<PostDetail>(createEmptyPost());
const aliasesText = ref("");
const previewHTML = ref("");
const statusMessage = ref("");
const errorMessage = ref("");
const isLoading = ref(false);
const isSaving = ref(false);
const isPreviewLoading = ref(false);
const adminTheme = ref<AdminThemeMode>(readStoredAdminTheme());
const siteSettings = ref<SiteSettings>(createEmptySiteSettings());

let previewTimerID: number | undefined;

const selectedPostID = computed(() => editorForm.value.id);
const canEditExistingPost = computed(() => editorMode.value === "edit" && selectedPostID.value !== "");
const adminThemeLabel = computed(() => {
  if (adminTheme.value === "light") {
    return "明亮";
  }
  if (adminTheme.value === "dark") {
    return "暗色";
  }
  return "自动";
});
const pageTitle = computed(() => (activeSection.value === "posts" ? "文章" : "设置"));

onMounted(() => {
  applyAdminTheme();
  window.addEventListener("keydown", handleGlobalKeydown);
  void loadPosts();
  void loadSettings();
  schedulePreview();
});

onBeforeUnmount(() => {
  window.removeEventListener("keydown", handleGlobalKeydown);
  if (previewTimerID !== undefined) {
    window.clearTimeout(previewTimerID);
  }
});

watch(
  () => editorForm.value.body,
  () => {
    schedulePreview();
  },
);

async function loadPosts(): Promise<void> {
  isLoading.value = true;
  errorMessage.value = "";
  try {
    posts.value = await fetchPosts();
  } catch (error) {
    errorMessage.value = readError(error);
  } finally {
    isLoading.value = false;
  }
}

async function loadSettings(): Promise<void> {
  errorMessage.value = "";
  try {
    siteSettings.value = await fetchSettings();
  } catch (error) {
    errorMessage.value = readError(error);
  }
}

function switchSection(nextSection: AdminSection): void {
  activeSection.value = nextSection;
  errorMessage.value = "";
  statusMessage.value = "";
  if (nextSection === "settings") {
    void loadSettings();
  } else {
    schedulePreview();
  }
}

async function selectPost(postID: string): Promise<void> {
  activeSection.value = "posts";
  isLoading.value = true;
  errorMessage.value = "";
  try {
    const loadedPost = await fetchPost(postID);
    editorMode.value = "edit";
    editorForm.value = loadedPost;
    aliasesText.value = loadedPost.aliases.join("\n");
    statusMessage.value = `已打开：${loadedPost.title}`;
    schedulePreview();
  } catch (error) {
    errorMessage.value = readError(error);
  } finally {
    isLoading.value = false;
  }
}

function createNewPost(): void {
  activeSection.value = "posts";
  editorMode.value = "create";
  editorForm.value = createEmptyPost();
  aliasesText.value = "";
  statusMessage.value = "正在新建文章。";
  errorMessage.value = "";
  schedulePreview();
}

async function saveCurrentPost(): Promise<void> {
  isSaving.value = true;
  errorMessage.value = "";
  try {
    const savePostRequest = buildSavePostRequest();
    const postDetailResponse =
      editorMode.value === "edit" && editorForm.value.id !== ""
        ? await updatePost(editorForm.value.id, savePostRequest)
        : await createPost(savePostRequest);
    editorMode.value = "edit";
    editorForm.value = postDetailResponse.post;
    aliasesText.value = postDetailResponse.post.aliases.join("\n");
    statusMessage.value = postDetailResponse.message ?? "文章已保存。";
    await loadPosts();
  } catch (error) {
    errorMessage.value = readError(error);
  } finally {
    isSaving.value = false;
  }
}

async function deleteCurrentPost(): Promise<void> {
  if (!canEditExistingPost.value) {
    return;
  }
  const shouldDelete = window.confirm(`确定删除 ${editorForm.value.title} 吗？`);
  if (!shouldDelete) {
    return;
  }

  isSaving.value = true;
  errorMessage.value = "";
  try {
    const messageResponse = await deletePost(editorForm.value.id);
    statusMessage.value = messageResponse.message;
    createNewPost();
    await loadPosts();
  } catch (error) {
    errorMessage.value = readError(error);
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
    statusMessage.value = settingsResponse.message ?? "站点设置已保存。";
    await loadPosts();
  } catch (error) {
    errorMessage.value = readError(error);
  } finally {
    isSaving.value = false;
  }
}

function schedulePreview(): void {
  if (previewTimerID !== undefined) {
    window.clearTimeout(previewTimerID);
  }
  previewTimerID = window.setTimeout(() => {
    void refreshPreview();
  }, 300);
}

async function refreshPreview(): Promise<void> {
  isPreviewLoading.value = true;
  try {
    previewHTML.value = await previewMarkdown(editorForm.value.body);
  } catch (error) {
    previewHTML.value = `<p>${escapeHTML(readError(error))}</p>`;
  } finally {
    isPreviewLoading.value = false;
  }
}

async function handleMarkdownKeydown(event: KeyboardEvent): Promise<void> {
  if (event.key !== "Tab") {
    return;
  }
  event.preventDefault();
  const textareaElement = event.currentTarget as HTMLTextAreaElement;
  const selectionStart = textareaElement.selectionStart;
  const selectionEnd = textareaElement.selectionEnd;
  const beforeSelection = editorForm.value.body.slice(0, selectionStart);
  const afterSelection = editorForm.value.body.slice(selectionEnd);
  editorForm.value.body = `${beforeSelection}  ${afterSelection}`;
  await nextTick();
  textareaElement.selectionStart = selectionStart + 2;
  textareaElement.selectionEnd = selectionStart + 2;
}

function handleGlobalKeydown(event: KeyboardEvent): void {
  const isSaveShortcut = (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s";
  if (!isSaveShortcut) {
    return;
  }
  event.preventDefault();
  if (activeSection.value === "settings") {
    void saveSettings();
    return;
  }
  void saveCurrentPost();
}

function cycleAdminTheme(): void {
  const currentThemeIndex = adminThemeModes.indexOf(adminTheme.value);
  const nextThemeIndex = (currentThemeIndex + 1) % adminThemeModes.length;
  adminTheme.value = adminThemeModes[nextThemeIndex];
  try {
    window.localStorage.setItem("blog-admin-theme", adminTheme.value);
  } catch {
    statusMessage.value = "浏览器阻止保存后台主题。";
  }
  applyAdminTheme();
}

function applyAdminTheme(): void {
  document.documentElement.dataset.adminTheme = adminTheme.value;
}

function readStoredAdminTheme(): AdminThemeMode {
  try {
    const storedTheme = window.localStorage.getItem("blog-admin-theme");
    if (storedTheme === "light" || storedTheme === "dark" || storedTheme === "auto") {
      return storedTheme;
    }
  } catch {
    return "auto";
  }
  return "auto";
}

function buildSavePostRequest(): SavePostRequest {
  return {
    id: editorMode.value === "create" ? "" : editorForm.value.id,
    title: editorForm.value.title,
    date: editorForm.value.date,
    description: editorForm.value.description,
    draft: editorForm.value.draft,
    url: editorForm.value.url,
    aliases: aliasesText.value
      .split("\n")
      .map((aliasText) => aliasText.trim())
      .filter((aliasText) => aliasText !== ""),
    comments: editorForm.value.comments,
    body: editorForm.value.body,
  };
}

function createEmptyPost(): PostDetail {
  return {
    id: "",
    title: "未命名文章",
    date: formatCurrentDate(),
    description: "",
    draft: false,
    url: "new-post.html",
    aliases: [],
    comments: true,
    body: "这里写 Markdown 正文。",
  };
}

function createEmptySiteSettings(): SiteSettings {
  return {
    title: "",
    description: "",
    baseUrl: "",
    language: "zh-CN",
    githubUrl: "",
    telegramUrl: "",
    commentEnabled: false,
    themeDefault: "auto",
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

function readError(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "发生未知错误。";
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
  <div class="admin-shell">
    <header class="topbar">
      <a class="topbar-brand" href="/" target="_blank">
        <span class="brand-mark">b</span>
        <span>blog</span>
      </a>
      <div class="topbar-actions">
        <button type="button" @click="cycleAdminTheme">主题：{{ adminThemeLabel }}</button>
      </div>
    </header>

    <div class="admin-layout">
      <aside class="sidebar">
        <nav class="menu" aria-label="后台导航">
          <button type="button" class="menu-item" :class="{ active: activeSection === 'posts' }"
            @click="switchSection('posts')">
            <span class="menu-icon">文</span>
            <span>文章</span>
          </button>
          <button type="button" class="menu-item" :class="{ active: activeSection === 'settings' }"
            @click="switchSection('settings')">
            <span class="menu-icon">设</span>
            <span>设置</span>
          </button>
        </nav>
      </aside>

      <main class="main">
        <header class="page-header">
          <div>
            <p class="eyebrow">blog</p>
            <h1>{{ pageTitle }}</h1>
          </div>
          <button v-if="activeSection === 'posts'" type="button" @click="createNewPost">新建文章</button>
        </header>

        <p v-if="statusMessage" class="status">{{ statusMessage }}</p>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>

        <section v-if="activeSection === 'posts'" class="posts-view">
          <aside class="post-list-panel">
            <div class="panel-heading">
              <h2>文章列表</h2>
              <span>{{ posts.length }}</span>
            </div>
            <p v-if="isLoading" class="muted">正在加载...</p>
            <div class="post-list">
              <button v-for="post in posts" :key="post.id" type="button" class="post-row"
                :class="{ active: post.id === selectedPostID }" @click="selectPost(post.id)">
                <span>{{ post.title }}</span>
                <small>{{ post.date }}</small>
                <em>{{ post.draft ? "草稿" : "已发布" }}</em>
              </button>
            </div>
          </aside>

          <section class="editor-panel">
            <header class="editor-header">
              <div>
                <p class="eyebrow">{{ editorMode === "create" ? "新建文章" : editorForm.id }}</p>
                <h2>{{ editorForm.title }}</h2>
              </div>
              <div class="actions">
                <a v-if="canEditExistingPost && !editorForm.draft" :href="`/${editorForm.url}`" target="_blank">查看</a>
                <button type="button" :disabled="isSaving" @click="saveCurrentPost">保存</button>
                <button type="button" :disabled="!canEditExistingPost || isSaving" class="danger"
                  @click="deleteCurrentPost">
                  删除
                </button>
              </div>
            </header>

            <section class="form-grid">
              <label>
                <span>标题</span>
                <input v-model="editorForm.title" type="text" />
              </label>
              <label>
                <span>固定链接</span>
                <input v-model="editorForm.url" type="text" />
              </label>
              <label>
                <span>发布时间</span>
                <input v-model="editorForm.date" type="text" />
              </label>
              <label>
                <span>摘要</span>
                <input v-model="editorForm.description" type="text" />
              </label>
              <label class="wide">
                <span>别名</span>
                <textarea v-model="aliasesText" rows="3"></textarea>
              </label>
              <div class="switches">
                <label><input v-model="editorForm.draft" type="checkbox" /> 草稿</label>
                <label><input v-model="editorForm.comments" type="checkbox" /> 评论</label>
              </div>
            </section>

            <section class="workspace">
              <label class="markdown-editor">
                <span>Markdown</span>
                <textarea v-model="editorForm.body" spellcheck="false" @keydown="handleMarkdownKeydown"></textarea>
              </label>
              <div class="preview">
                <div class="preview-header">
                  <span>预览</span>
                  <small>{{ isPreviewLoading ? "更新中" : "已同步" }}</small>
                </div>
                <article class="preview-body" v-html="previewHTML"></article>
              </div>
            </section>
          </section>
        </section>

        <section v-else class="settings-view">
          <section class="settings-card">
            <header class="card-header">
              <div>
                <p class="eyebrow">config.yaml</p>
                <h2>站点设置</h2>
              </div>
              <button type="button" :disabled="isSaving" @click="saveSettings">保存设置</button>
            </header>

            <div class="form-grid settings-grid">
              <label>
                <span>站点标题</span>
                <input v-model="siteSettings.title" type="text" />
              </label>
              <label>
                <span>站点描述</span>
                <input v-model="siteSettings.description" type="text" />
              </label>
              <label>
                <span>语言</span>
                <input v-model="siteSettings.language" type="text" />
              </label>
              <label>
                <span>默认主题</span>
                <select v-model="siteSettings.themeDefault">
                  <option value="auto">自动</option>
                  <option value="light">明亮</option>
                  <option value="dark">暗色</option>
                </select>
              </label>
              <div class="switches wide">
                <label><input v-model="siteSettings.commentEnabled" type="checkbox" /> 开启评论</label>
              </div>
            </div>
          </section>
        </section>
      </main>
    </div>
  </div>
</template>
