<script lang="ts">
  // 源：ui/AdminViews.tsx:1738-1929
  import { untrack } from "svelte";
  import type { AppConfig, PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { genderChoiceError, nextGenderIdForFaction, safePlayerStats, formatGiveawayValue } from "../../lib/playerDisplay";
  import FactionSelect from "../shell/FactionSelect.svelte";
  import GenderSelect from "../shell/GenderSelect.svelte";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";
  import ClaimKeyRevealField from "./ClaimKeyRevealField.svelte";

  let { config, player, onSave, onKick, onError }: {
    config: AppConfig;
    player: PublicPlayer;
    onSave: (payload: Record<string, unknown>) => void;
    onKick: () => void;
    onError: (message: string) => void;
  } = $props();

  type FocusField = "name" | "rankedPoints" | "title" | "giveaway" | "gender" | null;
  let focusedField: FocusField = null;

  let name = $state(untrack(() => player.name));
  let rankedPoints = $state(untrack(() => String(safePlayerStats(player).sortRankedPoints)));
  let rankedPointsTouched = $state(false);
  let title = $state(untrack(() => safePlayerStats(player).title));
  let titleTouched = $state(false);
  let giveawayInput = $state(untrack(() => formatGiveawayValue(player.giveawayEnabled ? player.giveawayValue || 0 : 0)));
  let giveawayTouched = $state(false);
  let genderId = $state(untrack(() => player.genderId));
  let factionId = $state(untrack(() => player.factionId));
  let genderTouched = $state(false);
  const titleCustom = $derived(!!safePlayerStats(player).titleCustom);

  type AltAccountsState = { loading: boolean; queried: boolean; error: string | null; players: PublicPlayer[]; truncated: boolean };
  const ALT_ACCOUNTS_IDLE: AltAccountsState = { loading: false, queried: false, error: null, players: [], truncated: false };
  let altAccounts = $state<AltAccountsState>(ALT_ACCOUNTS_IDLE);

  $effect(() => {
    const stats = safePlayerStats(player);
    if (focusedField !== "name") name = player.name;
    if (focusedField !== "rankedPoints") {
      rankedPoints = String(stats.sortRankedPoints);
      rankedPointsTouched = false;
    }
    if (focusedField !== "title") {
      title = stats.title;
      titleTouched = false;
    }
    if (focusedField !== "giveaway") {
      giveawayInput = formatGiveawayValue(player.giveawayEnabled ? player.giveawayValue || 0 : 0);
      giveawayTouched = false;
    }
    if (focusedField !== "gender") {
      genderId = player.genderId;
      factionId = player.factionId;
      genderTouched = false;
    }
  });

  $effect(() => {
    void player.id;
    altAccounts = ALT_ACCOUNTS_IDLE;
  });

  async function queryAltAccounts() {
    altAccounts = { ...altAccounts, loading: true, error: null };
    try {
      const result = await ask<{ players?: PublicPlayer[]; truncated?: boolean }>("admin:action", {
        action: "altAccounts",
        playerId: player.id
      });
      altAccounts = { loading: false, queried: true, error: null, players: result.players || [], truncated: !!result.truncated };
    } catch (error) {
      altAccounts = {
        loading: false, queried: true,
        error: error instanceof Error ? error.message : "查询失败",
        players: [], truncated: false
      };
    }
  }

  function save() {
    if (genderTouched) {
      const genderError = genderChoiceError(config, genderId);
      if (genderError) {
        onError(genderError);
        return;
      }
    }
    onSave({
      playerId: player.id,
      name,
      ...(rankedPointsTouched ? { rankedPoints: Number(rankedPoints) } : {}),
      ...(titleTouched ? { title } : {}),
      ...(giveawayTouched ? { giveawayValueInput: giveawayInput } : {}),
      ...(genderTouched ? { genderId } : {})
    });
  }
</script>

<div class="admin-player-editor">
  <div class="admin-player-head">
    <PlayerAvatar {player} size={28} />
    <PlayerBadge {player} compact />
    {#if player.nameWarEnabled}
      <span class="mode-chip">
        {player.nameWarPunished ? `名字争夺战中：${player.nameWarPenaltyName || "惩罚名生效"}` : "名字争夺战已开启"}
      </span>
    {/if}
  </div>
  <div class="admin-player-row1">
    <label class="field-label">
      <span>名字</span>
      <input
        value={name}
        maxlength="12"
        onfocus={() => { focusedField = "name"; }}
        onblur={() => { if (focusedField === "name") focusedField = null; }}
        oninput={(event) => (name = event.currentTarget.value)}
      />
    </label>
    <label class="field-label">
      <span>称号{titleCustom ? "（管理员自定义）" : ""}</span>
      <input
        value={title}
        maxlength="18"
        class={titleCustom ? "input-title-custom" : undefined}
        title={titleCustom ? "已由管理员手动设置，不随排位分变化自动改写；清空后保存可恢复自动称号" : undefined}
        onfocus={() => { focusedField = "title"; }}
        onblur={() => { if (focusedField === "title") focusedField = null; }}
        oninput={(event) => { title = event.currentTarget.value; titleTouched = true; }}
      />
    </label>
    <label class="field-label">
      <span>积分</span>
      <input
        type="number"
        value={rankedPoints}
        onfocus={() => { focusedField = "rankedPoints"; }}
        onblur={() => { if (focusedField === "rankedPoints") focusedField = null; }}
        oninput={(event) => { rankedPoints = event.currentTarget.value; rankedPointsTouched = true; }}
      />
    </label>
    <label class="field-label">
      <span>白给值</span>
      <input
        type="number"
        min="0" max="100" step="0.1"
        value={giveawayInput}
        onfocus={() => { focusedField = "giveaway"; }}
        onblur={() => { if (focusedField === "giveaway") focusedField = null; }}
        oninput={(event) => { giveawayInput = event.currentTarget.value; giveawayTouched = true; }}
        placeholder="0-100，精确到 0.1"
      />
    </label>
  </div>
  <div
    class="admin-player-row2"
    onfocusin={() => { focusedField = "gender"; }}
    onfocusout={() => { if (focusedField === "gender") focusedField = null; }}
  >
    <FactionSelect
      {config} {factionId}
      onFactionChange={(value) => {
        factionId = value;
        genderId = nextGenderIdForFaction(config, value, genderId);
        genderTouched = true;
      }}
    />
    <GenderSelect {config} {genderId} {factionId} onGenderChange={(value) => { genderId = value; genderTouched = true; }} />
    <ClaimKeyRevealField playerId={player.id} {onError} />
  </div>
  <div class="admin-action-row">
    <button class="primary" onclick={save}>保存玩家资料</button>
    <button onclick={queryAltAccounts} disabled={altAccounts.loading}>{altAccounts.loading ? "查询中…" : "小号查询"}</button>
    <button class="danger-button" onclick={onKick}>踢出</button>
  </div>
  {#if altAccounts.queried}
    <div class="admin-alt-accounts">
      {#if altAccounts.error}<p class="empty">{altAccounts.error}</p>{/if}
      {#if !altAccounts.error && altAccounts.players.length === 0}<p class="empty">未查询到该玩家关联设备登录过的其它账号</p>{/if}
      {#if !altAccounts.error && altAccounts.truncated}<p class="hint">关联范围过大，已达到服务器查询上限，本次结果可能不完整。</p>{/if}
      {#if !altAccounts.error}
        {#each altAccounts.players as alt (alt.id)}
          <div class="admin-alt-accounts-row">
            <PlayerAvatar player={alt} size={24} />
            <PlayerBadge player={alt} compact />
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>
