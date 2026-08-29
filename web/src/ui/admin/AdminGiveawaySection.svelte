<script lang="ts">
  // 「白给模式」分区：大厅面板文案 + 普通/宠物/主人三种关系各自的投票次数与幅度。
  // 源：ui/AdminViews.tsx:985-1113（activeSection === "giveaway"）。
  import type { AppConfig } from "../../shared/types";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";
  import NumberField from "../shell/NumberField.svelte";

  const draft = $derived(adminStore.draft as AppConfig);
  function patchGiveaway(next: Partial<AppConfig["giveaway"]>) {
    adminStore.patch({ giveaway: { ...draft.giveaway, ...next } });
  }
</script>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="白给模式" subtitle="修改大厅白给自救板的标题、说明和输入提示。" />
  <div class="admin-preview-card">
    <span>预览</span>
    <strong>{draft.giveaway.panelTitle}</strong>
    <p>{draft.giveaway.panelDescription}</p>
  </div>
  <div class="config-row">
    <label class="field-label">
      <span>大厅面板标题</span>
      <input value={draft.giveaway.panelTitle} maxlength="24" oninput={(event) => patchGiveaway({ panelTitle: event.currentTarget.value })} placeholder="白给自救板" />
    </label>
    <label class="field-label">
      <span>提交框提示</span>
      <input value={draft.giveaway.submitPlaceholder} maxlength="60" oninput={(event) => patchGiveaway({ submitPlaceholder: event.currentTarget.value })} placeholder="写下你的自我惩罚宣言..." />
    </label>
  </div>
  <label class="field-label">
    <span>面板说明</span>
    <textarea value={draft.giveaway.panelDescription} maxlength="160" oninput={(event) => patchGiveaway({ panelDescription: event.currentTarget.value })} placeholder="提交一点自我惩罚宣言..."></textarea>
  </label>
  <label class="field-label">
    <span>空状态文案</span>
    <input value={draft.giveaway.emptyText} maxlength="60" oninput={(event) => patchGiveaway({ emptyText: event.currentTarget.value })} placeholder="还没有人在白给自救板上。" />
  </label>
  <div class="config-row">
    <label class="field-label">
      <span>主动白给增量 (%)</span>
      <NumberField min={0.1} max={100} step={0.1} value={draft.giveaway.activeBoostValue} onChange={(activeBoostValue) => patchGiveaway({ activeBoostValue })} />
    </label>
    <label class="field-label">
      <span>胜利扣减白给值 (%)</span>
      <NumberField min={0.1} max={100} step={0.1} value={draft.giveaway.winPenaltyValue} onChange={(winPenaltyValue) => patchGiveaway({ winPenaltyValue })} />
    </label>
  </div>
  <p class="hint">胜利扣减仅对已开启白给模式的胜方生效（含断线判负）。</p>
</div>

<div class="config-section admin-section-card">
  <AdminSectionHeader title="自救版设置" subtitle="分别设置不同认主认宠关系下的投票次数与白给值变化。" />
  <div class="admin-settings-groups">
    <section class="admin-settings-group" aria-labelledby="giveaway-regular-votes-title">
      <div class="admin-card-title">
        <strong id="giveaway-regular-votes-title">普通用户</strong>
        <small>投票者与被投票者之间没有认主认宠关系时使用</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>点赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.likeVoteLimitPerHour} onChange={(likeVoteLimitPerHour) => patchGiveaway({ likeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>点赞降低值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.likeVoteValue} onChange={(likeVoteValue) => patchGiveaway({ likeVoteValue })} />
        </label>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>倒赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.dislikeVoteLimitPerHour} onChange={(dislikeVoteLimitPerHour) => patchGiveaway({ dislikeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>倒赞增加值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.dislikeVoteValue} onChange={(dislikeVoteValue) => patchGiveaway({ dislikeVoteValue })} />
        </label>
      </div>
    </section>

    <section class="admin-settings-group" aria-labelledby="giveaway-pet-votes-title">
      <div class="admin-card-title">
        <strong id="giveaway-pet-votes-title">对自己宠物</strong>
        <small>投票者是被投票者的直系主人时使用</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>点赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.petLikeVoteLimitPerHour} onChange={(petLikeVoteLimitPerHour) => patchGiveaway({ petLikeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>点赞降低值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.petLikeVoteValue} onChange={(petLikeVoteValue) => patchGiveaway({ petLikeVoteValue })} />
        </label>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>倒赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.petDislikeVoteLimitPerHour} onChange={(petDislikeVoteLimitPerHour) => patchGiveaway({ petDislikeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>倒赞增加值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.petDislikeVoteValue} onChange={(petDislikeVoteValue) => patchGiveaway({ petDislikeVoteValue })} />
        </label>
      </div>
    </section>

    <section class="admin-settings-group" aria-labelledby="giveaway-master-votes-title">
      <div class="admin-card-title">
        <strong id="giveaway-master-votes-title">对自己主人</strong>
        <small>投票者是被投票者的直系宠物时使用</small>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>点赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.masterLikeVoteLimitPerHour} onChange={(masterLikeVoteLimitPerHour) => patchGiveaway({ masterLikeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>点赞降低值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.masterLikeVoteValue} onChange={(masterLikeVoteValue) => patchGiveaway({ masterLikeVoteValue })} />
        </label>
      </div>
      <div class="config-row">
        <label class="field-label">
          <span>倒赞每小时次数上限</span>
          <NumberField min={1} step={1} value={draft.giveaway.masterDislikeVoteLimitPerHour} onChange={(masterDislikeVoteLimitPerHour) => patchGiveaway({ masterDislikeVoteLimitPerHour })} />
        </label>
        <label class="field-label">
          <span>倒赞增加值 (%)</span>
          <NumberField min={0.01} max={100} step={0.01} value={draft.giveaway.masterDislikeVoteValue} onChange={(masterDislikeVoteValue) => patchGiveaway({ masterDislikeVoteValue })} />
        </label>
      </div>
    </section>
  </div>
</div>
