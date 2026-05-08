import { useRouter } from "next/router";
import { useEffect, useState } from "react";

import { checkAdminSession, UnauthorizedError } from "@/api/posts";

export function useAdminSession() {
  const router = useRouter();
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    let mounted = true;
    async function verify() {
      try {
        await checkAdminSession();
        if (mounted) setIsReady(true);
      } catch (error) {
        if (error instanceof UnauthorizedError) {
          window.location.href = "/login";
          return;
        }
        if (mounted) setIsReady(true);
      }
    }
    void verify();
    return () => {
      mounted = false;
    };
  }, [router.pathname]);

  return isReady;
}
