<script lang="ts">
  // 源：ui/AppViews.tsx:79-203。改为直接读写 sessionStore/uiStore，不再需要
  // config/onDone/onError 三个 props——App.svelte 不必知道登录成功之后具体发生了什么。
  import { untrack } from "svelte";
  import type { MeState } from "../../lib/types";
  import { ask } from "../../lib/rpc";
  import { tokenKey } from "../../lib/constants";
  import { clearPlayerIdentity, claimIdentity, joinIdentityPayload } from "../../lib/session";
  import { firstFactionId, firstGenderId, nextGenderIdForFaction, genderChoiceError } from "../../lib/playerDisplay";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import FactionSelect from "./FactionSelect.svelte";
  import GenderSelect from "./GenderSelect.svelte";

  const config = $derived(sessionStore.config!);

  let name = $state("");
  // 只取一次初始值（与原 React useState(() => ...) 惰性初始化同构）：config 后续变化
  // 不应该覆盖用户已经选择的阵营/性别，untrack 显式声明"仅初始化时读取"。
  let factionId = $state(untrack(() => firstFactionId(config)));
  let genderId = $state(untrack(() => firstGenderId(config, firstFactionId(config))));
  let mode = $state<"new" | "restore">("new");
  let restoreCode = $state("");
  let restoreBusy = $state(false);
  let pendingJoinPayload = $state<Record<string, unknown> | null>(null);

  async function doJoin(payload: Record<string, unknown>) {
    const result = await ask<MeState & { alreadyOnline?: true }>("player:join", payload);
    if (result.alreadyOnline) {
      pendingJoinPayload = payload;
      return;
    }
    sessionStore.completeManualLogin(result);
  }

  async function submit() {
    const genderError = genderChoiceError(config, genderId);
    if (genderError) {
      uiStore.notify(genderError);
      return;
    }
    try {
      await doJoin({ name, genderId, token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload()) });
    } catch (error) {
      const message = error instanceof Error ? error.message : "进入失败";
      if (message === "玩家身份校验失败") {
        // 本地缓存的 playerId/playerSecret 服务端已经不认（常见于老账号未完成迁移）：
        // 清掉本地身份重试一次，等价于自动执行"清 localStorage 重新注册"。
        clearPlayerIdentity();
        try {
          await doJoin({ name, genderId, token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload()) });
          return;
        } catch (retryError) {
          uiStore.notify(retryError instanceof Error ? retryError.message : "进入失败");
          return;
        }
      }
      uiStore.notify(message);
    }
  }

  async function submitRestore() {
    if (!restoreCode.trim()) {
      uiStore.notify("请输入认领密钥");
      return;
    }
    restoreBusy = true;
    try {
      const claimed = await claimIdentity(restoreCode);
      await doJoin({
        name: claimed.name, genderId: claimed.genderId,
        token: localStorage.getItem(tokenKey), ...(await joinIdentityPayload())
      });
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "认领失败");
    } finally {
      restoreBusy = false;
    }
  }

  async function confirmKick() {
    if (!pendingJoinPayload) return;
    try {
      await doJoin({ ...pendingJoinPayload, forceKick: true });
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "进入失败");
    } finally {
      pendingJoinPayload = null;
    }
  }

  function onFactionChange(nextFactionId: string) {
    factionId = nextFactionId;
    genderId = nextGenderIdForFaction(config, nextFactionId, genderId);
  }
</script>

{#if pendingJoinPayload}
  <section class="login-card kick-confirm-card">
    <h2>该账号已在其他设备登录</h2>
    <p class="hint">继续登录会把另一台设备顶下线（那边会收到提示）。确定要继续吗？</p>
    <div class="kick-confirm-actions">
      <button onclick={() => (pendingJoinPayload = null)}>取消</button>
      <button class="primary" onclick={confirmKick}>确定顶替登录</button>
    </div>
  </section>
{:else}
  <section class="login-card">
    <h2>进入游戏</h2>
    {#if mode === "new"}
      <input value={name} oninput={(event) => (name = event.currentTarget.value)} maxlength="12" placeholder="你的名字，允许重复" />
      <FactionSelect {config} {factionId} {onFactionChange} />
      <GenderSelect {config} {genderId} {factionId} onGenderChange={(next) => (genderId = next)} />
      <button class="primary" onclick={submit}>进入大厅</button>
      <button class="link-button" onclick={() => (mode = "restore")}>已有账号？用认领密钥恢复</button>
    {:else}
      <p class="hint">在另一台设备的「个人设置」里获取认领密钥，粘贴到这里即可恢复该账号。</p>
      <input value={restoreCode} oninput={(event) => (restoreCode = event.currentTarget.value)} placeholder="粘贴认领密钥" />
      <button class="primary" disabled={restoreBusy} onclick={submitRestore}>{restoreBusy ? "恢复中…" : "恢复账号"}</button>
      <button class="link-button" onclick={() => (mode = "new")}>返回新建账号</button>
    {/if}
  </section>
{/if}
