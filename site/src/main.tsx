// 入口：若服务端预渲染了 HTML（root 有子节点）则水合，否则直接挂载。
// 适用于 Vite + React 的预渲染部署（见 scripts/prerender.cjs）。
import { StrictMode } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import App from "./App";
import "./styles.css";

const rootElement = document.getElementById("root")!;
const app = (
  <StrictMode>
    <App initialPath={window.location.pathname} />
  </StrictMode>
);

if (rootElement.hasChildNodes()) {
  hydrateRoot(rootElement, app);
} else {
  createRoot(rootElement).render(app);
}