<script lang="ts">
  // 「房间标签」分区：房主自定义 Tag 词库 + 房间头部规则标签的文字与配色。
  // 源：ui/AdminViews.tsx:1155-1202（activeSection === "roomTags"）。
  import type { AppConfig, RoomInfoTagStyle } from "../../shared/types";
  import { defaultRoomInfoTagStyle, roomInfoTagOrder, roomInfoTagStyle } from "../../lib/roomInfoTags";
  import { styleString } from "../../lib/style";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import ColorInput from "./ColorInput.svelte";
  import TagListEditor from "./TagListEditor.svelte";
  import RoomTagList from "../shell/RoomTagList.svelte";
  import RoomInfoTagList from "../shell/RoomInfoTagList.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const previewInfoTags = $derived(
    roomInfoTagOrder.slice(0, 7).map((item) => {
      const style = draft.roomInfoTags?.[item.key] || defaultRoomInfoTagStyle(item.label);
      return { key: item.key, text: style.label, style };
    })
  );
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="房间标签" subtitle="玩家创建房间时，可以开启并选择这里配置好的标签。最多显示 5 个。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <RoomTagList tags={draft.roomTags.slice(0, 5)} />
    <p>点下面标签可以删除；在输入框里输入文字后回车可以添加。</p>
  </div>
  <TagListEditor label="房间 Tag 池" placeholder="输入房间 Tag 后回车" values={draft.roomTags} onChange={(roomTags) => adminStore.patch({ roomTags })} />
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="房间信息标签" subtitle="修改房间顶部规则标签的名字和颜色。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <RoomInfoTagList tags={previewInfoTags} />
    <p>这些标签会显示在房间信息卡里，部分也会显示在大厅房间卡上。</p>
  </div>
  <div class="room-info-tag-admin-grid">
    {#each roomInfoTagOrder as item (item.key)}
      {@const style = draft.roomInfoTags?.[item.key] || defaultRoomInfoTagStyle(item.label)}
      {@const nextTags = draft.roomInfoTags || {}}
      {@const update = (nextStyle: RoomInfoTagStyle) => adminStore.patch({ roomInfoTags: { ...nextTags, [item.key]: nextStyle } })}
      <div class="mini-card room-info-tag-admin-card">
        <div class="admin-card-title">
          <strong>{item.label}</strong>
          <small>{item.key}</small>
        </div>
        <span class="room-info-tag preview" style={styleString(roomInfoTagStyle(style))}>{style.label}</span>
        <label class="field-label"><span>显示名字</span><input value={style.label} oninput={(event) => update({ ...style, label: event.currentTarget.value })} /></label>
        <div class="color-grid">
          <ColorInput label="文字颜色" value={style.textColor} onChange={(textColor) => update({ ...style, textColor })} />
          <ColorInput label="背景颜色" value={style.backgroundColor} onChange={(backgroundColor) => update({ ...style, backgroundColor })} />
          <ColorInput label="边框颜色" value={style.borderColor} onChange={(borderColor) => update({ ...style, borderColor })} />
        </div>
      </div>
    {/each}
  </div>
</div>
