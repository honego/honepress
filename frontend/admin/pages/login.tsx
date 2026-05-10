import Head from "next/head";
import { FormEvent, useEffect, useState } from "react";

import { checkAdminSession, loginAdmin } from "@/api/posts";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export default function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    void checkAdminSession()
      .then(() => {
        window.location.replace("/admin/dashboard");
      })
      .catch(() => undefined);
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    try {
      await loginAdmin(username.trim(), password);
      window.location.href = "/admin/dashboard";
    } catch (loginError) {
      setError(loginError instanceof Error ? loginError.message : "登录失败");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <>
      <Head>
        <title>登录 - HonePress</title>
      </Head>
      <main className="flex min-h-screen items-center justify-center bg-muted/30 px-4 py-10">
        <Card className="w-full max-w-sm">
          <CardHeader>
            <div className="mb-3 flex items-center gap-2 font-semibold">
              <img src="/honepress-black.svg" alt="" className="h-7 w-7 rounded-full" />
              HonePress
            </div>
            <CardTitle>后台登录</CardTitle>
            <CardDescription>输入后台账号后进入管理界面。</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="grid gap-4" onSubmit={handleSubmit}>
              <label className="grid gap-2 text-sm font-medium">
                用户名
                <Input
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  autoComplete="username"
                  placeholder="请输入用户名"
                />
              </label>
              <label className="grid gap-2 text-sm font-medium">
                密码
                <Input
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  autoComplete="current-password"
                  placeholder="请输入密码"
                  type="password"
                />
              </label>
              {error ? <p className="text-sm text-destructive">{error}</p> : null}
              <Button disabled={isSubmitting} type="submit">
                {isSubmitting ? "正在登录" : "登录"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </main>
    </>
  );
}
