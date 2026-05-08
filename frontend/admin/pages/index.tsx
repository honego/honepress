import { useEffect } from "react";

export default function AdminIndexPage() {
  useEffect(() => {
    window.location.replace("/admin/dashboard");
  }, []);

  return null;
}
