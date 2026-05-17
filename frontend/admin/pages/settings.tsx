import Head from "next/head";
import { Save } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import { fetchSettings, updateSettings } from "@/api/posts";
import { AdminLayout } from "@/components/admin-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useAdminSession } from "@/lib/use-admin-session";
import type { SiteSettings } from "@/types/posts";

export default function SettingsPage() {
  const isReady = useAdminSession();
  const [settings, setSettings] = useState<SiteSettings>(emptySettings());
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!isReady) return;
    void fetchSettings().then(setSettings);
  }, [isReady]);

  async function save() {
    try {
      const response = await updateSettings(settings);
      setSettings(response.settings);
      notifyAdminFaviconChange(response.settings.iconUrl);
      setMessage("配置已保存。");
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "请求失败。"}`);
    }
  }

  function setPermalinkStructure(structure: string) {
    setSettings({ ...settings, permalinkStructure: structure });
  }

  function appendPermalinkTag(tag: string) {
    const currentStructure = isCustomPermalink(settings.permalinkStructure)
      ? settings.permalinkStructure
      : "/%post_id%.html";
    setPermalinkStructure(currentStructure + tag);
  }

  const selectedPermalink = permalinkOptions.find((option) => option.structure === settings.permalinkStructure);
  const isCustomSelected = !selectedPermalink;
  const customStructure = isCustomSelected ? settings.permalinkStructure : "/%post_id%.html";

  return (
    <>
      <Head>
        <title>系统配置 - HonePress</title>
      </Head>
      <AdminLayout title="系统配置" description="站点、评论、主题与管理员账号配置。">
        <div className="grid gap-6">
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <div>
                <CardTitle>基础信息</CardTitle>
                <CardDescription>{message || "修改后会同步刷新静态产物。"}</CardDescription>
              </div>
              <Button onClick={() => void save()}>
                <Save className="h-4 w-4" />
                保存
              </Button>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <Field label="站点标题">
                <Input
                  value={settings.title}
                  onChange={(event) => setSettings({ ...settings, title: event.target.value })}
                />
              </Field>
              <Field label="站点描述">
                <Input
                  value={settings.description}
                  onChange={(event) => setSettings({ ...settings, description: event.target.value })}
                />
              </Field>
              <Field label="站点 Icon">
                <Input
                  value={settings.iconUrl}
                  onChange={(event) => setSettings({ ...settings, iconUrl: event.target.value })}
                />
              </Field>
              <Field label="默认主题">
                <Select
                  value={settings.themeDefault}
                  onChange={(event) =>
                    setSettings({ ...settings, themeDefault: event.target.value as SiteSettings["themeDefault"] })
                  }
                >
                  <option value="auto">跟随系统</option>
                  <option value="light">浅色</option>
                  <option value="dark">深色</option>
                </Select>
              </Field>
              <Field label="字体">
                <Select
                  value={settings.font}
                  onChange={(event) => setSettings({ ...settings, font: event.target.value as SiteSettings["font"] })}
                >
                  <option value="default">默认字体</option>
                  <option value="douyin-sans">抖音美好体</option>
                </Select>
              </Field>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>固定链接结构</CardTitle>
              <CardDescription>全站文章链接会按这里的结构重新生成。</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-5">
              {permalinkOptions.map((option) => (
                <label className="grid gap-2 text-sm" key={option.structure}>
                  <span className="flex items-center gap-2 font-medium">
                    <input
                      type="radio"
                      checked={settings.permalinkStructure === option.structure}
                      onChange={() => setPermalinkStructure(option.structure)}
                    />
                    {option.label}
                  </span>
                  <code className="max-w-full overflow-x-auto whitespace-nowrap rounded bg-muted px-2 py-1 text-xs font-normal text-muted-foreground">
                    {option.example}
                  </code>
                </label>
              ))}
              <label className="grid gap-2 text-sm">
                <span className="flex items-center gap-2 font-medium">
                  <input
                    type="radio"
                    checked={isCustomSelected}
                    onChange={() => setPermalinkStructure(customStructure)}
                  />
                  自定义结构
                </span>
                <div className="flex flex-col gap-2 md:flex-row md:items-center">
                  <Input
                    className="min-w-0 md:flex-1"
                    value={customStructure}
                    onChange={(event) => setPermalinkStructure(event.target.value)}
                    onFocus={() => setPermalinkStructure(customStructure)}
                    placeholder="/%post_id%.html"
                  />
                </div>
              </label>
              <div className="grid gap-2">
                <span className="text-sm text-muted-foreground">可用标签：</span>
                <div className="flex flex-wrap gap-2">
                  {permalinkTags.map((tag) => (
                    <button
                      type="button"
                      className="rounded-md border border-input px-2.5 py-1 text-xs text-muted-foreground hover:bg-muted"
                      key={tag}
                      onClick={() => appendPermalinkTag(tag)}
                    >
                      {tag}
                    </button>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>评论配置</CardTitle>
              <CardDescription>giscus 参数保持为空时不会渲染评论。</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <label className="flex items-center gap-2 text-sm">
                <input
                  checked={settings.commentEnabled}
                  onChange={(event) => setSettings({ ...settings, commentEnabled: event.target.checked })}
                  type="checkbox"
                />
                开启评论
              </label>
              <div />
              {settings.commentEnabled ? (
                <>
                  <Field label="GitHub 仓库">
                    <Input
                      value={settings.giscusRepo}
                      onChange={(event) => setSettings({ ...settings, giscusRepo: event.target.value })}
                    />
                  </Field>
                  <Field label="仓库 ID">
                    <Input
                      value={settings.giscusRepoId}
                      onChange={(event) => setSettings({ ...settings, giscusRepoId: event.target.value })}
                    />
                  </Field>
                  <Field label="分类">
                    <Input
                      value={settings.giscusCategory}
                      onChange={(event) => setSettings({ ...settings, giscusCategory: event.target.value })}
                    />
                  </Field>
                  <Field label="分类 ID">
                    <Input
                      value={settings.giscusCategoryId}
                      onChange={(event) => setSettings({ ...settings, giscusCategoryId: event.target.value })}
                    />
                  </Field>
                </>
              ) : null}
            </CardContent>
          </Card>
        </div>
      </AdminLayout>
    </>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-2 text-sm font-medium">
      {label}
      {children}
    </label>
  );
}

function emptySettings(): SiteSettings {
  return {
    title: "",
    description: "",
    iconUrl: "",
    adminUsername: "",
    adminPassword: "",
    commentEnabled: false,
    giscusRepo: "",
    giscusRepoId: "",
    giscusCategory: "",
    giscusCategoryId: "",
    permalinkStructure: "/?p=%post_id%",
    themeDefault: "auto",
    font: "default",
  };
}

function notifyAdminFaviconChange(iconUrl: string) {
  window.dispatchEvent(new CustomEvent("honepress-admin-favicon-change", { detail: iconUrl }));
}

const permalinkOptions = [
  { label: "朴素", structure: "/?p=%post_id%", example: "/?p=123" },
  { label: "日期和名称型", structure: "/%year%/%monthnum%/%day%/%postname%/", example: "/2026/05/16/sample-post/" },
  { label: "月份和名称型", structure: "/%year%/%monthnum%/%postname%/", example: "/2026/05/sample-post/" },
  { label: "数字型", structure: "/archives/%post_id%", example: "/archives/123" },
  { label: "文章名", structure: "/%postname%/", example: "/sample-post/" },
];

const permalinkTags = [
  "%year%",
  "%monthnum%",
  "%day%",
  "%hour%",
  "%minute%",
  "%second%",
  "%post_id%",
  "%postname%",
  "%category%",
];

function isCustomPermalink(structure: string): boolean {
  return !permalinkOptions.some((option) => option.structure === structure);
}
