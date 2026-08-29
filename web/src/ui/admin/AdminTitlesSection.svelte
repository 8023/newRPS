<script lang="ts">
  // 「排位与称号」分区：排位分显示上下限、各游戏排位分值、称号池、称号标签配色。
  // 源：ui/AdminViews.tsx:654-786（activeSection === "titles"）。
  import type { AppConfig, RoomInfoTagStyle } from "../../shared/types";
  import { withRankedScoreDefaults } from "../../lib/normalize";
  import { defaultRoomInfoTagStyle, titleTagStyleOrder } from "../../lib/roomInfoTags";
  import { titleStyle } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import { nextAdminId, taskGroupsFromFactions } from "../../lib/adminHelpers";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import ColorInput from "./ColorInput.svelte";
  import TagListEditor from "./TagListEditor.svelte";
  import NumberField from "../shell/NumberField.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  const filteredTitles = $derived(
    draft.titles.filter((segment) => {
      const keyword = adminStore.titleSearch.trim().toLowerCase();
      if (!keyword) return true;
      return `${segment.id} ${segment.minPercent} ${segment.maxPercent} ${segment.names.join(" ")}`.toLowerCase().includes(keyword);
    })
  );
  const selectedIndex = $derived(Math.max(0, draft.titles.findIndex((segment) => segment.id === adminStore.activeTitleId)));
  const segment = $derived(draft.titles[selectedIndex]);
  const taskGroups = $derived(taskGroupsFromFactions(draft.genderFactions));
  const rankedScore = $derived(withRankedScoreDefaults(draft.rankedScore));

  function patchRankedScore(next: Partial<AppConfig["rankedScore"]>) {
    adminStore.patch({ rankedScore: withRankedScoreDefaults({ ...rankedScore, ...next }) });
  }

  function addTitleSegment() {
    const nextId = nextAdminId("title", draft.titles.map((item) => item.id));
    adminStore.activeTitleId = nextId;
    adminStore.patch({ titles: [...draft.titles, { id: nextId, minPercent: 0, maxPercent: 0, names: ["新称号"], factionNames: Object.fromEntries(taskGroups.map((group) => [group.id, ["新称号"]])) }] });
  }
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="排位分显示" subtitle="控制排行榜/个人资料等展示时的封顶值，及每日衰减比例。数据库中的存储值无限制。" />
  <div class="config-row">
    <label class="field-label"><span>展示上限</span><NumberField min={1} value={rankedScore.max} onChange={(max) => patchRankedScore({ max })} /></label>
    <label class="field-label"><span>展示下限</span><NumberField max={-1} value={rankedScore.min} onChange={(min) => patchRankedScore({ min })} /></label>
    <label class="field-label"><span>名争展示下限</span><NumberField value={rankedScore.nameWarMin} onChange={(nameWarMin) => patchRankedScore({ nameWarMin })} /></label>
    <label class="field-label"><span>每日衰减比例</span><NumberField min={0.01} max={1} step={0.01} value={rankedScore.dailyDecayRatio} onChange={(dailyDecayRatio) => patchRankedScore({ dailyDecayRatio })} /></label>
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="游戏排位分值" subtitle="各游戏可选的积分档位。第 1 个为默认，最多 4 个，逗号分隔。" />
  <div class="config-row">
    {#each draft.games || [] as game, index (game.id)}
      <label class="field-label">
        <span>{game.id === "othello" ? `${game.name}（每子）` : `${game.name}（每局）`}</span>
        <input
          value={(game.stakes ?? []).join(",")}
          placeholder="如 5,10,20"
          oninput={(event) => {
            const stakes = event.currentTarget.value.split(/[,，\s]+/).map((part) => Number(part)).filter((n) => Number.isInteger(n) && n >= 1);
            const games = [...(draft.games || [])];
            games[index] = { ...game, stakes };
            adminStore.patch({ games });
          }}
        />
      </label>
    {/each}
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="称号池" subtitle="按排位分相对展示上下限的百分比分段后，按玩家阵营所属的生理性别从对应称号池随机装备默认称号。" />
  <div class="punishment-manager title-manager">
    <aside class="punishment-index-panel">
      <input value={adminStore.titleSearch} oninput={(event) => (adminStore.titleSearch = event.currentTarget.value)} placeholder="搜索段位 ID / 百分比 / 称号" />
      <div class="punishment-index-list">
        {#each filteredTitles as item (item.id)}
          <button class={item.id === segment?.id ? "active" : ""} onclick={() => (adminStore.activeTitleId = item.id)}>
            <span>{item.id} · {item.minPercent}% ~ {item.maxPercent}%</span>
            <small>通用 {item.names.length} 个 · {taskGroups.length} 个分组专属池</small>
          </button>
        {/each}
        {#if filteredTitles.length === 0}<p class="empty">没有匹配的段位</p>{/if}
      </div>
      <button onclick={addTitleSegment}>添加段位</button>
    </aside>
    {#if segment}
      <div class="mini-card punishment-detail-panel">
        <div class="admin-card-title">
          <strong>{segment.id} · {segment.minPercent}% ~ {segment.maxPercent}%</strong>
          <small>{selectedIndex + 1} / {draft.titles.length} · 通用 {segment.names.length} 个 · 相对展示上下限的百分比</small>
        </div>
        <div class="config-row compact">
          <label class="field-label"><span>段位 ID（自动生成，一般不用改）</span><input value={segment.id} oninput={(event) => { adminStore.activeTitleId = event.currentTarget.value; adminStore.patch({ titles: draft.titles.map((item, i) => i === selectedIndex ? { ...item, id: event.currentTarget.value } : item) }); }} /></label>
          <label class="field-label"><span>最低百分比（-100～100）</span><NumberField min={-100} max={100} step={0.01} value={segment.minPercent} onChange={(minPercent) => adminStore.patch({ titles: draft.titles.map((item, i) => i === selectedIndex ? { ...item, minPercent } : item) })} /></label>
          <label class="field-label"><span>最高百分比（-100～100）</span><NumberField min={-100} max={100} step={0.01} value={segment.maxPercent} onChange={(maxPercent) => adminStore.patch({ titles: draft.titles.map((item, i) => i === selectedIndex ? { ...item, maxPercent } : item) })} /></label>
        </div>
        <TagListEditor
          label="通用称号（专属为空时兜底）"
          placeholder="输入称号后回车"
          values={segment.names}
          onChange={(names) => adminStore.patch({ titles: draft.titles.map((item, i) => i === selectedIndex ? { ...item, names } : item) })}
        />
        <div class="title-faction-grid">
          {#each taskGroups as group (group.id)}
            <TagListEditor
              label={`${group.label}专属称号（${group.memberLabel}）`}
              placeholder={`输入${group.label}称号后回车`}
              values={segment.factionNames?.[group.id] || []}
              onChange={(names) => adminStore.patch({ titles: draft.titles.map((item, i) => i === selectedIndex ? { ...item, factionNames: { ...(item.factionNames || {}), [group.id]: names } } : item) })}
            />
          {/each}
        </div>
      </div>
    {/if}
  </div>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="称号标签配色" subtitle="称号标签的底色按该称号的赋予来源区分" />
  <div class="admin-preview-card">
    <span>预览</span>
    <div class="title-tag-style-preview">
      {#each titleTagStyleOrder as item (item.key)}
        {@const style = draft.titleTagStyles?.[item.key] || defaultRoomInfoTagStyle(item.label)}
        <span class="title-chip" style={styleString(titleStyle(style))}>{style.label}</span>
      {/each}
    </div>
  </div>
  <div class="room-info-tag-admin-grid">
    {#each titleTagStyleOrder as item (item.key)}
      {@const style = draft.titleTagStyles?.[item.key] || defaultRoomInfoTagStyle(item.label)}
      {@const nextStyles = draft.titleTagStyles || {}}
      {@const update = (nextStyle: RoomInfoTagStyle) => adminStore.patch({ titleTagStyles: { ...nextStyles, [item.key]: nextStyle } })}
      <div class="mini-card room-info-tag-admin-card">
        <div class="admin-card-title">
          <strong>{item.label}</strong>
          <small>{item.key}</small>
        </div>
        <span class="title-chip preview" style={styleString(titleStyle(style))}>{style.label}</span>
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
