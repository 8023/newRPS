<script lang="ts">
  // 源：ui/AppViews.tsx:677-1372（建房弹窗，含 timedGameSettingKeys 已拆到
  // GameTimerSettings.svelte，房名生成/标签过滤已拆到 lib/roomNaming.ts）。
  // 只有「是否展示这个弹窗」由 Lobby 通过 onCancel/onCreated 回调控制——这是真正
  // 局部于调用方的 UI 状态；其余数据一律直接读 sessionStore/uiStore。
  import { untrack } from "svelte";
  import type { RoomSettings, RoomSnapshot } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { stakeTiersFor } from "../../lib/normalize";
  import { normalizeRoomSnapshot } from "../../lib/normalize";
  import {
    defaultChessRoomName, defaultCoinFlipRoomName, defaultGomokuRoomName, defaultJungleRoomName, defaultLiarsDiceRoomName,
    defaultOthelloRoomName, defaultRoomName, defaultTicTacToeRoomName,
    chessBoardThemes, gomokuBoardThemes, jungleBoardThemes, othelloBoardThemes, tictactoeBoardThemes
  } from "../../lib/constants";
  import { filterTagIds, generateRoomName, sameStringArray } from "../../lib/roomNaming";
  import { moveSecondsOptions, gameMinutesOptions } from "../../lib/constants";
  import { rankMultiplierForSettings } from "../../lib/playerDisplay";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import GameIcon from "../shell/GameIcon.svelte";
  import Toggle from "../shell/Toggle.svelte";
  import Select from "../shell/Select.svelte";
  import TagPicker from "./TagPicker.svelte";
  import GameTimerSettings, { isTimedGame } from "./GameTimerSettings.svelte";

  let { onCreated, onCancel }: { onCreated: (room?: RoomSnapshot) => void; onCancel: () => void } = $props();

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);

  function preventEnterSubmit(event: KeyboardEvent) {
    if (event.key === "Enter") event.preventDefault();
  }

  // 只取一次初始值：建房面板打开期间 config 理论上不会变，untrack 只是显式声明
  // "这里不是响应式读取"，避免与后面 settings 内部的响应式 effect 混淆。
  const { tagIds, initialPrefs } = untrack(() => {
    const ids = (config.punishmentTags || []).map((t) => t.id);
    const included: string[] = [];
    const excluded: string[] = [];
    for (const id of ids) {
      const state = uiStore.punishmentTagPrefs[id];
      if (state === "include") included.push(id);
      else if (state === "exclude") excluded.push(id);
    }
    return { tagIds: ids, initialPrefs: { included, excluded } };
  });

  let settings = $state<RoomSettings>({
    name: defaultRoomName,
    gameId: "rps",
    enablePunishment: false,
    enablePerPiecePunishment: false,
    punishmentSource: "random",
    punishmentTagsIncluded: initialPrefs.included,
    punishmentTagsExcluded: initialPrefs.excluded,
    punishmentSeriesId: untrack(() => config.punishmentSeriesSummaries?.[0]?.id || ""),
    enableTags: false,
    tags: [],
    allowProofImage: true,
    tieDoublePunish: false,
    // 默认需要胜方审批证明（系统惩罚与自定义任务一致，避免提交后直接开新局）。
    requireOpponentConfirm: true,
    enableRanked: false,
    stake: 5,
    enableRankMultiplier: false,
    rankMultiplier: 1,
    enableExtremeRanked: false,
    othelloBoardTheme: "classic",
    tictactoeBoardTheme: "paper",
    gomokuBoardTheme: "wood",
    jungleBoardTheme: "forest",
    chessBoardTheme: "classic",
    gomokuUndoLimit: 0,
    jungleUndoLimit: 0,
    chessUndoLimit: 0,
    othelloUndoLimit: 0,
    liarsDiceMinPlayers: 3,
    liarsDiceMaxPlayers: 3,
    othelloMoveSeconds: 0,
    othelloGameMinutes: 0,
    gomokuMoveSeconds: 0,
    gomokuGameMinutes: 0,
    jungleMoveSeconds: 0,
    jungleGameMinutes: 0,
    chessMoveSeconds: 0,
    chessGameMinutes: 0
  });
  let customRoomName = $state(false);
  let seriesSearch = $state("");

  // 搜索框留空＝不启用筛选，下拉里能选任何系列任务；填了关键字则只在标题命中的条目里选。
  const filteredSeries = $derived.by(() => {
    const all = config.punishmentSeriesSummaries || [];
    const keyword = seriesSearch.trim().toLowerCase();
    return keyword ? all.filter((series) => (series.name || "").toLowerCase().includes(keyword)) : all;
  });

  // 搜索结果变化后，当前选中项若不再在结果里，自动改选第一条；没有结果就清空选择
  // （对应下面把 <select> 禁用、创建房间也一并挡住，而不是留着一个选不中的旧值）。
  $effect(() => {
    if (settings.punishmentSource !== "series") return;
    if (filteredSeries.some((series) => series.id === settings.punishmentSeriesId)) return;
    settings = { ...settings, punishmentSeriesId: filteredSeries[0]?.id || "" };
  });

  $effect(() => {
    void config.punishmentTags;
    void config.punishmentSeriesSummaries;
    void config.roomTags;
    let changed = false;
    const next = { ...settings };
    const validIncl = filterTagIds(config, next.punishmentTagsIncluded || []);
    const validExcl = filterTagIds(config, next.punishmentTagsExcluded || []);
    if ((next.punishmentSource || "random") === "random" || next.punishmentSource === "system") {
      if (!sameStringArray(validIncl, next.punishmentTagsIncluded || []) || !sameStringArray(validExcl, next.punishmentTagsExcluded || [])) {
        next.punishmentTagsIncluded = validIncl;
        next.punishmentTagsExcluded = validExcl;
        changed = true;
      }
    }
    // 系列任务的有效性交给上面按 filteredSeries 联动的 effect 统一处理（它还要顾及搜索关键字）。
    if (next.tags?.some((tag) => !config.roomTags.includes(tag))) {
      next.tags = next.tags.filter((tag) => config.roomTags.includes(tag));
      next.enableTags = Boolean(next.tags.length);
      changed = true;
    }
    if (changed) settings = next;
  });

  function patch(next: Partial<RoomSettings>) {
    const old = settings;
    const merged: RoomSettings = { ...old, ...next };
    if (!customRoomName && next.gameId) {
      merged.name = next.gameId === "othello" ? defaultOthelloRoomName
        : next.gameId === "tictactoe" ? defaultTicTacToeRoomName
          : next.gameId === "liarsdice" ? defaultLiarsDiceRoomName
            : next.gameId === "gomoku" ? defaultGomokuRoomName
              : next.gameId === "jungle" ? defaultJungleRoomName
                : next.gameId === "chess" ? defaultChessRoomName
                  : next.gameId === "coinflip" ? defaultCoinFlipRoomName
                    : defaultRoomName;
    }
    if (next.gameId === "othello" || merged.gameId === "othello") {
      merged.othelloBoardTheme = merged.othelloBoardTheme || "classic";
      merged.othelloUndoLimit = merged.othelloUndoLimit ?? 0;
    }
    if (next.gameId === "tictactoe" || merged.gameId === "tictactoe") {
      merged.tictactoeBoardTheme = merged.tictactoeBoardTheme || "paper";
    }
    if (next.gameId === "gomoku" || merged.gameId === "gomoku") {
      merged.gomokuBoardTheme = merged.gomokuBoardTheme || "wood";
      merged.gomokuUndoLimit = merged.gomokuUndoLimit ?? 0;
    }
    if (next.gameId === "jungle" || merged.gameId === "jungle") {
      merged.jungleBoardTheme = merged.jungleBoardTheme || "forest";
      merged.jungleUndoLimit = merged.jungleUndoLimit ?? 0;
    }
    if (next.gameId === "chess" || merged.gameId === "chess") {
      merged.chessBoardTheme = merged.chessBoardTheme || "classic";
      merged.chessUndoLimit = merged.chessUndoLimit ?? 0;
    }
    if (next.gameId && next.gameId !== "chess" && next.gameId !== "jungle") {
      merged.enablePerPiecePunishment = false;
    }
    const timedKeys = {
      othello: { move: "othelloMoveSeconds", game: "othelloGameMinutes" },
      gomoku: { move: "gomokuMoveSeconds", game: "gomokuGameMinutes" },
      jungle: { move: "jungleMoveSeconds", game: "jungleGameMinutes" },
      chess: { move: "chessMoveSeconds", game: "chessGameMinutes" }
    } as const;
    for (const keys of Object.values(timedKeys)) {
      const moveSeconds = next[keys.move as keyof RoomSettings];
      if (moveSeconds !== undefined && !moveSecondsOptions.includes(moveSeconds as typeof moveSecondsOptions[number])) {
        (merged as any)[keys.move] = 0;
      }
      const gameMinutes = next[keys.game as keyof RoomSettings];
      if (gameMinutes !== undefined && !gameMinutesOptions.includes(gameMinutes as typeof gameMinutesOptions[number])) {
        (merged as any)[keys.game] = 0;
      }
    }
    if (next.gameId === "liarsdice" || merged.gameId === "liarsdice") {
      merged.liarsDiceMinPlayers = merged.liarsDiceMinPlayers || 3;
      merged.liarsDiceMaxPlayers = merged.liarsDiceMaxPlayers || 3;
    }
    if ("liarsDiceMaxPlayers" in next) {
      const maxP = Math.min(8, Math.max(2, next.liarsDiceMaxPlayers || 3));
      merged.liarsDiceMaxPlayers = maxP;
      if ((merged.liarsDiceMinPlayers || 3) > maxP) merged.liarsDiceMinPlayers = maxP;
    }
    if ("liarsDiceMinPlayers" in next) {
      const maxP = merged.liarsDiceMaxPlayers || 3;
      merged.liarsDiceMinPlayers = Math.min(maxP, Math.max(2, next.liarsDiceMinPlayers || 3));
    }
    if (next.punishmentSource === "player" || next.punishmentSource === "series" || next.punishmentSource === "random") {
      merged.enablePunishment = true;
    }
    if (next.enableRanked === false) {
      merged.enableRankMultiplier = false;
      merged.rankMultiplier = 1;
      merged.enableExtremeRanked = false;
    }
    if (next.enableRankMultiplier) {
      merged.enableRanked = true;
      merged.enableExtremeRanked = false;
      if (!([2, 5, 10] as const).includes(merged.rankMultiplier as 2 | 5 | 10)) merged.rankMultiplier = 2;
    }
    if (next.enableExtremeRanked) {
      merged.enableRanked = true;
      merged.enableRankMultiplier = false;
      merged.rankMultiplier = 1;
    }
    if (!merged.enableRankMultiplier) {
      merged.rankMultiplier = 1;
    }
    if (!merged.enableRanked) {
      merged.enableExtremeRanked = false;
    }
    if (merged.gameId === "coinflip") {
      // 猜硬币没有真人对手：结构性关闭排位/倍率/极限/平局双罚/需对手确认，
      // 并强制开启惩罚（唯一的玩法内容）；玩家发布任务没有对手可以发布，降级为随机任务。
      merged.enableRanked = false;
      merged.enableRankMultiplier = false;
      merged.rankMultiplier = 1;
      merged.enableExtremeRanked = false;
      merged.tieDoublePunish = false;
      merged.requireOpponentConfirm = false;
      merged.enablePunishment = true;
      if (merged.punishmentSource === "player") merged.punishmentSource = "random";
    }
    // 档位按游戏从服务端配置读取；不在档位内回退默认档（首个）。
    const tiers = stakeTiersFor(config, merged.gameId);
    if (!tiers.includes(merged.stake)) merged.stake = tiers[0];
    if (merged.gameId === "othello") {
      if (merged.enableExtremeRanked) {
        merged.enableRankMultiplier = false;
        merged.rankMultiplier = 1;
      }
    }
    const src = merged.punishmentSource === "system" ? "random" : (merged.punishmentSource || "random");
    merged.punishmentSource = src;
    if (src === "random") {
      merged.punishmentTagsIncluded = filterTagIds(config, merged.punishmentTagsIncluded || []);
      merged.punishmentTagsExcluded = filterTagIds(config, merged.punishmentTagsExcluded || []);
      merged.punishmentSeriesId = "";
    } else if (src === "series") {
      merged.punishmentTagsIncluded = [];
      merged.punishmentTagsExcluded = [];
      if (!merged.punishmentSeriesId) {
        merged.punishmentSeriesId = config.punishmentSeriesSummaries?.[0]?.id || "";
      }
    } else {
      merged.punishmentTagsIncluded = [];
      merged.punishmentTagsExcluded = [];
      merged.punishmentSeriesId = "";
    }
    if (next.enableTags === false) {
      merged.tags = [];
    }
    if (next.tags) {
      merged.tags = next.tags.filter((tag) => config.roomTags.includes(tag)).slice(0, 5);
      merged.enableTags = merged.tags.length > 0;
    }
    if (!customRoomName && merged.enablePunishment && ("enablePunishment" in next || "punishmentTagsIncluded" in next || "punishmentTagsExcluded" in next || "punishmentSeriesId" in next || "punishmentSource" in next)) {
      merged.name = generateRoomName(config, merged);
    }
    settings = merged;
  }

  /** 三态循环：缺省 → 选中 → 拒绝 → 缺省 */
  function cycleTagState(tagId: string) {
    const included = new Set(settings.punishmentTagsIncluded || []);
    const excluded = new Set(settings.punishmentTagsExcluded || []);
    if (included.has(tagId)) {
      included.delete(tagId);
      excluded.add(tagId);
    } else if (excluded.has(tagId)) {
      excluded.delete(tagId);
    } else {
      included.add(tagId);
    }
    patch({
      punishmentTagsIncluded: (config.punishmentTags || []).map((t) => t.id).filter((id) => included.has(id)),
      punishmentTagsExcluded: (config.punishmentTags || []).map((t) => t.id).filter((id) => excluded.has(id))
    });
  }

  function tagTriState(tagId: string): "default" | "include" | "exclude" {
    if ((settings.punishmentTagsIncluded || []).includes(tagId)) return "include";
    if ((settings.punishmentTagsExcluded || []).includes(tagId)) return "exclude";
    return "default";
  }

  async function create() {
    try {
      const result = await ask<{ room: RoomSnapshot }>("room:create", { settings });
      // 随机任务模式下，创建成功即把本次标签三态偏好写回本地浏览器存储，供下次开房面板预填。
      if (settings.enablePunishment && (settings.punishmentSource === "random" || settings.punishmentSource === "system")) {
        const prefs: Record<string, string> = {};
        for (const id of settings.punishmentTagsIncluded || []) prefs[id] = "include";
        for (const id of settings.punishmentTagsExcluded || []) prefs[id] = "exclude";
        uiStore.setPunishmentTagPrefs(prefs);
      }
      onCreated(normalizeRoomSnapshot(result.room));
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "创建失败");
    }
  }

  async function unlockMultiplierMode() {
    try {
      await ask("rankMultiplier:unlock", {});
      uiStore.notify("倍率模式已解锁，已扣除 200 排位积分。");
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "解锁失败");
    }
  }

</script>

<div class="modal-backdrop" onclick={(event) => { if (event.target === event.currentTarget) event.stopPropagation(); }}>
  <section class="create-modal" onclick={(event) => event.stopPropagation()}>
    <div class="modal-title">
      <div>
        <h2>🏠 创建房间</h2>
        <p class="hint">选择玩法后就可以邀请朋友加入。</p>
      </div>
      <button type="button" class="icon-button" onclick={onCancel}>×</button>
    </div>
    <div class="create-scroll-area">
      <div class="create-box">
        <div class="create-section game-create-section">
          <h3>游戏</h3>
          <div class="game-choice-grid">
            {#each config.games as game (game.id)}
              <button
                type="button"
                class={`game-choice-card ${settings.gameId === game.id ? "active" : ""}`}
                onclick={() => patch({ gameId: game.id, ...(game.id === "liarsdice" ? { tieDoublePunish: false } : {}) })}
              >
                <span class="game-choice-icon" aria-hidden="true"><GameIcon gameId={game.id} /></span>
                <span class="game-choice-copy">
                  <strong>{game.name}</strong>
                  <small>{game.description}</small>
                </span>
              </button>
            {/each}
          </div>
          {#if settings.gameId === "othello"}<p class="hint">黑白棋支持真人 1v1、观战、聊天、排位和惩罚；排位房会支持白给/上贡结算，可向对方请求悔棋或认输。</p>{/if}
          {#if settings.gameId === "tictactoe"}<p class="hint">井字棋支持真人 1v1、观战、聊天、排位和惩罚；双方准备后随机 X/O 先手。</p>{/if}
          {#if settings.gameId === "liarsdice"}<p class="hint">大话骰支持 2-8 人参战，进房默认观战，可自由加入/离开参战席；全员准备且名单 5 秒无变动后自动开局。</p>{/if}
          {#if settings.gameId === "gomoku"}<p class="hint">五子棋支持真人 1v1、观战、聊天、排位和惩罚；15x15 棋盘先连成五子者胜，可向对方请求悔棋或认输。</p>{/if}
          {#if settings.gameId === "jungle"}<p class="hint">斗兽棋支持真人 1v1、观战、聊天、排位和惩罚；7×9 棋盘，先入对方兽穴或令对方无子可走者胜，可向对方请求悔棋或认输。开启白给后，轮到你时按白给值一半的概率跳过本手。</p>{/if}
          {#if settings.gameId === "chess"}<p class="hint">国际象棋支持真人 1v1、观战、聊天、排位和惩罚；8×8 棋盘白方先走，将死对方的王获胜，可向对方请求悔棋或认输。</p>{/if}
          {#if settings.gameId === "coinflip"}<p class="hint">猜硬币不需要对手，一个人就能玩：坐到参战席猜「字」还是「花」，服务器立即抛硬币公开结算，猜错就会收到系统随机/系列任务；不计入排位积分、胜负场次或白给值，仍支持观战和聊天。</p>{/if}
          {#if isTimedGame(settings.gameId)}
            <GameTimerSettings gameId={settings.gameId} {settings} onPatch={patch} />
          {/if}
          {#if settings.gameId === "liarsdice"}
            <div class="liarsdice-roster-settings">
              <label>
                最少参战人数
                <select value={settings.liarsDiceMinPlayers ?? 3} onchange={(event) => patch({ liarsDiceMinPlayers: Number(event.currentTarget.value) })}>
                  {#each Array.from({ length: (settings.liarsDiceMaxPlayers ?? 3) - 1 }, (_, i) => i + 2) as n (n)}
                    <option value={n}>{n} 人</option>
                  {/each}
                </select>
              </label>
              <label>
                最多参战人数
                <select value={settings.liarsDiceMaxPlayers ?? 3} onchange={(event) => patch({ liarsDiceMaxPlayers: Number(event.currentTarget.value) })}>
                  {#each Array.from({ length: 7 }, (_, i) => i + 2) as n (n)}
                    <option value={n}>{n} 人</option>
                  {/each}
                </select>
              </label>
            </div>
          {/if}
          {#if settings.gameId === "othello"}
            <div class="othello-theme-grid">
              {#each othelloBoardThemes as theme (theme.id)}
                <button type="button" class={`othello-theme-card ${settings.othelloBoardTheme === theme.id ? "active" : ""}`} onclick={() => patch({ othelloBoardTheme: theme.id })}
                  style={`--theme-board:${theme.board};--theme-cell:${theme.cell};--theme-line:${theme.line};--theme-border:${theme.border};--theme-black-disc:${theme.blackDisc};--theme-white-disc:${theme.whiteDisc};--theme-black-ring:${theme.blackRing};--theme-white-ring:${theme.whiteRing}`}>
                  <span class="othello-theme-preview"><i><b class="preview-disc black"></b></i><i></i><i></i><i><b class="preview-disc white"></b></i></span>
                  <strong>{theme.name}</strong>
                  <small>{theme.description}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if settings.gameId === "tictactoe"}
            <div class="tictactoe-theme-grid">
              {#each tictactoeBoardThemes as theme (theme.id)}
                <button type="button" class={`tictactoe-theme-card ${settings.tictactoeBoardTheme === theme.id ? "active" : ""}`} onclick={() => patch({ tictactoeBoardTheme: theme.id })}
                  style={`--ttt-board:${theme.board};--ttt-cell:${theme.cell};--ttt-line:${theme.line};--ttt-border:${theme.border};--ttt-x:${theme.x};--ttt-o:${theme.o}`}>
                  <span class="tictactoe-theme-preview"><i>×</i><i></i><i>○</i><i></i><i>×</i><i></i><i>○</i><i></i><i>×</i></span>
                  <strong>{theme.name}</strong>
                  <small>{theme.description}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if settings.gameId === "gomoku"}
            <div class="othello-theme-grid">
              {#each gomokuBoardThemes as theme (theme.id)}
                <button type="button" class={`othello-theme-card ${settings.gomokuBoardTheme === theme.id ? "active" : ""}`} onclick={() => patch({ gomokuBoardTheme: theme.id })}
                  style={`--theme-board:${theme.board};--theme-cell:${theme.cell};--theme-line:${theme.line};--theme-border:${theme.border};--theme-black-disc:${theme.blackDisc};--theme-white-disc:${theme.whiteDisc};--theme-black-ring:${theme.blackRing};--theme-white-ring:${theme.whiteRing}`}>
                  <span class="othello-theme-preview"><i><b class="preview-disc black"></b></i><i></i><i></i><i><b class="preview-disc white"></b></i></span>
                  <strong>{theme.name}</strong>
                  <small>{theme.description}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if settings.gameId === "chess"}
            <div class="othello-theme-grid">
              {#each chessBoardThemes as theme (theme.id)}
                <button type="button" class={`othello-theme-card ${settings.chessBoardTheme === theme.id ? "active" : ""}`} onclick={() => patch({ chessBoardTheme: theme.id })}
                  style={`--theme-board:${theme.board};--theme-cell:${theme.light};--theme-line:${theme.dark};--theme-hover:${theme.hover};--theme-border:${theme.border}`}>
                  <span class="othello-theme-preview chess-theme-preview" aria-hidden="true"><i><b class="chess-preview-piece light">♟</b></i><i><b class="chess-preview-piece dark">♟</b></i><i><b class="chess-preview-piece dark">♟</b></i><i><b class="chess-preview-piece light">♟</b></i></span>
                  <strong>{theme.name}</strong>
                  <small>{theme.description}</small>
                </button>
              {/each}
            </div>
          {/if}
          {#if settings.gameId === "jungle"}
            <div class="othello-theme-grid">
              {#each jungleBoardThemes as theme (theme.id)}
                <button type="button" class={`othello-theme-card ${settings.jungleBoardTheme === theme.id ? "active" : ""}`} onclick={() => patch({ jungleBoardTheme: theme.id })}
                  style={`--theme-board:${theme.board};--theme-cell:${theme.land};--theme-line:${theme.water};--theme-border:${theme.border}`}>
                  <span class="othello-theme-preview"><i></i><i></i><i></i><i></i></span>
                  <strong>{theme.name}</strong>
                  <small>{theme.description}</small>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <!-- 基础＋玩法放同一独立列，惩罚单独一列：两列各自按自身内容撑高，互不拉伸对方
             （CSS Grid 同行拉伸会让矮的一侧内部被空白填满，见 .create-columns 的注释）。 -->
        <div class="create-column">
          <div class="create-section">
            <h3>基础</h3>
            <input value={settings.name} onkeydown={preventEnterSubmit} oninput={(event) => { customRoomName = true; patch({ name: event.currentTarget.value }); }} placeholder="房间名" />
            <input value={settings.password || ""} onkeydown={preventEnterSubmit} oninput={(event) => patch({ password: event.currentTarget.value || undefined })} placeholder="房间密码，可不填" />
            <Toggle label="显示房间 Tag" value={settings.enableTags ?? false} onChange={(value) => patch({ enableTags: value })} />
            {#if settings.enableTags}
              <TagPicker options={config.roomTags} value={settings.tags || []} onChange={(tags) => patch({ tags })} />
            {/if}
          </div>
          {#if settings.gameId !== "coinflip"}
            <div class="create-section">
              <h3>玩法</h3>
              <div class="ranked-choice-grid">
                <button type="button" class={`ranked-choice-card ${!settings.enableRanked ? "active" : ""}`} onclick={() => patch({ enableRanked: false, enableExtremeRanked: false })}>
                  <span>🎮 普通局</span>
                  <small>不增加/减少排位积分，适合随意对战。</small>
                </button>
                {#each stakeTiersFor(config, settings.gameId) as stake (stake)}
                  <button type="button" class={`ranked-choice-card ${settings.enableRanked && settings.stake === stake ? "active" : ""}`} onclick={() => patch({ enableRanked: true, stake, enableExtremeRanked: Boolean(me.extremeModeEnabled) })}>
                    <span>{settings.gameId === "othello" || settings.gameId === "tictactoe" || settings.gameId === "gomoku" || settings.gameId === "jungle" || settings.gameId === "chess" ? "🏆 排位" : me.extremeModeEnabled ? "⚡ 极限排位" : "🏆 排位"} {stake}{settings.gameId === "othello" ? " 分/子" : " 分"}</span>
                    <small>
                      {#if settings.gameId === "othello"}每翻掉对方 1 子立即结算 {stake} 分，终局不重复结算。
                      {:else if settings.gameId === "liarsdice"}胜利 +{stake}，失败 -{stake}；其余参战玩家本局平，不计分。
                      {:else if me.extremeModeEnabled}只能创建极限排位房；非极限玩家无法进入。
                      {:else}胜利 +{stake}，失败 -{stake}；普通平局不扣分，平局双罚时双方 -{stake}。{/if}
                    </small>
                  </button>
                {/each}
              </div>
              {#if settings.gameId === "othello"}<p class="hint">黑白棋排位按实时翻子结算，可选 {stakeTiersFor(config, "othello").join("/")} 分/子；支持倍率和极限模式，但两者不能同时开启。</p>{/if}
              {#if settings.gameId !== "othello" && settings.gameId !== "liarsdice"}<p class="hint">{config.games.find((game) => game.id === settings.gameId)?.name ?? "本游戏"}排位按胜负固定分结算，可选 {stakeTiersFor(config, settings.gameId).join("/")} 分；支持倍率和极限模式。</p>{/if}
              {#if settings.gameId === "liarsdice"}<p class="hint">大话骰排位按胜负固定分结算，可选 {stakeTiersFor(config, "liarsdice").join("/")} 分；支持倍率和极限模式。</p>{/if}
              {#if settings.enableRanked && me.extremeModeEnabled}
                <div class="multiplier-box extreme-mode-box">
                  <div class="multiplier-head"><strong>⚡ 极限排位已开启</strong><span>禁用倍率</span></div>
                  <p class="hint">极限排位会按你的极限模式分段调整加减分；非极限玩家无法进入这个房间。</p>
                </div>
              {/if}
              {#if settings.enableRanked && !me.extremeModeEnabled}
                <p class="hint">你没有开启极限模式，因此只能创建普通排位房。</p>
              {/if}
              {#if settings.enableRanked}
                <div class="multiplier-box">
                  <div class="multiplier-head"><strong>倍率模式</strong><span>{settings.enableRankMultiplier ? `x${settings.rankMultiplier || 1}` : "未开启"}</span></div>
                  {#if me.extremeModeEnabled}
                    <p class="hint danger-hint">极限模式不能开启倍率房间，也不能进入倍率房；黑白棋极限排位会按每次翻子实时套用极限折扣。</p>
                  {:else if !me.rankMultiplierUnlocked}
                    <p class="hint">提交 200 排位积分后，本次服务器运行期间可创建 2倍 / 5倍 / 10倍排位房。</p>
                    <button type="button" class="soft-button" disabled={me.stats.rankedPoints < 200} onclick={unlockMultiplierMode}>提交 200 积分解锁</button>
                    {#if me.stats.rankedPoints < 200}<p class="hint danger-hint">你的排位积分不足 200，暂时不能解锁。</p>{/if}
                  {:else}
                    <div class="multiplier-choice-grid">
                      {#each [1, 2, 5, 10] as multiplier (multiplier)}
                        <button type="button" class={`ranked-choice-card ${rankMultiplierForSettings(settings) === multiplier ? "active" : ""}`} onclick={() => patch({ enableRankMultiplier: multiplier > 1, rankMultiplier: multiplier as 1 | 2 | 5 | 10 })}>
                          <span>{multiplier === 1 ? "普通倍率" : `x${multiplier} 倍房`}</span>
                          <small>{multiplier === 1 ? "按基础赌分结算。" : settings.gameId === "othello" ? `每翻 1 子按 ${settings.stake * multiplier} 分结算。` : `胜负按 ${settings.stake * multiplier} 分结算。`}</small>
                        </button>
                      {/each}
                    </div>
                    <p class="hint">当前：排位 {settings.stake}{settings.gameId === "othello" ? " 分/子" : " 分"} × {rankMultiplierForSettings(settings)} 倍 = {settings.gameId === "othello" ? `每翻 1 子 ${settings.stake * rankMultiplierForSettings(settings)} 分` : `胜负 ${settings.stake * rankMultiplierForSettings(settings)} 分`}。</p>
                  {/if}
                </div>
              {/if}
            </div>
          {/if}
        </div>
        <div class="create-column">
          <div class="create-section">
            <h3>惩罚</h3>
            <Select
              value={!settings.enablePunishment ? "none" : ((settings.punishmentSource === "system" ? "random" : settings.punishmentSource) || "random")}
              onChange={(value) => {
                if (value === "none") patch({ enablePunishment: false, enablePerPiecePunishment: false });
                else patch({ punishmentSource: value as RoomSettings["punishmentSource"], enablePunishment: true });
              }}
              options={
                settings.gameId === "coinflip"
                  ? [{ value: "random", label: "随机惩罚任务" }, { value: "series", label: "系列惩罚任务" }]
                  : [
                    { value: "none", label: "无惩罚" },
                    { value: "random", label: "随机惩罚任务" },
                    { value: "series", label: "系列惩罚任务" },
                    { value: "player", label: "自定义惩罚任务" }
                  ]
              }
            />
            {#if settings.gameId === "othello"}<p class="hint">黑白棋惩罚会在终局、认输、逃跑或断线判负后触发。</p>{/if}
            {#if settings.gameId === "tictactoe"}<p class="hint">井字棋惩罚会在终局或断线判负后触发。</p>{/if}
            {#if settings.gameId === "gomoku"}<p class="hint">五子棋惩罚会在终局、认输或断线判负后触发。</p>{/if}
            {#if settings.gameId === "jungle"}<p class="hint">斗兽棋惩罚会在终局、认输或断线判负后触发；开启每子惩罚时，被吃子也会立刻受罚（棋钟暂停，不计积分/白给）。</p>{/if}
            {#if settings.gameId === "chess"}<p class="hint">国际象棋惩罚会在终局、认输或断线判负后触发；开启每子惩罚时，被吃子也会立刻受罚（棋钟暂停，不计积分/白给）。</p>{/if}
            {#if settings.gameId === "liarsdice"}<p class="hint">大话骰惩罚仅对败者触发（叫点/开牌对决中的负方，或断线判负方）；其余参战玩家记平但不计分、不受罚。</p>{/if}
            {#if settings.gameId === "coinflip"}<p class="hint">猜硬币惩罚在每次猜错后立即触发，没有真人对手可以审核，提交证明即视为完成。</p>{/if}
            {#if settings.enablePunishment}
              {#if (settings.punishmentSource || "random") === "random" || settings.punishmentSource === "system"}
                <p class="hint">点击选择任务标签：蓝色为必须包含，红色为拒绝，不选则随机出现。惩罚难度会随游戏进度增加。</p>
                <div class="punishment-tag-tri-grid">
                  {#each config.punishmentTags || [] as tag (tag.id)}
                    <button type="button" class={`punishment-tag-tri ${tagTriState(tag.id)}`} onclick={() => cycleTagState(tag.id)}>
                      <span>{tag.name}</span>
                    </button>
                  {/each}
                </div>
                {#if (config.punishmentTags || []).length === 0}<p class="hint">后台还没有配置惩罚标签。</p>{/if}
              {:else if settings.punishmentSource === "series"}
                <p class="hint">系列进度按玩家分别推进；你在当前房间里的进度会保留，换房或房内更换系列后从头开始。</p>
                <input type="text" value={seriesSearch} oninput={(event) => (seriesSearch = event.currentTarget.value)} placeholder="搜索系列任务标题（留空显示全部）" style="margin-bottom: 8px" />
                <select value={settings.punishmentSeriesId || ""} onchange={(event) => patch({ punishmentSeriesId: event.currentTarget.value })} disabled={filteredSeries.length === 0}>
                  {#if filteredSeries.length === 0}<option value="">没有匹配的系列任务</option>{/if}
                  {#each filteredSeries as series (series.id)}
                    <option value={series.id}>{series.name}（{series.stepCount ?? 0} 步）</option>
                  {/each}
                </select>
                {#if (config.punishmentSeriesSummaries || []).length === 0}<p class="hint">后台还没有配置系列任务。</p>{/if}
              {:else}
                <p class="hint">本局结算后，由对手临时写惩罚任务；任务不会保存到后台配置。</p>
              {/if}
              {#if settings.gameId !== "liarsdice" && settings.gameId !== "coinflip"}
                <Toggle label="平局双罚" value={settings.tieDoublePunish} onChange={(value) => patch({ tieDoublePunish: value })} />
                {#if settings.enableRanked && settings.tieDoublePunish}<p class="hint">排位平局双罚开启时，平局双方都会扣 {settings.stake} 分。</p>{/if}
              {/if}
              {#if settings.gameId !== "coinflip"}
                <Toggle label="惩罚需对手确认" value={settings.requireOpponentConfirm} onChange={(value) => patch({ requireOpponentConfirm: value })} />
              {/if}
              <Toggle label="允许图片证明" value={settings.allowProofImage ?? true} onChange={(value) => patch({ allowProofImage: value })} />
              {#if settings.gameId === "chess" || settings.gameId === "jungle"}
                <Toggle label="每子惩罚" value={Boolean(settings.enablePerPiecePunishment)} onChange={(value) => patch({ enablePerPiecePunishment: value })} />
                <p class="hint">勾选后，对局中每次被吃子都会立刻受罚并暂停棋钟；积分和白给值值仍按每局结算。</p>
              {/if}
            {/if}
          </div>
        </div>
      </div>
    </div>
    <div class="modal-actions">
      <button type="button" onclick={onCancel}>取消</button>
      <button type="button" class="primary" disabled={settings.enablePunishment && settings.punishmentSource === "series" && !settings.punishmentSeriesId} onclick={create}>创建房间</button>
    </div>
  </section>
</div>
