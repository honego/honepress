import Head from "next/head";
import { Save } from "lucide-react";
import { useEffect, useState } from "react";

import { fetchSettings, updateSettings } from "@/api/posts";
import { AdminLayout } from "@/components/admin-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useAdminSession } from "@/lib/use-admin-session";
import type { SiteSettings } from "@/types/posts";

export default function UsersPage() {
  const isReady = useAdminSession();
  const [settings, setSettings] = useState<SiteSettings | null>(null);
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!isReady) return;
    void fetchSettings().then(setSettings);
  }, [isReady]);

  async function save() {
    if (!settings) return;
    const response = await updateSettings(settings);
    setSettings(response.settings);
    setMessage("管理员账号已更新。");
  }

  return (
    <>
      <Head>
        <title>用户与权限 - HonePress</title>
      </Head>
      <AdminLayout title="用户与权限" description="当前版本内置单管理员角色，鉴权使用 HttpOnly JWT Cookie。">
        <Card>
          <CardHeader>
            <CardTitle>管理员</CardTitle>
            <CardDescription>{message || "设置用户名和密码后，后台 API 将要求登录。"}</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 md:max-w-xl">
            <label className="grid gap-2 text-sm font-medium">
              用户名
              <Input
                value={settings?.adminUsername ?? ""}
                onChange={(event) => settings && setSettings({ ...settings, adminUsername: event.target.value })}
                autoComplete="username"
              />
            </label>
            <label className="grid gap-2 text-sm font-medium">
              密码
              <Input
                value={settings?.adminPassword ?? ""}
                onChange={(event) => settings && setSettings({ ...settings, adminPassword: event.target.value })}
                autoComplete="new-password"
                type="password"
              />
            </label>
            <div className="rounded-md border bg-muted/40 p-3 text-sm text-muted-foreground">
              角色：<strong className="text-foreground">admin</strong>。JWT Payload 仅包含 user_id、role、exp。
            </div>
            <Button className="w-fit" onClick={() => void save()}>
              <Save className="h-4 w-4" />
              保存用户
            </Button>
          </CardContent>
        </Card>
      </AdminLayout>
    </>
  );
}
