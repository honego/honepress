import { useEffect } from "react";

export default function LoginRedirectPage() {
  useEffect(() => {
    window.location.replace("/admin/login");
  }, []);

  return null;
}
