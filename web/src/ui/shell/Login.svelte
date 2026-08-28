<script lang="ts">
  // 源：ui/AppViews.tsx:79-203
  import type { AppConfig } from "../../shared/types";
  import type { MeState } from "../../lib/types";
  import { untrack } from "svelte";
  import { ask } from "../../lib/rpc";
  import { tokenKey, playerSecretKey } from "../../lib/constants";
  import { cacheJoinProfile, clearPlayerIdentity, claimIdentity, joinIdentityPayload } from "../../lib/session";
  import { firstFactionId, firstGenderId, nextGenderIdForFaction, genderChoiceError } from "../../lib/playerDisplay";
  import FactionSelect from "./FactionSelect.svelte";
  import GenderSelect from "./GenderSelect.svelte";

  let { config, onDone, onError }: { config: AppConfig; onDone: (me: MeState) => void; onError: (message: string) => void } = $props();

  // 只取一次初始值（与原 React useState(() => ...) 惰性初始化同构）：config 后续变化
  // 不应该覆盖用户已经选择的阵营/性别，untrack 显式声明"仅初始化时读取"。
  let name = $state("");
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
    localStorage.setItem(tokenKey, result.token);
    // 以服务端确认后的资料为准缓存。
    if (result.player) cacheJoinProfile(result.player);
    if (result.reissuedSecret) localStorage.setItem(playerSecretKey, result.reissuedSecret);
    onDone(result);
  }

  async function submit() {
    const genderError = genderChoiceError(config, genderId);
    if (genderError) {
      onError(genderError);
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
          onError(retryError instanceof Error ? retryError.message : "进入失败");
          return;
        }
      }
      onError(message);
    }
  }

  async function submitRestore() {
    if (!restoreCode.trim()) {
      onError("请输入认领密钥");
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
      onError(error instanceof Error ? error.message : "认领失败");
    } finally {
      restoreBusy = false;
    }
  }

  async function confirmKick() {
    if (!pendingJoinPayload) return;
    try {
      await doJoin({ ...pendingJoinPayload, forceKick: true });
    } catch (error) {
      onError(error instanceof Error ? error.message : "进入失败");
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
