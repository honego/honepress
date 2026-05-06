<script setup lang="ts">
import { nextTick, ref } from "vue";

type MarkdownAction = "bold" | "italic" | "link";

const props = defineProps<{
  modelValue: string;
  previewHtml: string;
  isPreviewLoading: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: string];
}>();

const textareaElement = ref<HTMLTextAreaElement | null>(null);

function handleInput(event: Event): void {
  emit("update:modelValue", (event.currentTarget as HTMLTextAreaElement).value);
}

async function handleKeydown(event: KeyboardEvent): Promise<void> {
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
  const currentTextareaElement = event.currentTarget as HTMLTextAreaElement;
  const selectionStart = currentTextareaElement.selectionStart;
  const selectionEnd = currentTextareaElement.selectionEnd;
  const nextValue = `${props.modelValue.slice(0, selectionStart)}  ${props.modelValue.slice(selectionEnd)}`;
  emit("update:modelValue", nextValue);
  await nextTick();
  currentTextareaElement.selectionStart = selectionStart + 2;
  currentTextareaElement.selectionEnd = selectionStart + 2;
}

async function applyMarkdownAction(action: MarkdownAction): Promise<void> {
  const currentTextareaElement = textareaElement.value;
  if (currentTextareaElement === null) {
    return;
  }
  const selectionStart = currentTextareaElement.selectionStart;
  const selectionEnd = currentTextareaElement.selectionEnd;
  const selectedText = props.modelValue.slice(selectionStart, selectionEnd);
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
  } else if (action === "link") {
    replacementText = `[${selectedText || "链接文字"}](https://example.com)`;
    nextSelectionStart = selectionStart + 1;
    nextSelectionEnd = selectionStart + (selectedText || "链接文字").length + 1;
  }

  emit("update:modelValue", `${props.modelValue.slice(0, selectionStart)}${replacementText}${props.modelValue.slice(selectionEnd)}`);
  await nextTick();
  currentTextareaElement.focus();
  currentTextareaElement.selectionStart = nextSelectionStart;
  currentTextareaElement.selectionEnd = nextSelectionEnd;
}
</script>

<template>
  <div class="markdown-workspace">
    <section class="editor-surface markdown-surface">
      <div class="surface-header">
        <div>
          <h3>Markdown</h3>
          <p>支持 Ctrl+B / Ctrl+I / Ctrl+K 和 Tab 缩进</p>
        </div>
      </div>
      <textarea ref="textareaElement" class="markdown-textarea" :value="modelValue" spellcheck="false"
        @input="handleInput" @keydown="handleKeydown"></textarea>
    </section>

    <section class="editor-surface preview-surface">
      <div class="surface-header">
        <div>
          <h3>预览</h3>
          <p>{{ isPreviewLoading ? "正在同步" : "已同步" }}</p>
        </div>
      </div>
      <article class="preview-body" v-html="previewHtml"></article>
    </section>
  </div>
</template>
