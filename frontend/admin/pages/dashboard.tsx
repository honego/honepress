import Head from "next/head";
import Link from "next/link";
import { FileClock, FileText, Send } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import { fetchAdminStats, fetchPosts } from "@/api/posts";
import { AdminLayout } from "@/components/admin-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useAdminSession } from "@/lib/use-admin-session";
import type { AdminStats, PostSummary } from "@/types/posts";

export default function DashboardPage() {
  const isReady = useAdminSession();
  const [stats, setStats] = useState<AdminStats | null>(null);
  const [recentPosts, setRecentPosts] = useState<PostSummary[]>([]);

  useEffect(() => {
    if (!isReady) return;
    async function load() {
      const [loadedStats, loadedPosts] = await Promise.all([fetchAdminStats(), fetchPosts({ page: 1, pageSize: 5 })]);
      setStats(loadedStats);
      setRecentPosts(loadedPosts.posts);
    }
    void load();
  }, [isReady]);

  return (
    <>
      <Head>
        <title>仪表盘 - HonePress</title>
      </Head>
      <AdminLayout title="仪表盘" description="站点内容、草稿和最近文章概览。">
        <section className="grid gap-4 md:grid-cols-3">
          <MetricCard icon={<FileText className="h-5 w-5" />} label="全部文章" value={stats?.totalPosts ?? 0} />
          <MetricCard icon={<Send className="h-5 w-5" />} label="已发布" value={stats?.publishedPosts ?? 0} />
          <MetricCard icon={<FileClock className="h-5 w-5" />} label="草稿" value={stats?.draftPosts ?? 0} />
        </section>
        <Card className="mt-6">
          <CardHeader className="flex-row items-center justify-between gap-4">
            <div>
              <CardTitle>最近文章</CardTitle>
              <CardDescription>继续编辑或查看最新内容。</CardDescription>
            </div>
            <Link href="/posts">
              <Button variant="outline">管理文章</Button>
            </Link>
          </CardHeader>
          <CardContent>
            <div className="grid gap-2">
              {recentPosts.map((post) => (
                <Link
                  href={`/posts?edit=${encodeURIComponent(post.id)}`}
                  key={post.id}
                  className="flex items-center justify-between rounded-md border p-3 text-sm hover:bg-muted"
                >
                  <span>
                    <strong className="block">{post.title}</strong>
                    <span className="text-muted-foreground">{post.date}</span>
                  </span>
                  <span className="rounded-full border px-2 py-0.5 text-xs text-muted-foreground">
                    {post.draft ? "草稿" : "已发布"}
                  </span>
                </Link>
              ))}
              {recentPosts.length === 0 ? <p className="text-sm text-muted-foreground">还没有文章。</p> : null}
            </div>
          </CardContent>
        </Card>
      </AdminLayout>
    </>
  );
}

function MetricCard({ icon, label, value }: { icon: ReactNode; label: string; value: number }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardDescription>{label}</CardDescription>
          <span className="rounded-md bg-muted p-2">{icon}</span>
        </div>
      </CardHeader>
      <CardContent>
        <strong className="text-3xl">{value}</strong>
      </CardContent>
    </Card>
  );
}
