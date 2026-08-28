<script module lang="ts">
  // 源：ui/AppViews.tsx:625-675（timedGameSettingKeys 常量随本组件一起挪过来，唯一用途在此）。
  import type { RoomSettings } from "../../shared/types";

  const timedGameSettingKeys = {
    othello: { move: "othelloMoveSeconds", game: "othelloGameMinutes", undo: "othelloUndoLimit" },
    gomoku: { move: "gomokuMoveSeconds", game: "gomokuGameMinutes", undo: "gomokuUndoLimit" },
    jungle: { move: "jungleMoveSeconds", game: "jungleGameMinutes", undo: "jungleUndoLimit" },
    chess: { move: "chessMoveSeconds", game: "chessGameMinutes", undo: "chessUndoLimit" }
  } as const;

  export type TimedGameID = keyof typeof timedGameSettingKeys;

  export function isTimedGame(gameId: RoomSettings["gameId"]): gameId is TimedGameID {
    return gameId in timedGameSettingKeys;
  }
</script>

<script lang="ts">
  import { gameMinutesOptions, moveSecondsOptions } from "../../lib/constants";

  let { gameId, settings, onPatch }: {
    gameId: TimedGameID;
    settings: RoomSettings;
    onPatch: (next: Partial<RoomSettings>) => void;
  } = $props();

  const keys = $derived(timedGameSettingKeys[gameId]);
</script>

<div class="game-timer-settings">
  <label>
    悔棋次数
    <select value={Number(settings[keys.undo]) || 0} onchange={(event) => onPatch({ [keys.undo]: Number(event.currentTarget.value) } as Partial<RoomSettings>)}>
      {#each [0, 1, 3, 10] as limit (limit)}
        <option value={limit}>{limit === 0 ? "禁止" : `${limit}次`}</option>
      {/each}
    </select>
  </label>
  <label>
    每子时长
    <select value={Number(settings[keys.move]) || 0} onchange={(event) => onPatch({ [keys.move]: Number(event.currentTarget.value) } as Partial<RoomSettings>)}>
      {#each moveSecondsOptions as value (value)}
        <option {value}>{value === 0 ? "不限" : `${value} 秒`}</option>
      {/each}
    </select>
  </label>
  <label>
    每局时长
    <select value={Number(settings[keys.game]) || 0} onchange={(event) => onPatch({ [keys.game]: Number(event.currentTarget.value) } as Partial<RoomSettings>)}>
      {#each gameMinutesOptions as value (value)}
        <option {value}>{value === 0 ? "不限" : `${value} 分钟`}</option>
      {/each}
    </select>
  </label>
</div>
