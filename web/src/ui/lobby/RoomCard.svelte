<script lang="ts">
  // 源：ui/AppViews.tsx:564-582（Lobby 房间列表 map 内联的卡片，抽成独立组件）。
  import type { AppConfig, LobbySnapshot } from "../../shared/types";
  import { lobbyRoomInfoTags } from "../../lib/roomInfoTags";
  import { roomStatusText } from "../../lib/roomInfoTags";
  import { styleString } from "../../lib/style";
  import ExtremeRankedBadge from "../shell/ExtremeRankedBadge.svelte";
  import RankMultiplierBadge from "../shell/RankMultiplierBadge.svelte";
  import RoomTagList from "../shell/RoomTagList.svelte";
  import RoomInfoTagList from "../shell/RoomInfoTagList.svelte";
  import RoomVersusLine from "./RoomVersusLine.svelte";

  let {
    config, room, password, onPasswordChange, onJoin
  }: {
    config: AppConfig;
    room: LobbySnapshot["rooms"][number];
    password: string;
    onPasswordChange: (value: string) => void;
    onJoin: () => void;
  } = $props();
</script>

<div
  class={`room-card ${room.roomBackgroundImage ? "has-room-card-background" : ""}`}
  style={room.roomBackgroundImage ? styleString({ "--room-card-bg": `url(${room.roomBackgroundImage})` }) : undefined}
>
  <div>
    <h3>{room.name} <ExtremeRankedBadge enabled={room.enableExtremeRanked} /> <RankMultiplierBadge multiplier={room.rankMultiplier} /></h3>
    {#if room.tags?.length}<RoomTagList tags={room.tags} />{/if}
    <RoomVersusLine {room} />
    <p>{roomStatusText(room.status)} · {room.gameId === "liarsdice" ? `${room.players} 人参战` : room.gameId === "coinflip" ? `${room.players}/1 参战席` : `${room.players}/2 战斗席`} · {room.spectators} 观战</p>
    <RoomInfoTagList tags={lobbyRoomInfoTags(config, room)} />
  </div>
  <div class="join-box">
    {#if room.hasPassword}
      <input placeholder="房间密码" value={password} oninput={(event) => onPasswordChange(event.currentTarget.value)} />
    {/if}
    <button onclick={onJoin}>加入</button>
  </div>
</div>
