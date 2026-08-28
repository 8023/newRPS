<script lang="ts">
  // 认领密钥展示区：个人资料页 / 登出确认弹窗共用。
  // 源：ui/AppViews.tsx:206-263
  import { fetchClaimKey, refreshClaimKey, encodeClaimCode } from "../../lib/session";

  let { onError }: { onError: (message: string) => void } = $props();

  let code = $state<string | null>(null);
  let loading = $state(false);
  let copied = $state(false);

  async function load() {
    loading = true;
    try {
      const result = await fetchClaimKey();
      code = encodeClaimCode(result.playerId, result.claimKey);
    } catch (error) {
      onError(error instanceof Error ? error.message : "获取失败");
    } finally {
      loading = false;
    }
  }

  async function refresh() {
    loading = true;
    try {
      const result = await refreshClaimKey();
      code = encodeClaimCode(result.playerId, result.claimKey);
      copied = false;
    } catch (error) {
      onError(error instanceof Error ? error.message : "刷新失败");
    } finally {
      loading = false;
    }
  }

  async function copy() {
    if (!code) return;
    try {
      await navigator.clipboard.writeText(code);
      copied = true;
      window.setTimeout(() => { copied = false; }, 2000);
    } catch {
      onError("请手动选中复制");
    }
  }
</script>

<div class="claim-key-panel">
  <p class="hint">本网站没有传统用户名/密码认证体系，认领密钥作为你在本站的唯一标识，请务必妥善保存、不要告知他人。认领密钥用于在另一台设备登录同一个账号，或在清空浏览器缓存或登出后恢复该账号信息。认领密钥只能存在一个（获取新认领密钥后，旧密钥立即作废）且仅可使用一次（使用认领密钥恢复账号后作废，但可以申请新的认领密钥）。</p>
  {#if code}
    <code class="claim-key-code">{code}</code>
    <div class="claim-key-actions">
      <button type="button" onclick={copy}>{copied ? "已复制" : "复制密钥"}</button>
      <button type="button" disabled={loading} onclick={refresh}>刷新密钥</button>
    </div>
  {:else}
    <button type="button" disabled={loading} onclick={load}>{loading ? "获取中…" : "获取认领密钥"}</button>
  {/if}
</div>
