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
    const response = await updateSettings(settings);
    setSettings(response.settings);
    setMessage("配置已保存。");
  }

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
    themeDefault: "auto",
    font: "default",
  };
}
