// 纯导航状态：当前视图 + 后台入口的 hash/快捷键路由。不 import 任何其他 store——
// "从后台返回该去哪" 这类需要业务数据（是否登录/是否在房间）的判断，由调用方
// （App.svelte 组合层）把结论以布尔参数喂给 leaveAdmin()，路由层本身不查会话状态，
// 避免 sessionStore ⇄ routerStore 出现循环依赖。
import { isAdminRoute } from "../rpc";
import { trackPageview } from "../analytics";

export type AppView = "login" | "lobby" | "room" | "admin" | "contribute";

class RouterStore {
  view = $state<AppView>(isAdminRoute() ? "admin" : "login");
  keepContribute = $state(false);
  #viewBeforeAdmin: AppView = isAdminRoute() ? "login" : "lobby";

  /** 唯一的视图切换入口：顺带打点 pageview，等价于原 React 版"监听 view 变化" 的 effect。 */
  goto(next: AppView) {
    this.view = next;
    trackPageview(next);
  }

  openAdmin() {
    if (this.view !== "admin") {
      this.#viewBeforeAdmin = this.view;
      if (this.view === "contribute") this.keepContribute = true;
    }
    if (!isAdminRoute()) window.location.hash = "admin";
    this.goto("admin");
  }

  /** hasMe/hasRoom 由调用方按 sessionStore 当前值传入（见上方文件头注释）。 */
  leaveAdmin(hasMe: boolean, hasRoom: boolean) {
    if (window.location.hash === "#admin") window.location.hash = "";
    const prev = this.#viewBeforeAdmin;
    if (prev === "contribute" && hasMe) {
      this.goto("contribute");
      return;
    }
    if (prev === "room" && hasMe && hasRoom) {
      this.goto("room");
      return;
    }
    this.keepContribute = false;
    this.goto(hasMe ? "lobby" : "login");
  }

  /** 管理入口故意不放在普通页面按钮里：地址加 #admin，或按 Ctrl/Command + Shift + A。
      返回清理函数，供 App.svelte 用 $effect 包一层挂载一次。 */
  wireHashAndKeyboard(): () => void {
    const openHiddenAdmin = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.shiftKey && event.key.toLowerCase() === "a") {
        this.openAdmin();
      }
    };
    const openFromHash = () => {
      if (isAdminRoute()) this.openAdmin();
    };
    if (isAdminRoute()) this.openAdmin();
    window.addEventListener("keydown", openHiddenAdmin);
    window.addEventListener("hashchange", openFromHash);
    return () => {
      window.removeEventListener("keydown", openHiddenAdmin);
      window.removeEventListener("hashchange", openFromHash);
    };
  }
}

export const routerStore = new RouterStore();
