import Head from "next/head";
import { useRouter } from "next/router";
import { Eye, Pencil, Plus, Save, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { createPost, deletePost, fetchPost, fetchPosts, previewMarkdown, updatePost } from "@/api/posts";
import { AdminLayout } from "@/components/admin-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useAdminSession } from "@/lib/use-admin-session";
import type { AdminPostsResponse, PostDetail, SavePostRequest } from "@/types/posts";

type DraftFilter = "all" | "true" | "false";

export default function PostsPage() {
  const router = useRouter();
  const isReady = useAdminSession();
  const [postsResponse, setPostsResponse] = useState<AdminPostsResponse>({
    posts: [],
    page: 1,
    pageSize: 10,
    total: 0,
    totalPages: 1,
  });
  const [search, setSearch] = useState("");
  const [draft, setDraft] = useState<DraftFilter>("all");
  const [page, setPage] = useState(1);
  const [editor, setEditor] = useState<PostDetail>(createEmptyPost());
  const [isEditing, setIsEditing] = useState(false);
  const [preview, setPreview] = useState("");
  const [message, setMessage] = useState("");

  const tagsText = useMemo(() => editor.tags.join(", "), [editor.tags]);
  const aliasesText = useMemo(() => editor.aliases.join("\n"), [editor.aliases]);

  useEffect(() => {
    if (!isReady) return;
    void loadPosts();
  }, [isReady, page, draft]);

  useEffect(() => {
    if (!isReady || typeof router.query.edit !== "string") return;
    void openEditor(router.query.edit);
  }, [isReady, router.query.edit]);

  useEffect(() => {
    if (!isEditing) return;
    const timer = window.setTimeout(async () => {
      setPreview(await previewMarkdown(editor.body).catch((error) => `<p>${readError(error)}</p>`));
    }, 250);
    return () => window.clearTimeout(timer);
  }, [editor.body, isEditing]);

  async function loadPosts(nextPage = page) {
    const loadedPosts = await fetchPosts({ page: nextPage, pageSize: 10, search, draft });
    setPostsResponse(loadedPosts);
  }

  async function openEditor(postID: string) {
    const post = await fetchPost(postID);
    setEditor(normalizePost(post));
    setIsEditing(true);
    setMessage("");
  }

  function startCreate() {
    setEditor(createEmptyPost());
    setPreview("");
    setIsEditing(true);
    setMessage("");
  }

  async function saveEditor() {
    const request = buildSaveRequest(editor);
    const response = editor.id ? await updatePost(editor.id, request) : await createPost(request);
    setEditor(normalizePost(response.post));
    setMessage("文章已保存。");
    await loadPosts(1);
    setPage(1);
  }

  async function removeEditor() {
    if (!editor.id) return;
    if (!window.confirm(`确定删除「${editor.title}」吗？`)) return;
    await deletePost(editor.id);
    setIsEditing(false);
    setEditor(createEmptyPost());
    setMessage("文章已删除。");
    await loadPosts(1);
    setPage(1);
  }

  function setField<Key extends keyof PostDetail>(key: Key, value: PostDetail[Key]) {
    setEditor((current) => ({ ...current, [key]: value }));
  }

  return (
    <>
      <Head>
        <title>文章管理 - HonePress</title>
      </Head>
      <AdminLayout title="文章管理" description="后端分页、搜索、筛选和 Markdown 实时预览。">
        <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_520px]">
          <Card>
            <CardHeader className="gap-4">
              <div className="flex flex-col justify-between gap-3 md:flex-row md:items-center">
                <div>
                  <CardTitle>文章列表</CardTitle>
                  <CardDescription>
                    共 {postsResponse.total} 篇，当前第 {postsResponse.page} 页。
                  </CardDescription>
                </div>
                <Button onClick={startCreate}>
                  <Plus className="h-4 w-4" />
                  新建
                </Button>
              </div>
              <div className="grid gap-3 md:grid-cols-[1fr_160px_auto]">
                <Input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder="搜索标题、摘要、标签"
                />
                <Select value={draft} onChange={(event) => setDraft(event.target.value as DraftFilter)}>
                  <option value="all">全部状态</option>
                  <option value="false">已发布</option>
                  <option value="true">草稿</option>
                </Select>
                <Button variant="outline" onClick={() => void loadPosts(1)}>
                  搜索
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="overflow-x-auto">
                <table className="w-full min-w-[720px] text-sm">
                  <thead>
                    <tr className="border-b text-left text-muted-foreground">
                      <th className="py-2 pr-4 font-medium">标题</th>
                      <th className="py-2 pr-4 font-medium">状态</th>
                      <th className="py-2 pr-4 font-medium">时间</th>
                      <th className="py-2 pr-4 font-medium">链接</th>
                      <th className="py-2 text-right font-medium">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {postsResponse.posts.map((post) => (
                      <tr className="border-b" key={post.id}>
                        <td className="max-w-[280px] py-3 pr-4">
                          <strong className="block truncate">{post.title}</strong>
                          <span className="block truncate text-muted-foreground">{post.description || "无摘要"}</span>
                        </td>
                        <td className="py-3 pr-4">
                          <span className="rounded-full border px-2 py-0.5 text-xs">
                            {post.draft ? "草稿" : "已发布"}
                          </span>
                        </td>
                        <td className="py-3 pr-4 text-muted-foreground">{post.date}</td>
                        <td className="py-3 pr-4 font-mono text-xs">{post.url}</td>
                        <td className="py-3 text-right">
                          <Button variant="ghost" size="icon" onClick={() => void openEditor(post.id)} title="编辑">
                            <Pencil className="h-4 w-4" />
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="mt-4 flex items-center justify-between">
                <Button
                  variant="outline"
                  disabled={postsResponse.page <= 1}
                  onClick={() => setPage((current) => Math.max(1, current - 1))}
                >
                  上一页
                </Button>
                <span className="text-sm text-muted-foreground">
                  {postsResponse.page} / {postsResponse.totalPages}
                </span>
                <Button
                  variant="outline"
                  disabled={postsResponse.page >= postsResponse.totalPages}
                  onClick={() => setPage((current) => current + 1)}
                >
                  下一页
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{isEditing ? (editor.id ? "编辑文章" : "新建文章") : "编辑器"}</CardTitle>
              <CardDescription>{message || "选择文章或新建文章后开始编辑。"}</CardDescription>
            </CardHeader>
            <CardContent>
              {isEditing ? (
                <div className="grid gap-4">
                  <Input
                    value={editor.title}
                    onChange={(event) => setField("title", event.target.value)}
                    placeholder="标题"
                  />
                  <div className="grid gap-3 md:grid-cols-2">
                    <Input
                      value={editor.date}
                      onChange={(event) => setField("date", event.target.value)}
                      placeholder="YYYY-MM-DD HH:mm:ss"
                    />
                    <Input
                      value={editor.url}
                      onChange={(event) => setField("url", event.target.value)}
                      placeholder="example.html"
                    />
                  </div>
                  <Textarea
                    value={editor.description}
                    onChange={(event) => setField("description", event.target.value)}
                    placeholder="摘要"
                  />
                  <Input
                    value={editor.thumbnail}
                    onChange={(event) => setField("thumbnail", event.target.value)}
                    placeholder="缩略图 URL"
                  />
                  <Input
                    value={editor.icon}
                    onChange={(event) => setField("icon", event.target.value)}
                    placeholder="Emoji / favicon override"
                  />
                  <Input
                    value={tagsText}
                    onChange={(event) =>
                      setField(
                        "tags",
                        event.target.value
                          .split(",")
                          .map((tag) => tag.trim())
                          .filter(Boolean),
                      )
                    }
                    placeholder="标签，用英文逗号分隔"
                  />
                  <Textarea
                    value={aliasesText}
                    onChange={(event) =>
                      setField(
                        "aliases",
                        event.target.value
                          .split("\n")
                          .map((alias) => alias.trim())
                          .filter(Boolean),
                      )
                    }
                    placeholder="别名链接，每行一个"
                  />
                  <div className="grid gap-3 md:grid-cols-2">
                    <Input
                      value={editor.seoTitle}
                      onChange={(event) => setField("seoTitle", event.target.value)}
                      placeholder="SEO 标题"
                    />
                    <Textarea
                      value={editor.seoDescription}
                      onChange={(event) => setField("seoDescription", event.target.value)}
                      placeholder="SEO 描述"
                    />
                  </div>
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      checked={editor.draft}
                      onChange={(event) => setField("draft", event.target.checked)}
                      type="checkbox"
                    />
                    保存为草稿
                  </label>
                  <div className="grid gap-4 lg:grid-cols-2">
                    <Textarea
                      className="min-h-[360px] font-mono"
                      value={editor.body}
                      onChange={(event) => setField("body", event.target.value)}
                      placeholder="Markdown 正文"
                    />
                    <div
                      className="prose-preview min-h-[360px] overflow-auto rounded-md border p-4"
                      dangerouslySetInnerHTML={{ __html: preview }}
                    />
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <Button onClick={() => void saveEditor()}>
                      <Save className="h-4 w-4" />
                      保存
                    </Button>
                    {editor.id && !editor.draft ? (
                      <a
                        className="inline-flex h-10 items-center gap-2 rounded-md border px-4 text-sm"
                        href={`/${editor.url}`}
                        target="_blank"
                        rel="noreferrer"
                      >
                        <Eye className="h-4 w-4" />
                        查看
                      </a>
                    ) : null}
                    {editor.id ? (
                      <Button variant="destructive" onClick={() => void removeEditor()}>
                        <Trash2 className="h-4 w-4" />
                        删除
                      </Button>
                    ) : null}
                  </div>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">暂无打开的文章。</p>
              )}
            </CardContent>
          </Card>
        </div>
      </AdminLayout>
    </>
  );
}

function createEmptyPost(): PostDetail {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, "0");
  return {
    id: "",
    title: "未命名文章",
    icon: "",
    thumbnail: "",
    date: `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`,
    description: "",
    seoTitle: "",
    seoDescription: "",
    draft: false,
    url: "new-post.html",
    aliases: [],
    tags: [],
    body: "",
  };
}

function normalizePost(post: PostDetail): PostDetail {
  return {
    ...post,
    icon: post.icon ?? "",
    thumbnail: post.thumbnail ?? "",
    aliases: post.aliases ?? [],
    tags: post.tags ?? [],
  };
}

function buildSaveRequest(post: PostDetail): SavePostRequest {
  return {
    id: post.id,
    title: post.title,
    icon: post.icon,
    thumbnail: post.thumbnail,
    date: post.date,
    description: post.description,
    seoTitle: post.seoTitle,
    seoDescription: post.seoDescription,
    draft: post.draft,
    url: post.url,
    aliases: post.aliases,
    tags: post.tags,
    body: post.body,
  };
}

function readError(error: unknown): string {
  return error instanceof Error ? error.message : "请求失败";
}
