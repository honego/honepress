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
  uploadSiteIcon,
} from "./api/post";
import type { PostDetail, PostSummary, SavePostRequest, SiteSettings } from "./types/post";

type EditorMode = "create" | "edit";
type AdminSection = "posts" | "settings";
type AdminThemeMode = "auto" | "light" | "dark";
type MarkdownAction = "bold" | "italic" | "heading" | "quote" | "code" | "codeblock" | "link" | "image" | "ul" | "ol";

const themeStorageKey = "honepress-theme";
const adminThemeModes: AdminThemeMode[] = ["auto", "light", "dark"];
const emojiOptions = [
  { value: "", label: "默认网站 icon" },
  { value: "☘️", label: "☘️ 日常" },
  { value: "🌱", label: "🌱 记录" },
  { value: "✨", label: "✨ 灵感" },
  { value: "📝", label: "📝 笔记" },
  { value: "💡", label: "💡 想法" },
  { value: "🚀", label: "🚀 项目" },
  { value: "📌", label: "📌 摘录" },
  { value: "🌙", label: "🌙 夜读" },
  { value: "🧩", label: "🧩 技术" },
];

const activeSection = ref<AdminSection>("posts");
const posts = ref<PostSummary[]>([]);
const editorMode = ref<EditorMode>("create");
const isEditorOpen = ref(false);
const editorForm = ref<PostDetail>(createEmptyPost());
const aliasesText = ref("");
const tagDraft = ref("");
const previewHTML = ref("");
const statusMessage = ref("");
const errorMessage = ref("");
const isSaving = ref(false);
const isPreviewLoading = ref(false);
const adminTheme = ref<AdminThemeMode>(readStoredAdminTheme());
const siteSettings = ref<SiteSettings>(createEmptySiteSettings());
const siteIconFileInput = ref<HTMLInputElement | null>(null);
const markdownTextarea = ref<HTMLTextAreaElement | null>(null);

let previewTimerID: number | undefined;

const selectedPostID = computed(() => (isEditorOpen.value && editorMode.value === "edit" ? editorForm.value.id : ""));
const canEditExistingPost = computed(() => isEditorOpen.value && editorMode.value === "edit" && selectedPostID.value !== "");
const adminThemeLabel = computed(() => {
  if (adminTheme.value === "light") {
    return "主题：亮色";
  }
  if (adminTheme.value === "dark") {
    return "主题：暗色";
  }
  return "主题：自动";
});
const pageTitle = computed(() => (activeSection.value === "posts" ? "文章" : "设置"));

onMounted(() => {
  applyAdminTheme();
  window.addEventListener("keydown", handleGlobalKeydown);
  void loadPosts();
  void loadSettings();
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
    if (isEditorOpen.value) {
      schedulePreview();
    }
  },
);

async function loadPosts(): Promise<void> {
  errorMessage.value = "";
  try {
    posts.value = await fetchPosts();
  } catch (error) {
    errorMessage.value = readError(error);
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
  } else if (isEditorOpen.value) {
    schedulePreview();
  }
}

async function selectPost(postID: string): Promise<void> {
  activeSection.value = "posts";
  errorMessage.value = "";
  try {
    const loadedPost = await fetchPost(postID);
    editorMode.value = "edit";
    isEditorOpen.value = true;
    editorForm.value = normalizePostDetail(loadedPost);
    aliasesText.value = loadedPost.aliases.join("\n");
    tagDraft.value = "";
    statusMessage.value = `已打开：${loadedPost.title}`;
    schedulePreview();
  } catch (error) {
    errorMessage.value = readError(error);
  }
}

function createNewPost(): void {
  activeSection.value = "posts";
  editorMode.value = "create";
  isEditorOpen.value = true;
  editorForm.value = createEmptyPost();
  aliasesText.value = "";
  tagDraft.value = "";
  statusMessage.value = "正在新建文章。";
  errorMessage.value = "";
  schedulePreview();
}

async function saveCurrentPost(): Promise<void> {
  if (!isEditorOpen.value) {
    return;
  }
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
    isEditorOpen.value = true;
    editorForm.value = normalizePostDetail(postDetailResponse.post);
    aliasesText.value = postDetailResponse.post.aliases.join("\n");
    tagDraft.value = "";
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
    closeEditor();
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

function chooseSiteIcon(): void {
  siteIconFileInput.value?.click();
}

async function handleSiteIconUpload(event: Event): Promise<void> {
  const inputElement = event.currentTarget as HTMLInputElement;
  const iconFile = inputElement.files?.[0];
  if (iconFile === undefined) {
    return;
  }

  isSaving.value = true;
  errorMessage.value = "";
  try {
    const settingsResponse = await uploadSiteIcon(iconFile);
    siteSettings.value = settingsResponse.settings;
    statusMessage.value = settingsResponse.message ?? "网站 icon 已上传。";
  } catch (error) {
    errorMessage.value = readError(error);
  } finally {
    inputElement.value = "";
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
  if (!isEditorOpen.value) {
    return;
  }
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
  const shortcutKey = event.key.toLowerCase();
  if ((event.ctrlKey || event.metaKey) && shortcutKey === "b") {
    event.preventDefault();
    await applyMarkdownAction("bold");
    return;
  }
  if ((event.ctrlKey || event.metaKey) && shortcutKey === "i") {
    event.preventDefault();
    await applyMarkdownAction("italic");
    return;
  }
  if ((event.ctrlKey || event.metaKey) && shortcutKey === "k") {
    event.preventDefault();
    await applyMarkdownAction("link");
    return;
  }
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

async function applyMarkdownAction(action: MarkdownAction): Promise<void> {
  const textareaElement = markdownTextarea.value;
  if (textareaElement === null) {
    return;
  }
  const selectionStart = textareaElement.selectionStart;
  const selectionEnd = textareaElement.selectionEnd;
  const selectedText = editorForm.value.body.slice(selectionStart, selectionEnd);
  let replacementText = selectedText;
  let nextSelectionStart = selectionStart;
  let nextSelectionEnd = selectionEnd;

  if (action === "bold") {
    replacementText = `**${selectedText || "加粗文字"}**`;
    nextSelectionStart = selectionStart + 2;
    nextSelectionEnd = selectionStart + replacementText.length - 2;
  } else if (action === "italic") {
    replacementText = `*${selectedText || "斜体文字"}*`;
    nextSelectionStart = selectionStart + 1;
    nextSelectionEnd = selectionStart + replacementText.length - 1;
  } else if (action === "code") {
    replacementText = `\`${selectedText || "code"}\``;
    nextSelectionStart = selectionStart + 1;
    nextSelectionEnd = selectionStart + replacementText.length - 1;
  } else if (action === "codeblock") {
    replacementText = `\`\`\`shell\n${selectedText || "echo hello"}\n\`\`\``;
    nextSelectionStart = selectionStart + 9;
    nextSelectionEnd = selectionStart + replacementText.length - 4;
  } else if (action === "link") {
    replacementText = `[${selectedText || "链接文字"}](https://example.com)`;
    nextSelectionStart = selectionStart + 1;
    nextSelectionEnd = selectionStart + (selectedText || "链接文字").length + 1;
  } else if (action === "image") {
    replacementText = `![${selectedText || "图片描述"}](https://example.com/image.png)`;
    nextSelectionStart = selectionStart + 2;
    nextSelectionEnd = selectionStart + (selectedText || "图片描述").length + 2;
  } else if (action === "heading") {
    replacementText = prefixSelectedLines(selectedText || "小标题", "## ");
    nextSelectionEnd = selectionStart + replacementText.length;
  } else if (action === "quote") {
    replacementText = prefixSelectedLines(selectedText || "引用内容", "> ");
    nextSelectionEnd = selectionStart + replacementText.length;
  } else if (action === "ul") {
    replacementText = prefixSelectedLines(selectedText || "列表项", "- ");
    nextSelectionEnd = selectionStart + replacementText.length;
  } else if (action === "ol") {
    replacementText = prefixSelectedLines(selectedText || "列表项", "1. ");
    nextSelectionEnd = selectionStart + replacementText.length;
  }

  editorForm.value.body = `${editorForm.value.body.slice(0, selectionStart)}${replacementText}${editorForm.value.body.slice(selectionEnd)}`;
  await nextTick();
  textareaElement.focus();
  textareaElement.selectionStart = nextSelectionStart;
  textareaElement.selectionEnd = nextSelectionEnd;
}

function prefixSelectedLines(text: string, prefix: string): string {
  return text
    .split("\n")
    .map((lineText) => `${prefix}${lineText}`)
    .join("\n");
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
  if (normalizedTag === "" || editorForm.value.tags.includes(normalizedTag)) {
    return;
  }
  editorForm.value.tags = [...editorForm.value.tags, normalizedTag];
}

function removeTag(tagIndex: number): void {
  editorForm.value.tags = editorForm.value.tags.filter((_, currentIndex) => currentIndex !== tagIndex);
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
  if (!isEditorOpen.value) {
    return;
  }
  void saveCurrentPost();
}

function cycleAdminTheme(): void {
  const currentThemeIndex = adminThemeModes.indexOf(adminTheme.value);
  const nextThemeIndex = (currentThemeIndex + 1) % adminThemeModes.length;
  adminTheme.value = adminThemeModes[nextThemeIndex];
  try {
    window.localStorage.setItem(themeStorageKey, adminTheme.value);
  } catch {
    statusMessage.value = "浏览器阻止保存主题。";
  }
  applyAdminTheme();
}

function applyAdminTheme(): void {
  document.documentElement.dataset.adminTheme = adminTheme.value;
  document.documentElement.dataset.theme = adminTheme.value;
}

function readStoredAdminTheme(): AdminThemeMode {
  try {
    const storedTheme = window.localStorage.getItem(themeStorageKey);
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
    icon: editorForm.value.icon,
    date: editorForm.value.date,
    description: editorForm.value.description,
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

function closeEditor(): void {
  isEditorOpen.value = false;
  editorMode.value = "create";
  editorForm.value = createEmptyPost();
  aliasesText.value = "";
  tagDraft.value = "";
  previewHTML.value = "";
  if (previewTimerID !== undefined) {
    window.clearTimeout(previewTimerID);
    previewTimerID = undefined;
  }
}

function createEmptyPost(): PostDetail {
  return {
    id: "",
    title: "未命名文章",
    icon: "",
    date: formatCurrentDate(),
    description: "",
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
        <span>HonePress</span>
      </a>
      <div class="topbar-actions">
        <button type="button" @click="cycleAdminTheme">{{ adminThemeLabel }}</button>
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
            <p class="eyebrow">HonePress</p>
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
            <div class="post-list">
              <button v-for="post in posts" :key="post.id" type="button" class="post-row"
                :class="{ active: post.id === selectedPostID }" @click="selectPost(post.id)">
                <span>{{ post.title }}</span>
                <small>{{ post.date }}</small>
                <em>{{ post.draft ? "草稿" : "已发布" }}</em>
              </button>
            </div>
          </aside>

          <section v-if="isEditorOpen" class="editor-panel">
            <header class="editor-header">
              <div>
                <p class="eyebrow">{{ editorMode === "create" ? "新建文章" : "编辑文章" }}</p>
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
                <span>Emoji</span>
                <select v-model="editorForm.icon">
                  <option v-for="emojiOption in emojiOptions" :key="emojiOption.label" :value="emojiOption.value">
                    {{ emojiOption.label }}
                  </option>
                </select>
              </label>
              <label>
                <span>固定链接</span>
                <input v-model="editorForm.url" type="text" />
              </label>
              <label>
                <span>发布时间</span>
                <div class="inline-input-row">
                  <input v-model="editorForm.date" type="text" />
                  <button type="button" @click="setPostDateToNow">生成</button>
                </div>
              </label>
              <label>
                <span>摘要</span>
                <input v-model="editorForm.description" type="text" />
              </label>
              <label class="wide">
                <span>别名</span>
                <textarea v-model="aliasesText" rows="3"></textarea>
              </label>
              <label class="wide tag-editor-field">
                <span>标签</span>
                <div class="tag-editor">
                  <span v-for="(tag, tagIndex) in editorForm.tags" :key="tag" class="tag-chip">
                    {{ tag }}
                    <button type="button" :aria-label="`删除标签 ${tag}`" @click="removeTag(tagIndex)">×</button>
                  </span>
                  <input v-model="tagDraft" type="text" placeholder="输入标签后回车" @keydown="handleTagKeydown"
                    @blur="addTagFromDraft" />
                </div>
              </label>
              <div class="switches">
                <label><input v-model="editorForm.draft" type="checkbox" /> 草稿</label>
              </div>
            </section>

            <section class="workspace">
              <section class="markdown-editor">
                <div class="markdown-editor-head">
                  <span>Markdown</span>
                  <div class="markdown-toolbar" aria-label="Markdown 工具栏">
                    <button type="button" title="加粗 Ctrl+B" @click="applyMarkdownAction('bold')">B</button>
                    <button type="button" title="斜体 Ctrl+I" @click="applyMarkdownAction('italic')">I</button>
                    <button type="button" title="标题" @click="applyMarkdownAction('heading')">H</button>
                    <button type="button" title="引用" @click="applyMarkdownAction('quote')">&gt;</button>
                    <button type="button" title="行内代码" @click="applyMarkdownAction('code')">`</button>
                    <button type="button" title="代码块" @click="applyMarkdownAction('codeblock')">{ }</button>
                    <button type="button" title="链接 Ctrl+K" @click="applyMarkdownAction('link')">链</button>
                    <button type="button" title="图片" @click="applyMarkdownAction('image')">图</button>
                    <button type="button" title="无序列表" @click="applyMarkdownAction('ul')">-</button>
                    <button type="button" title="有序列表" @click="applyMarkdownAction('ol')">1.</button>
                  </div>
                </div>
                <textarea ref="markdownTextarea" v-model="editorForm.body" spellcheck="false"
                  @keydown="handleMarkdownKeydown"></textarea>
              </section>
              <div class="preview">
                <div class="preview-header">
                  <span>预览</span>
                  <small>{{ isPreviewLoading ? "更新中" : "已同步" }}</small>
                </div>
                <article class="preview-body" v-html="previewHTML"></article>
              </div>
            </section>
          </section>
          <section v-else class="editor-blank" aria-hidden="true"></section>
        </section>

        <section v-else class="settings-view">
          <section class="settings-card">
            <header class="card-header">
              <div>
                <h2>站点设置</h2>
              </div>
              <button type="button" :disabled="isSaving" @click="saveSettings">保存设置</button>
            </header>

            <div class="form-grid settings-grid">
              <section class="settings-section wide">
                <h3>基础信息</h3>
              </section>
              <label>
                <span>站点标题</span>
                <input v-model="siteSettings.title" type="text" />
              </label>
              <label>
                <span>站点描述</span>
                <input v-model="siteSettings.description" type="text" />
              </label>
              <label>
                <span>默认主题</span>
                <select v-model="siteSettings.themeDefault">
                  <option value="auto">自动</option>
                  <option value="light">明亮</option>
                  <option value="dark">暗色</option>
                </select>
              </label>
              <label>
                <span>字体</span>
                <select v-model="siteSettings.font">
                  <option value="default">默认字体</option>
                  <option value="douyin-sans">抖音美好体</option>
                </select>
              </label>
              <label class="site-icon-field">
                <span>网站 icon</span>
                <div class="site-icon-row">
                  <span class="site-icon-preview">
                    <img v-if="siteSettings.iconUrl" :src="siteSettings.iconUrl" alt="" />
                    <span v-else>b</span>
                  </span>
                  <input v-model="siteSettings.iconUrl" type="text" />
                  <button type="button" :disabled="isSaving" @click="chooseSiteIcon">上传</button>
                  <input ref="siteIconFileInput" class="visually-hidden" type="file"
                    accept=".ico,.png,.jpg,.jpeg,.webp,.svg,image/*" @change="handleSiteIconUpload" />
                </div>
              </label>
              <section class="settings-section wide">
                <h3>评论设置</h3>
              </section>
              <div class="switches wide">
                <label><input v-model="siteSettings.commentEnabled" type="checkbox" /> 开启 giscus 评论</label>
              </div>
              <template v-if="siteSettings.commentEnabled">
                <label>
                  <span>GitHub 仓库</span>
                  <input v-model="siteSettings.giscusRepo" type="text" placeholder="owner/repo" />
                </label>
                <label>
                  <span>仓库 ID</span>
                  <input v-model="siteSettings.giscusRepoId" type="text" />
                </label>
                <label>
                  <span>讨论分类</span>
                  <input v-model="siteSettings.giscusCategory" type="text" />
                </label>
                <label>
                  <span>分类 ID</span>
                  <input v-model="siteSettings.giscusCategoryId" type="text" />
                </label>
              </template>
            </div>
          </section>
        </section>
      </main>
    </div>
  </div>
</template>
