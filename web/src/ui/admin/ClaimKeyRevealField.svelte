<script lang="ts">
  // 认领密钥显示框：默认显示占位文案；聚焦（点击）时签发一把新密钥（旧密钥立即作废，见后端
  // showClaimKey 的无条件轮换）并自动复制到剪贴板；失焦 5 秒后（期间重新聚焦会取消这次复原、
  // 不重新签发）恢复默认文案，再次聚焦视为全新一轮，会再签发一把新密钥。
  // 源：ui/AdminViews.tsx:1934-1995
  import { ask } from "../../lib/rpc";
  import { encodeClaimCode } from "../../lib/session";

  let { playerId, onError }: { playerId: string; onError: (message: string) => void } = $props();

  const DEFAULT_TEXT = "点击显示认领密钥";
  let value = $state(DEFAULT_TEXT);
  let revealed = $state(false);
  let revertTimer: number | null = null;

  $effect(() => {
    return () => {
      if (revertTimer != null) window.clearTimeout(revertTimer);
    };
  });

  async function reveal() {
    try {
      const result = await ask<{ claimKey: string; playerId: string }>("admin:action", {
        action: "showClaimKey",
        playerId
      });
      const code = encodeClaimCode(result.playerId, result.claimKey);
      value = code;
      revealed = true;
      try {
        await navigator.clipboard.writeText(code);
      } catch {
        onError("复制失败，请手动选中密钥");
      }
    } catch (error) {
      onError(error instanceof Error ? error.message : "获取认领密钥失败");
    }
  }

  function handleFocus() {
    if (revertTimer != null) {
      window.clearTimeout(revertTimer);
      revertTimer = null;
    }
    if (!revealed) void reveal();
  }

  function handleBlur() {
    revertTimer = window.setTimeout(() => {
      value = DEFAULT_TEXT;
      revealed = false;
      revertTimer = null;
    }, 5000);
  }
</script>

<label class="field-label">
  <span>认领密钥</span>
  <input class="admin-claim-key-input" readonly {value} spellcheck="false" autocomplete="off" onfocus={handleFocus} onblur={handleBlur} />
</label>
