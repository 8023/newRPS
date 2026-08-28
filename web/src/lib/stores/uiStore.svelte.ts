// 纯客户端 UI 状态：免责声明确认、主题、全局弹窗开关、toast 触发文案、随机任务标签
// 偏好。不依赖 sessionStore/routerStore——这一层只管"页面上现在展示什么"，不参与任何
// 业务判断，任何组件都可以放心 import 而不必担心引入循环依赖。
import { securityDisclaimerKey } from "../constants";
import { todayKey } from "../rpc";
import { readPunishmentTagPrefs, writePunishmentTagPrefs } from "../session";
import { trackThemeToggle } from "../analytics";

class UiStore {
  // 每天每个浏览器只需确认一次；未过期就跳过声明页。
  disclaimerConfirmed = $state(localStorage.getItem(securityDisclaimerKey) === todayKey());
  theme = $state<"light" | "dark">(localStorage.getItem("rps-online-theme") === "dark" ? "dark" : "light");

  /** toast 触发文案：写入即弹出，Toast.svelte 自行管理进出场动画的定时器。 */
  notice = $state("");

  profileOpen = $state(false);
  leaderboardOpen = $state(false);
  aboutOpen = $state(false);
  helpOpen = $state(false);

  // 随机任务开房标签偏好：纯本地浏览器存储，不随 player:join 从服务端下发、不跨设备同步。
  punishmentTagPrefs = $state<Record<string, string>>(readPunishmentTagPrefs());

  notify(message: string) {
    this.notice = message;
  }

  confirmDisclaimer() {
    localStorage.setItem(securityDisclaimerKey, todayKey());
    this.disclaimerConfirmed = true;
  }

  /** 管理员把声明总开关关掉时，即使今天还没确认过也直接放行，不强行卡住。 */
  releaseDisclaimerIfDisabled() {
    this.disclaimerConfirmed = true;
  }

  setTheme(next: "light" | "dark") {
    this.theme = next;
    localStorage.setItem("rps-online-theme", next);
    trackThemeToggle(next);
  }

  setPunishmentTagPrefs(prefs: Record<string, string>) {
    this.punishmentTagPrefs = prefs;
    writePunishmentTagPrefs(prefs);
  }
}

export const uiStore = new UiStore();
