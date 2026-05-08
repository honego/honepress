import Link from "next/link";
import { useRouter } from "next/router";
import { FileText, LayoutDashboard, LogOut, Settings, Users } from "lucide-react";
import { ReactNode } from "react";

import { logoutAdmin } from "@/api/posts";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const navItems = [
  { href: "/dashboard", label: "仪表盘", icon: LayoutDashboard },
  { href: "/posts", label: "文章", icon: FileText },
  { href: "/settings", label: "设置", icon: Settings },
  { href: "/users", label: "用户", icon: Users },
];

export function AdminLayout({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  const router = useRouter();

  async function logout() {
    await logoutAdmin().catch(() => undefined);
    window.location.href = "/login";
  }

  return (
    <div className="min-h-screen bg-muted/30">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r bg-background p-4 md:block">
        <Link href="/dashboard" className="mb-8 flex items-center gap-2 font-semibold">
          <img src="/honepress-black.svg" alt="" className="h-7 w-7 rounded-full" />
          HonePress
        </Link>
        <nav className="grid gap-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const active = router.pathname === item.href;
            return (
              <Link
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                  active && "bg-muted text-foreground",
                )}
                href={item.href}
                key={item.href}
              >
                <Icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
      </aside>
      <div className="md:pl-64">
        <header className="sticky top-0 z-10 border-b bg-background/95 px-4 py-3 backdrop-blur md:px-8">
          <div className="flex items-center justify-between gap-4">
            <div>
              <h1 className="text-xl font-semibold tracking-normal">{title}</h1>
              {description ? <p className="text-sm text-muted-foreground">{description}</p> : null}
            </div>
            <Button variant="outline" size="sm" onClick={logout}>
              <LogOut className="h-4 w-4" />
              退出
            </Button>
          </div>
          <nav className="mt-3 flex gap-2 overflow-x-auto md:hidden">
            {navItems.map((item) => (
              <Link className="rounded-md border px-3 py-1.5 text-sm" href={item.href} key={item.href}>
                {item.label}
              </Link>
            ))}
          </nav>
        </header>
        <main className="p-4 md:p-8">{children}</main>
      </div>
    </div>
  );
}
