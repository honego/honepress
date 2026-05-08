import Head from "next/head";
import { FormEvent, useState } from "react";

import { login } from "../src/lib/api";

export default function LoginPage() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setIsSubmitting(true);
    try {
      await login(username.trim(), password);
      window.location.href = "/";
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
      <main className="login-page">
        <section className="login-panel">
          <div className="brand-row">
            <img src="/honepress-black.svg" alt="" />
            <span>HonePress</span>
          </div>
          <h1>登录</h1>
          <p>使用后台账号进入站点与管理端。</p>
          <form onSubmit={handleSubmit} className="login-form">
            <label>
              用户名
              <input
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="username"
                type="text"
              />
            </label>
            <label>
              密码
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete="current-password"
                type="password"
              />
            </label>
            {error ? <p className="form-error">{error}</p> : null}
            <button disabled={isSubmitting} type="submit">
              {isSubmitting ? "正在登录" : "登录"}
            </button>
          </form>
        </section>
      </main>
    </>
  );
}
