<script lang="ts">
  // 个人设置面板。源：ui/AppViews.tsx:4695-5199。config/me/theme 原为 props，现直接读
  // sessionStore/uiStore；onClose/onUpdated/onError/onThemeChange/onLoggedOut 全部
  // 替换成直接调用对应 store 方法。
  import { untrack } from "svelte";
  import UserRound from "@lucide/svelte/icons/user-round";
  import Pencil from "@lucide/svelte/icons/pencil";
  import Save from "@lucide/svelte/icons/save";
  import type { PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { tokenKey } from "../../lib/constants";
  import { encodeClaimCode, logout, refreshClaimKey } from "../../lib/session";
  import { withPetBondDefaults, withRankedScoreDefaults, DEFAULT_NAME_WAR_PENALTY_THRESHOLD, DEFAULT_NAME_WAR_RENAME_MIN_POINTS } from "../../lib/normalize";
  import { prepareAvatarImageForUpload } from "../../lib/avatarImage";
  import { formatOnlineDuration } from "../../lib/format";
  import {
    safePlayerStats, formatGiveawayValue, totalOnlineMsOf, genderChoiceError, nextGenderIdForFaction
  } from "../../lib/playerDisplay";
  import {
    currentPushSubscriptionStatus, disablePushSubscription, ensurePushSubscription, fetchPushPreferences,
    requestNotificationPermission, sendTestPush, updatePushPreferences, type PushPreferences, type PushSubscriptionStatus
  } from "../../lib/pushNotify";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";
  import FactionSelect from "../shell/FactionSelect.svelte";
  import GenderSelect from "../shell/GenderSelect.svelte";
  import Toggle from "../shell/Toggle.svelte";
  import Stat from "../shell/Stat.svelte";
  import ClaimKeyPanel from "../shell/ClaimKeyPanel.svelte";

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);

  function onError(message: string) {
    uiStore.notify(message);
  }
  function onClose() {
    uiStore.profileOpen = false;
  }
  function onUpdated(player: PublicPlayer) {
    sessionStore.updateMyProfile(player);
  }

  // 表单初始值只取一次（与原 React useState(me.xxx) 惰性初始化同构）：me 后续因
  // player:batch 等推送更新不应打断用户正在编辑的表单，untrack 显式声明"仅初始化读取"。
  let name = $state(untrack(() => me.name));
  let selfTitle = $state(untrack(() => me.stats.selfTitle || ""));
  let genderId = $state(untrack(() => me.genderId));
  let factionId = $state(untrack(() => me.factionId));
  let nameWarEnabled = $state(untrack(() => Boolean(me.nameWarEnabled)));
  let nameWarAllowRename = $state(untrack(() => Boolean(me.nameWarAllowRename)));
  let giveawayEnabled = $state(untrack(() => Boolean(me.giveawayEnabled)));
  let extremeModeEnabled = $state(untrack(() => Boolean(me.extremeModeEnabled)));
  let bondMasterEnabled = $state(untrack(() => Boolean(me.bondMasterEnabled)));
  let bondPetEnabled = $state(untrack(() => Boolean(me.bondPetEnabled)));
  let bondPublicDisplay = $state(untrack(() => Boolean(me.bondPublicDisplay)));
  let now = $state(Date.now());
  const stats = $derived(safePlayerStats(me));
  const decisive = $derived(stats.wins + stats.losses);
  const winRate = $derived(decisive === 0 ? 0 : Math.round((stats.wins / decisive) * 100));
  const total = $derived(stats.wins + stats.losses + stats.draws);
  const nameChanged = $derived(name.trim() !== me.name);
  const cooldownMs = $derived(me.profileUpdatedAt ? Math.max(0, 60_000 - (now - me.profileUpdatedAt)) : 0);
  const nameCooldownSeconds = $derived(Math.ceil(cooldownMs / 1000));
  const nameWarChanged = $derived(nameWarEnabled !== Boolean(me.nameWarEnabled));
  const nameWarAllowRenameChanged = $derived(nameWarAllowRename !== Boolean(me.nameWarAllowRename));
  const nameWarCooldownMs = $derived(me.nameWarToggledAt ? Math.max(0, 43_200_000 - (now - me.nameWarToggledAt)) : 0);
  const nameWarCooldownHours = $derived(Math.ceil(nameWarCooldownMs / 3_600_000));
  const nameLockedByWar = $derived(Boolean(me.nameWarEnabled || nameWarEnabled));
  const giveawayValue = $derived(me.giveawayValue || 0);
  const giveawayCannotClose = $derived(Boolean(me.giveawayEnabled && !giveawayEnabled && giveawayValue > 0));
  const extremeCooldownMs = $derived(me.extremeModeCooldownUntil ? Math.max(0, me.extremeModeCooldownUntil - now) : 0);
  const extremeCooldownHours = $derived(Math.ceil(extremeCooldownMs / 3_600_000));
  const extremeCannotEnable = $derived(Boolean(!me.extremeModeEnabled && extremeModeEnabled && (stats.rankedPoints < 0 || extremeCooldownMs > 0)));
  const extremeCannotClose = $derived(Boolean(me.extremeModeEnabled && !extremeModeEnabled && stats.rankedPoints <= 0));

  let logoutConfirmOpen = $state(false);
  let logoutClaimCode = $state<string | null>(null);
  let logoutBusy = $state(false);
  let pushPrefs = $state<PushPreferences>({ mentionEnabled: false, turnEnabled: false, seatEnabled: false, bondEnabled: false });
  let notificationPermission = $state<NotificationPermission>(typeof Notification === "undefined" ? "denied" : Notification.permission);
  let pushStatus = $state<PushSubscriptionStatus>(currentPushSubscriptionStatus());
  let pushBusy = $state(false);
  let avatarBusy = $state(false);
  let avatarInputEl: HTMLInputElement | null = $state(null);

  $effect(() => {
    if (!cooldownMs && !nameWarCooldownMs && !extremeCooldownMs) return;
    const timer = window.setInterval(() => { now = Date.now(); }, 1000);
    return () => window.clearInterval(timer);
  });

  $effect(() => {
    fetchPushPreferences().then((next) => { pushPrefs = next; }).catch(() => undefined);
    if (typeof Notification !== "undefined" && Notification.permission === "granted") {
      ensurePushSubscription().then((next) => { pushStatus = next; }).catch(() => undefined);
    }
  });

  async function enableNotifications() {
    pushBusy = true;
    try {
      const permission = await requestNotificationPermission();
      notificationPermission = permission;
      if (permission === "granted") {
        const status = await ensurePushSubscription({ force: true });
        pushStatus = status;
        if (status.state !== "active") onError(status.message);
      } else {
        pushStatus = currentPushSubscriptionStatus();
        onError("需要允许通知权限才能开启推送");
      }
    } finally {
      pushBusy = false;
    }
  }

  async function togglePushPref(key: keyof PushPreferences, value: boolean) {
    if (value && pushStatus.state !== "active") {
      pushBusy = true;
      try {
        // 开偏好时一并走权限请求，避免只 ensure 却卡在 permission-required。
        const permission = await requestNotificationPermission();
        notificationPermission = permission;
        if (permission !== "granted") {
          pushStatus = currentPushSubscriptionStatus();
          onError("需要允许通知权限才能开启推送");
          return;
        }
        const status = await ensurePushSubscription({ force: true });
        pushStatus = status;
        if (status.state !== "active") {
          onError(status.message);
          return;
        }
      } finally {
        pushBusy = false;
      }
    }
    const prev = pushPrefs;
    pushPrefs = { ...pushPrefs, [key]: value };
    try {
      await updatePushPreferences({ [key]: value });
    } catch (error) {
      pushPrefs = prev; // 回滚
      onError(error instanceof Error ? error.message : "保存推送偏好失败");
    }
  }

  async function stopNotificationsOnDevice() {
    pushBusy = true;
    try {
      const status = await disablePushSubscription();
      pushStatus = status;
      onError(status.message);
    } finally {
      pushBusy = false;
    }
  }

  async function testNotifications() {
    pushBusy = true;
    onError("测试通知将在 3 秒后发出；请立即切到后台或锁屏");
    try {
      const result = await sendTestPush();
      onError(
        `推送网关已接受 ${result.acceptedCount}/${result.subscriptionCount} 个设备订阅`
        + (result.failedCount ? `，失败 ${result.failedCount} 个` : "")
      );
    } catch (error) {
      onError(error instanceof Error ? error.message : "发送测试通知失败");
    } finally {
      pushBusy = false;
    }
  }

  $effect(() => {
    if (!nameWarEnabled) nameWarAllowRename = false;
  });

  async function saveProfile() {
    const genderError = genderChoiceError(config, genderId);
    if (genderError) {
      onError(genderError);
      return;
    }
    if (nameChanged && nameLockedByWar) {
      onError("名字争夺战开启后不能修改名字");
      return;
    }
    if (nameChanged && cooldownMs > 0) {
      onError(`改名冷却中，请 ${nameCooldownSeconds} 秒后再试`);
      return;
    }
    if ((nameWarChanged || nameWarAllowRenameChanged) && nameWarCooldownMs > 0) {
      onError(`名字争夺战冷却中，请 ${nameWarCooldownHours} 小时后再试`);
      return;
    }
    if (giveawayCannotClose) {
      onError("白给值归零前不能关闭白给模式");
      return;
    }
    if (extremeCannotEnable) {
      onError(extremeCooldownMs > 0 ? `极限模式冷却中，请 ${extremeCooldownHours} 小时后再开启` : "负分玩家不能开启极限模式");
      return;
    }
    if (extremeCannotClose) {
      onError("排位分必须大于 0 才能关闭极限模式，0 分不能关闭");
      return;
    }
    if (!me.extremeModeEnabled && extremeModeEnabled) {
      const ok = window.confirm("开启极限模式会把当前排位分归零，并禁止进入倍率房。胜负平和惩罚次数会保留。确认开启？");
      if (!ok) return;
    }
    try {
      const result = await ask<{ player: PublicPlayer }>("player:updateProfile", {
        name, genderId, selfTitle,
        nameWarEnabled, nameWarAllowRename, giveawayEnabled, extremeModeEnabled,
        bondMasterEnabled, bondPetEnabled, bondPublicDisplay
      });
      onUpdated(result.player);
      onError("个人资料已更新");
    } catch (error) {
      onError(error instanceof Error ? error.message : "保存失败");
    }
  }

  async function openLogoutConfirm() {
    logoutConfirmOpen = true;
    logoutBusy = true;
    try {
      const result = await refreshClaimKey();
      logoutClaimCode = encodeClaimCode(result.playerId, result.claimKey);
    } catch (error) {
      onError(error instanceof Error ? error.message : "生成认领密钥失败");
      logoutConfirmOpen = false;
    } finally {
      logoutBusy = false;
    }
  }

  async function confirmLogout() {
    logoutBusy = true;
    try {
      await logout();
      window.location.reload();
    } catch (error) {
      onError(error instanceof Error ? error.message : "登出失败");
    } finally {
      logoutBusy = false;
    }
  }

  async function uploadAvatar(file: File) {
    avatarBusy = true;
    try {
      const uploadFile = await prepareAvatarImageForUpload(file);
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("image", uploadFile, "avatar.webp");
      const response = await fetch("/api/avatar-image", { method: "POST", body: form });
      let data: { message?: string; player?: PublicPlayer } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `上传失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "上传失败");
      if (!data.player) throw new Error("服务器未返回玩家资料");
      onUpdated(data.player);
      onError("头像已更新");
    } catch (error) {
      onError(error instanceof Error ? error.message : "头像上传失败");
    } finally {
      avatarBusy = false;
      if (avatarInputEl) avatarInputEl.value = "";
    }
  }

  async function clearAvatar() {
    avatarBusy = true;
    try {
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("clear", "1");
      const response = await fetch("/api/avatar-image", { method: "POST", body: form });
      let data: { message?: string; player?: PublicPlayer } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `清空失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "清空失败");
      if (!data.player) throw new Error("服务器未返回玩家资料");
      onUpdated(data.player);
      onError("已恢复默认头像");
    } catch (error) {
      onError(error instanceof Error ? error.message : "清空头像失败");
    } finally {
      avatarBusy = false;
      if (avatarInputEl) avatarInputEl.value = "";
    }
  }

  async function forceCloseExtremeMode() {
    const ok = window.confirm(config.extremeMode.forceCloseWarning || "强行关闭极限模式后，你会进入通用改名处，可被符合条件的极限玩家改名。确认继续？");
    if (!ok) return;
    try {
      const result = await ask<{ player: PublicPlayer }>("extreme:forceClose", {});
      extremeModeEnabled = false;
      onUpdated(result.player);
      onError("已强行关闭极限模式");
    } catch (error) {
      onError(error instanceof Error ? error.message : "强行关闭失败");
    }
  }

  const gameStatRows: Array<[string, { wins?: number; losses?: number; draws?: number } | undefined]> = $derived([
    ["锤子剪刀布", me.gameStats?.rps], ["黑白棋", me.gameStats?.othello], ["井字棋", me.gameStats?.tictactoe],
    ["五子棋", me.gameStats?.gomoku], ["斗兽棋", me.gameStats?.jungle], ["国际象棋", me.gameStats?.chess], ["大话骰", me.gameStats?.liarsdice]
  ]);
</script>

<div class="modal-backdrop" onclick={onClose}>
  <section class="profile-panel" onclick={(event) => event.stopPropagation()}>
    <div class="profile-hero">
      <div class="avatar-ring">{#if me.avatarUrl}<PlayerAvatar player={me} size={56} />{:else}<UserRound size={34} />{/if}</div>
      <div>
        <h2><PlayerBadge player={me} /></h2>
        <p>{`${stats.title} · ${stats.rankedPoints} 排位积分`}</p>
      </div>
      <button class="profile-close-button" type="button" aria-label="关闭个人设置" onclick={onClose}>×</button>
    </div>

    <div class="profile-edit profile-stats-card">
      <h3>统计数据</h3>
      <div class="profile-stats">
        <Stat label="总局数" value={`${total}`} />
        <Stat label="总胜率" value={`${winRate}%`} />
        <Stat label="惩罚次数" value={`${stats.punishments}`} />
        <Stat label="排位积分" value={`${stats.rankedPoints}`} />
        <Stat label="历史战绩" value={`${stats.lowestScore} ～ ${stats.highestScore}`} />
        <Stat label="当前称号" value={stats.title} />
        <Stat label="在线时长" value={formatOnlineDuration(totalOnlineMsOf(me))} />
        <Stat label="白给值" value={`${formatGiveawayValue(giveawayValue)}%`} />
        <Stat label="极限模式" value={me.extremeModeEnabled ? `连胜 ${me.extremeWinStreak || 0}` : "未开启"} />
      </div>
    </div>

    <div class="profile-edit profile-stats-card">
      <h3>对局详情（胜/负/平）</h3>
      <div class="profile-stats profile-game-stats">
        <Stat label="对局总计" value={`${stats.wins} / ${stats.losses} / ${stats.draws}`} />
        {#each gameStatRows as [label, g] (label)}
          <Stat {label} value={`${g?.wins || 0} / ${g?.losses || 0} / ${g?.draws || 0}`} />
        {/each}
      </div>
    </div>

    <div class="profile-edit">
      <h3><Pencil size={18} /> 修改资料</h3>
      <div class="profile-edit-grid">
        <label class="field-label profile-name-field">
          <span>名字</span>
          <input value={name} maxlength="12" disabled={nameLockedByWar} oninput={(event) => (name = event.currentTarget.value)} placeholder="新的名字" />
          <small>{nameLockedByWar ? "名字争夺战开启后不能修改名字" : nameChanged && cooldownMs > 0 ? `改名冷却：${nameCooldownSeconds} 秒` : "名字会显示在大厅、房间和聊天里"}</small>
        </label>
        <label class="field-label">
          <span>称号</span>
          <input value={selfTitle} maxlength="12" oninput={(event) => (selfTitle = event.currentTarget.value)} placeholder="称号" />
        </label>
        <FactionSelect
          {config} {factionId}
          onFactionChange={(nextFactionId) => {
            factionId = nextFactionId;
            genderId = nextGenderIdForFaction(config, nextFactionId, genderId);
          }}
        />
        <GenderSelect {config} {genderId} {factionId} onGenderChange={(next) => (genderId = next)} />
        <div class="profile-edit-grid-full profile-avatar-upload-row">
          <input
            bind:this={avatarInputEl}
            type="file"
            accept="image/*"
            class="profile-avatar-upload-input"
            disabled={avatarBusy}
            onchange={(event) => {
              const file = event.currentTarget.files?.[0];
              if (file) void uploadAvatar(file);
            }}
          />
          <button type="button" disabled={avatarBusy} onclick={() => avatarInputEl?.click()}>{avatarBusy ? "上传中…" : "上传头像"}</button>
          <button type="button" disabled={avatarBusy || !me.avatarUrl} onclick={() => void clearAvatar()}>清空头像</button>
        </div>
      </div>
      <p class="hint">上次改名：{me.profileUpdatedAt ? new Date(me.profileUpdatedAt).toLocaleString() : "还没有修改过"}</p>
      <div class="name-war-card">
        <div class="admin-card-title">
          <strong>夜间模式</strong>
          <small>{uiStore.theme === "dark" ? "已开启" : "未开启"}</small>
        </div>
        <Toggle label="开启夜间模式" value={uiStore.theme === "dark"} onChange={(value) => uiStore.setTheme(value ? "dark" : "light")} />
      </div>
      <div class="name-war-card">
        <div class="admin-card-title">
          <strong>名字争夺战</strong>
          <small>{me.nameWarEnabled ? "已开启" : "未开启"}</small>
        </div>
        <Toggle label="开启名字争夺战" value={nameWarEnabled} disabled={nameWarCooldownMs > 0} onChange={(v) => (nameWarEnabled = v)} />
        <Toggle label="允许其他玩家改名" value={nameWarAllowRename} disabled={!nameWarEnabled || nameWarCooldownMs > 0} onChange={(v) => (nameWarAllowRename = v)} />
        <p class="hint">开启后排位分展示下限变为 {withRankedScoreDefaults(config.rankedScore).nameWarMin}；积分跌到 {config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} 及以下后，只显示系统惩罚名，不显示性别和称号。</p>
        <p class="hint">允许其他玩家改名后，真实分跌到 {config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD} 及以下会出现在大厅失格者名单，{config.nameWar?.renameMinPoints ?? DEFAULT_NAME_WAR_RENAME_MIN_POINTS} 分以上玩家可以抢先给你改名。</p>
        <p class="hint">被其他玩家改名后，保护期内即使真实分回到失格线以上也不会提前恢复；保护期结束且真实分高于失格线才恢复。</p>
        {#if me.nameWarRenameProtectedUntil && me.nameWarRenameProtectedUntil > now}<p class="hint">改名保护中：约 {Math.ceil((me.nameWarRenameProtectedUntil - now) / 3_600_000)} 小时。</p>{/if}
        {#if me.nameWarPunished && me.nameWarPenaltyName}<p class="hint">当前惩罚名：{me.nameWarPenaltyName}。</p>{/if}
        {#if nameWarCooldownMs > 0}<p class="hint">开关冷却：{nameWarCooldownHours} 小时</p>{/if}
      </div>
      <div class="name-war-card giveaway-profile-card">
        <div class="admin-card-title">
          <strong>白给模式</strong>
          <small>{me.giveawayEnabled ? `${formatGiveawayValue(giveawayValue)}%` : "未开启"}</small>
        </div>
        <Toggle label="开启白给模式" value={giveawayEnabled} disabled={giveawayCannotClose} onChange={(v) => (giveawayEnabled = v)} />
        <p class="hint">开启后白给值默认为 0.1%；锤子剪刀布会按概率改为白给，黑白棋排位支持白给/上贡，井字棋支持随机白给，五子棋支持主动选择或按概率把本手落子变成对方棋子。</p>
        <p class="hint">出拳区点击"白给"会让白给值 +{formatGiveawayValue(config.giveaway.activeBoostValue)}%，触发强制白给后也会 +{formatGiveawayValue(config.giveaway.activeBoostValue)}%，最高 100%。</p>
        <p class="hint">黑白棋白给会让本手翻子不结算排位分并按 0.1%/子增加白给值；上贡会把本手分数给对面并按 0.2%/子增加白给值。</p>
        <p class="hint">白给值归零后，该模式自动关闭。游玩任何游戏获胜会让白给值 -{formatGiveawayValue(config.giveaway.winPenaltyValue)}%；也可以在大厅的白给自救板提交宣言，等待其他玩家点赞帮你降低；</p>
        {#if giveawayCannotClose}<p class="hint danger-hint">当前还有 {formatGiveawayValue(giveawayValue)}% 白给值，暂时不能关闭。</p>{/if}
      </div>
      <div class="name-war-card extreme-profile-card">
        <div class="admin-card-title">
          <strong>{config.extremeMode.emoji} {config.extremeMode.label}</strong>
          <small>{me.extremeModeEnabled ? `连胜 ${me.extremeWinStreak || 0}` : extremeCooldownMs > 0 ? `冷却 ${extremeCooldownHours} 小时` : "未开启"}</small>
        </div>
        <Toggle label="开启极限模式" value={extremeModeEnabled} disabled={(!me.extremeModeEnabled && (stats.rankedPoints < 0 || extremeCooldownMs > 0)) || extremeCannotClose} onChange={(v) => (extremeModeEnabled = v)} />
        <p class="hint">开启要求当前排位分不为负；开启后排位分归零，但胜负平和惩罚次数保留。</p>
        <p class="hint">极限模式不能创建倍率房；进入普通排位或倍率房时只能观战，不能上桌。非极限玩家进入极限排位房也只能观战。</p>
        <p class="hint">正分输分和负分加分会按段位折扣；整点会自动扣分，离线也会扣。</p>
        <p class="hint">极限排位连胜达到 {config.extremeMode.winStreakThreshold} 局后，每次继续获胜都有 {Math.round(config.extremeMode.winStreakCrashChance * 100)}% 几率额外扣 {config.extremeMode.crashTargetPoints} 分。</p>
        {#if stats.rankedPoints < 0 && !me.extremeModeEnabled}<p class="hint danger-hint">你当前是负分，不能开启极限模式。</p>{/if}
        {#if extremeCannotClose}<p class="hint danger-hint">排位分必须大于 0 才能关闭极限模式，0 分不能关闭。</p>{/if}
        {#if me.extremeForceClosed}<p class="hint danger-hint">你曾强行关闭极限模式，已进入通用改名处。</p>{/if}
        {#if me.extremeRenameProtectedUntil && me.extremeRenameProtectedUntil > now}<p class="hint">极限改名保护中：约 {Math.ceil((me.extremeRenameProtectedUntil - now) / 3_600_000)} 小时。</p>{/if}
        {#if extremeCooldownMs > 0}<p class="hint">关闭后冷却：约 {extremeCooldownHours} 小时后可重新开启。</p>{/if}
        {#if me.extremeModeEnabled}<button type="button" class="danger-button" onclick={forceCloseExtremeMode}>强行关闭极限模式</button>{/if}
      </div>
      <div class="name-war-card pet-bond-profile-card">
        <div class="admin-card-title">
          <strong>🐾 认主 / 认宠</strong>
          <small>{[bondMasterEnabled && "认主", bondPetEnabled && "认宠", bondPublicDisplay && "公开"].filter(Boolean).join(" · ") || "未开启"}</small>
        </div>
        <Toggle label="开启认主" value={bondMasterEnabled} onChange={(v) => (bondMasterEnabled = v)} />
        <Toggle label="开启认宠" value={bondPetEnabled} onChange={(v) => (bondPetEnabled = v)} />
        <Toggle label="公开展示" value={bondPublicDisplay} onChange={(v) => (bondPublicDisplay = v)} />
        <p class="hint">允许同时开启认主与认宠。关闭开关不会取消已有关系，但无法再新增主/宠。</p>
        <p class="hint">关闭「公开展示」后，你不会出现在大厅关系图谱中（已有关系仍保留）。</p>
        <p class="hint">主人最多 {withPetBondDefaults(config.petBond).maxPetsPerMaster} 只宠物，宠物最多 {withPetBondDefaults(config.petBond).maxMastersPerPet} 位主人；认新主人需已有主人全体同意。</p>
      </div>
      <div class="name-war-card push-settings-card">
        <div class="admin-card-title">
          <strong>🔔 推送通知</strong>
          <small>{pushStatus.state === "active" ? "已启用" : pushStatus.state === "checking" ? "检查中" : pushStatus.state === "stopped" ? "已停止" : "未启用"}</small>
        </div>
        <p class={`hint ${pushStatus.state === "error" || pushStatus.state === "unsupported" || pushStatus.state === "denied" ? "danger-hint" : ""}`}>{pushStatus.message}</p>
        {#if pushStatus.state === "unsupported"}<p class="hint">iPhone/iPad 请先将本站"添加到主屏幕"，再从主屏幕图标打开；其他设备请使用支持 Web Push 的现代浏览器。</p>{/if}
        {#if notificationPermission !== "granted" && pushStatus.state !== "unsupported"}
          <p class="hint">开启后，页面在后台、手机锁屏或浏览器关闭时均可收到通知；前台正在查看本站时不会重复弹出。</p>
          <button type="button" disabled={pushBusy} onclick={enableNotifications}>{pushBusy ? "请求中…" : "开启推送通知"}</button>
          {#if notificationPermission === "denied"}<p class="hint danger-hint">浏览器已拒绝通知权限，需要在浏览器设置里手动允许后再试。</p>{/if}
        {/if}
        {#if notificationPermission === "granted"}
          {#if pushStatus.state !== "active"}
            <button type="button" disabled={pushBusy} onclick={enableNotifications}>{pushBusy ? "订阅中…" : "订阅推送通知"}</button>
          {/if}
          <Toggle label="有人 @ 我" value={pushPrefs.mentionEnabled} onChange={(v) => togglePushPref("mentionEnabled", v)} />
          <Toggle label="轮到我出招/落子" value={pushPrefs.turnEnabled} onChange={(v) => togglePushPref("turnEnabled", v)} />
          <Toggle label="我的房间参战席被坐满" value={pushPrefs.seatEnabled} onChange={(v) => togglePushPref("seatEnabled", v)} />
          <Toggle label="我的主人/宠物上线" value={pushPrefs.bondEnabled} onChange={(v) => togglePushPref("bondEnabled", v)} />
          {#if pushStatus.state === "active"}
            <div class="profile-action-row">
              <button type="button" disabled={pushBusy} onclick={testNotifications}>测试通知</button>
              <button type="button" class="danger-button" disabled={pushBusy} onclick={stopNotificationsOnDevice}>停止通知</button>
            </div>
          {/if}
        {/if}
      </div>
      <div class="name-war-card account-devices-card">
        <div class="admin-card-title">
          <strong>账号与设备</strong>
          <small>最多同时记住 3 台设备</small>
        </div>
        <ClaimKeyPanel onError={onError} />
        <button type="button" class="danger-button" onclick={openLogoutConfirm}>登出</button>
      </div>
      <div class="profile-action-row">
        <button class="primary" disabled={(nameChanged && (cooldownMs > 0 || nameLockedByWar)) || ((nameWarChanged || nameWarAllowRenameChanged) && nameWarCooldownMs > 0) || giveawayCannotClose || extremeCannotEnable || extremeCannotClose} onclick={saveProfile}>
          <Save size={16} /> 保存资料
        </button>
        <button type="button" onclick={onClose}>关闭设置</button>
      </div>
    </div>
  </section>
  {#if logoutConfirmOpen}
    <div class="modal-backdrop logout-confirm-backdrop" onclick={() => !logoutBusy && (logoutConfirmOpen = false)}>
      <section class="logout-confirm-card" onclick={(event) => event.stopPropagation()}>
        <h3>确认登出？</h3>
        <p class="hint danger-hint">
          登出后将无法再次登录当前账号，除非使用下面这把认领密钥在需要的设备上恢复。请务必先保存到私密位置（不要发给任何人）。
        </p>
        {#if logoutClaimCode}<code class="claim-key-code">{logoutClaimCode}</code>{:else}<p class="hint">正在生成密钥…</p>{/if}
        <div class="kick-confirm-actions">
          <button disabled={logoutBusy} onclick={() => (logoutConfirmOpen = false)}>取消</button>
          <button class="danger-button" disabled={logoutBusy || !logoutClaimCode} onclick={confirmLogout}>{logoutBusy ? "处理中…" : "我已保存密钥，确认登出"}</button>
        </div>
      </section>
    </div>
  {/if}
</div>
