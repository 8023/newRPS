<script lang="ts">
  // 源：ui/AdminViews.tsx:2005-2014
  import type { RoomNamePool } from "../../shared/types";
  import TagListEditor from "./TagListEditor.svelte";

  let { title, pool, onChange }: { title: string; pool: RoomNamePool; onChange: (pool: RoomNamePool) => void } = $props();
</script>

<div class="room-name-pool-editor">
  <b>{title}</b>
  <TagListEditor label="形容词（可为空）" placeholder="输入形容词后回车" values={pool.adjectives} onChange={(adjectives) => onChange({ ...pool, adjectives })} />
  <TagListEditor label="名词/动词" placeholder="输入名词或动词后回车" values={pool.subjects} onChange={(subjects) => onChange({ ...pool, subjects })} />
  <TagListEditor label="房间词" placeholder="输入房间词后回车" values={pool.roomWords} onChange={(roomWords) => onChange({ ...pool, roomWords })} />
</div>
