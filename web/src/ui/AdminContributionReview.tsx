import { useEffect, useMemo, useRef, useState } from "react";
import type { AppConfig, ContributionItem, ContributionStatus, PublicPlayer } from "../shared/types";
import { ask } from "../lib/rpc";
import { ContributionStatusChip, PlayerAvatar, PlayerBadge } from "./AppViews";
import { ContributeSeriesForm } from "./ContributeSeriesForm";
import { ContributionPreview, asContributionDraft, type StepPreview } from "./ContributionPreview";
import { StepEditor } from "./StepEditor";
import {
  contributionDraftTitle,
  contributionStatusLabel,
  contributionVoteTail,
  emptySeriesStep,
  emptyStepDraft,
  formatContributionDay,
  formatContributionListMeta,
  isValidOrder,
  stepHasFactionOverlap,
  voteRatio,
  type StepDraft,
} from "./contributeSeries";

const kindLabel: Record<string, string> = {
  task: "随机任务",
  series: "系列任务",
};

// 需要管理员做决定的状态：排在任意列表的最前面。其余（已通过/已驳回/已撤回）按更新时间倒序跟在后面。
const pendingLikeStatuses = new Set<ContributionStatus>(["pending"]);

function sortReviewQueue(items: ContributionItem[]): ContributionItem[] {
  const pendingGroup = items
    .filter((it) => pendingLikeStatuses.has(it.status))
    .sort((a, b) => a.updatedAt - b.updatedAt);
  const restGroup = items
    .filter((it) => !pendingLikeStatuses.has(it.status))
    .sort((a, b) => b.updatedAt - a.updatedAt);
  return [...pendingGroup, ...restGroup];
}

function toAdminStepDraft(raw: StepPreview | null | undefined, config: AppConfig): StepDraft {
  const factionIds = config.genderFactions.map((f) => f.id);
  const tagIds = (config.punishmentTags || []).map((t) => t.id);
  const base = emptyStepDraft(factionIds, true);
  if (!raw || typeof raw !== "object") return base;
  return {
    variants: Array.isArray(raw.variants) && raw.variants.length
      ? raw.variants.map((v) => ({
        text: typeof v?.text === "string" ? v.text : "",
        factionIds: Array.isArray(v?.factionIds) ? v.factionIds.filter((id) => factionIds.includes(id)) : [],
      }))
      : base.variants,
    inRandomPool: raw.inRandomPool !== false,
    order: typeof raw.order === "number" && raw.order > 0 ? raw.order : base.order,
    tagIds: Array.isArray(raw.tagIds) ? raw.tagIds.filter((id) => tagIds.includes(id)) : [],
    backgroundImage: typeof raw.backgroundImage === "string" ? raw.backgroundImage : "",
    backgroundOpacity: typeof raw.backgroundOpacity === "number" ? raw.backgroundOpacity : base.backgroundOpacity,
  };
}

export type ContributionReviewCounts = {
  pending: number;
};

type OverviewCounts = Record<string, number>;
type PoolStats = { randomTasks?: number; series?: number };
type PendingOverviewResponse = {
  counts: { task?: OverviewCounts; series?: OverviewCounts };
  poolStats?: PoolStats;
};

type JumpTarget = { status: ContributionStatus; nonce: number };

/** 随机任务 + 系列任务加总的待审条数，供顶部「待处理」选项卡摘要与父组件
    （AdminViews.tsx）的审核徽标共用同一份口径。 */
function reviewQueueCounts(res: PendingOverviewResponse): ContributionReviewCounts {
  const t = res.counts?.task || {};
  const s = res.counts?.series || {};
  return { pending: (t.pending || 0) + (s.pending || 0) };
}

export function AdminContributionReview({ config, password, onError, onCounts }: {
  config: AppConfig;
  password: string;
  onError: (message: string) => void;
  onCounts?: (counts: ContributionReviewCounts) => void;
}) {
  const [section, setSection] = useState<"pending" | "task" | "series">("pending");
  const [jumpTarget, setJumpTarget] = useState<{ kind: "task" | "series" } & JumpTarget | null>(null);
  // 总览数据提到这一层统一拉取：三个顶部选项卡各自的摘要数字（待处理条数、随机任务/系列任务
  // 使用中条数）不管当前停在哪个板块都要显示，不能只在「待处理」板块挂载时才有数据。
  const [overview, setOverview] = useState<PendingOverviewResponse | null>(null);
  const [overviewNonce, setOverviewNonce] = useState(0);
  const refreshOverview = () => setOverviewNonce((n) => n + 1);

  useEffect(() => {
    ask<PendingOverviewResponse>("admin:action", { action: "contributionPendingOverview", password }).then((res) => {
      setOverview(res);
      if (onCounts) onCounts(reviewQueueCounts(res));
    }).catch((e) => onError(e instanceof Error ? e.message : "加载失败"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [overviewNonce]);

  function jumpTo(kind: "task" | "series", status: ContributionStatus) {
    setSection(kind);
    setJumpTarget({ kind, status, nonce: Date.now() });
  }

  // 「待处理」= 随机任务 + 系列任务待审条数之和；「随机任务」「系列任务」=
  // 各自当前已批准（正在使用）的投稿数——系列任务的每一步不是独立投稿，天然不会被随机
  // 任务这边重复计入。
  const queueCounts = overview ? reviewQueueCounts(overview) : null;
  const pendingTotal = queueCounts ? queueCounts.pending : null;
  const taskApproved = overview?.counts?.task?.approved ?? null;
  const seriesApproved = overview?.counts?.series?.approved ?? null;

  return (
    <div className="admin-contribution contribute-panel">
      <div className="contribute-tabs" role="tablist">
        <button type="button" className={`small${section === "pending" ? " active" : ""}`} onClick={() => setSection("pending")}>
          <span><strong>待处理</strong></span>
          <em>{pendingTotal == null ? "…" : `${pendingTotal} 条`}</em>
        </button>
        <button type="button" className={`small${section === "task" ? " active" : ""}`} onClick={() => setSection("task")}>
          <span><strong>随机任务</strong></span>
          <em>{taskApproved == null ? "…" : `${taskApproved} 条`}</em>
        </button>
        <button type="button" className={`small${section === "series" ? " active" : ""}`} onClick={() => setSection("series")}>
          <span><strong>系列任务</strong></span>
          <em>{seriesApproved == null ? "…" : `${seriesApproved} 组`}</em>
        </button>
      </div>
      {section === "pending" ? (
        <PendingSection data={overview} onJump={jumpTo} />
      ) : (
        <ReviewKindPanel
          key={section}
          kind={section}
          config={config}
          password={password}
          onError={onError}
          onChanged={refreshOverview}
          jumpTarget={jumpTarget && jumpTarget.kind === section ? jumpTarget : null}
        />
      )}
    </div>
  );
}

const overviewStatusOrder: Array<{ status: ContributionStatus; label: string }> = [
  { status: "pending", label: "待审" },
  { status: "approved", label: "通过" },
  { status: "rejected", label: "驳回" },
  { status: "withdrawn", label: "撤回" },
];

function PendingSection({ data, onJump }: {
  data: PendingOverviewResponse | null;
  onJump: (kind: "task" | "series", status: ContributionStatus) => void;
}) {
  return (
    <div className="stack">
      <div className="admin-review-section">
        <h3>总览</h3>
        {data ? (
          <>
            <p className="hint">
              已发布随机任务 <strong>{data.poolStats?.randomTasks ?? 0}</strong> 条
              （含系列任务里同时进随机池的步骤） · 已发布系列任务 <strong>{data.poolStats?.series ?? 0}</strong> 组
            </p>
            <OverviewRow kind="task" label="随机任务" counts={data.counts?.task} onJump={onJump} />
            <OverviewRow kind="series" label="系列任务" counts={data.counts?.series} onJump={onJump} />
          </>
        ) : <p className="hint">加载中…</p>}
      </div>
    </div>
  );
}

function OverviewRow({ kind, label, counts, onJump }: {
  kind: "task" | "series";
  label: string;
  counts?: OverviewCounts;
  onJump: (kind: "task" | "series", status: ContributionStatus) => void;
}) {
  return (
    <div className="admin-overview-kind">
      <strong>{label}</strong>
      <div className="admin-overview-grid">
        {overviewStatusOrder.map(({ status, label: statLabel }) => {
          const n = counts?.[status] || 0;
          return (
            <button
              key={status}
              type="button"
              className="contribute-item admin-overview-stat"
              disabled={n === 0}
              onClick={() => onJump(kind, status)}
            >
              <strong>{n}</strong>
              <small>{statLabel}</small>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function ReviewKindPanel({ kind, config, password, onError, onChanged, jumpTarget }: {
  kind: "task" | "series";
  config: AppConfig;
  password: string;
  onError: (message: string) => void;
  onChanged: () => void;
  jumpTarget: ({ kind: "task" | "series" } & JumpTarget) | null;
}) {
  const [items, setItems] = useState<ContributionItem[]>([]);
  const [reloadNonce, setReloadNonce] = useState(0);

  async function reload() {
    try {
      const res = await ask<{ items?: ContributionItem[] } | ContributionItem[]>(
        "admin:action", { action: "contributionList", status: "", kind, password },
      );
      const list = Array.isArray(res) ? res : (res.items || []);
      setItems(sortReviewQueue(list));
    } catch (e) {
      onError(e instanceof Error ? e.message : "加载失败");
    }
  }

  useEffect(() => {
    reload().catch(() => undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, reloadNonce]);

  return (
    <SubmissionReviewPanel
      items={items}
      kind={kind}
      config={config}
      password={password}
      onError={onError}
      onChanged={() => { setReloadNonce((n) => n + 1); onChanged(); }}
      jumpTarget={jumpTarget}
      emptyText={`暂无${kindLabel[kind]}投稿`}
    />
  );
}

/** 列表 + 详情的共建审核核心组件：随机任务/系列任务 tab 都复用它，与玩家端「参与共建」
    （ContributeView.tsx）同一套 .contribute-layout 骨架，只是右侧换成预览+审核操作而非
    编辑表单。 */
function SubmissionReviewPanel({ items, kind, config, password, onError, onChanged, jumpTarget, emptyText }: {
  items: ContributionItem[];
  kind: "task" | "series";
  config: AppConfig;
  password: string;
  onError: (message: string) => void;
  onChanged: () => void;
  jumpTarget?: JumpTarget | null;
  emptyText: string;
}) {
  const [currentId, setCurrentId] = useState<string | null>(null);
  const [current, setCurrent] = useState<ContributionItem | null>(null);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [search, setSearch] = useState("");
  const [editing, setEditing] = useState(false);
  // 点击过一次「保存并立即发布/批准」但校验没过（难度为空/越界等）：标红对应输入框，
  // 与 ContributeSeriesForm 的 attempted 是同一套视觉语言。
  const [editAttempted, setEditAttempted] = useState(false);
  const [editTask, setEditTask] = useState<StepDraft>(() => emptyStepDraft(config.genderFactions.map((f) => f.id), true));
  const [editSeriesName, setEditSeriesName] = useState("");
  const [editSeriesTargets, setEditSeriesTargets] = useState<string[]>([]);
  const [editSeriesSteps, setEditSeriesSteps] = useState<StepDraft[]>([emptySeriesStep()]);
  const itemRefs = useRef<Record<string, HTMLButtonElement | null>>({});
  const consumedJumpNonce = useRef<number | null>(null);

  const draft = useMemo(() => asContributionDraft(current?.content), [current]);

  const filteredItems = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return items;
    return items.filter((it) => {
      const t = it.title || kindLabel[it.kind] || it.kind;
      return t.toLowerCase().includes(q) || (it.submitterName || "").toLowerCase().includes(q);
    });
  }, [items, search]);

  function factionLabel(id: string) {
    return config.genderFactions.find((f) => f.id === id)?.label || id;
  }
  function tagLabel(id: string) {
    return (config.punishmentTags || []).find((t) => t.id === id)?.name || id;
  }

  async function openItem(id: string) {
    try {
      const detail = await ask<ContributionItem>("admin:action", { action: "contributionGet", id, kind, password });
      setCurrentId(id);
      setCurrent(detail);
      setComment(detail.reviewComment || "");
      setEditing(false);
      setEditAttempted(false);
    } catch (e) {
      onError(e instanceof Error ? e.message : "加载失败");
    }
  }

  useEffect(() => {
    if (currentId && !items.some((it) => it.id === currentId)) {
      setCurrentId(null);
      setCurrent(null);
    }
  }, [items, currentId]);

  // 从「待处理·总览」跳转过来：items 加载完成前 jumpTarget 会先到，等 items 真正
  // 有内容后再定位对应条目，避免在列表还是空的那一瞬间误报"没有这个状态"。
  useEffect(() => {
    if (!jumpTarget || consumedJumpNonce.current === jumpTarget.nonce || items.length === 0) return;
    consumedJumpNonce.current = jumpTarget.nonce;
    const match = items.find((it) => it.status === jumpTarget.status);
    if (!match) {
      onError(`没有「${contributionStatusLabel[jumpTarget.status] || jumpTarget.status}」状态的${kindLabel[kind]}投稿`);
      return;
    }
    setSearch("");
    void openItem(match.id);
    requestAnimationFrame(() => itemRefs.current[match.id]?.scrollIntoView({ block: "center", behavior: "smooth" }));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [jumpTarget, items]);

  async function act(action: string, reviewedContent?: string) {
    if (!currentId) return;
    setBusy(true);
    try {
      await ask("admin:action", { action, id: currentId, kind, comment, password, ...(reviewedContent == null ? {} : { reviewedContent }) });
      setCurrentId(null);
      setCurrent(null);
      setComment("");
      setEditing(false);
      onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  function beginEditing() {
    if (!current || !draft) return;
    if (current.kind === "task") {
      setEditTask(toAdminStepDraft(draft, config));
    } else {
      const allFactionIds = config.genderFactions.map((f) => f.id);
      setEditSeriesName(draft.name || "");
      const targets = (draft.targetFactionIds || []).filter((id) => allFactionIds.includes(id));
      setEditSeriesTargets(targets.length ? targets : allFactionIds);
      setEditSeriesSteps((draft.steps || []).length
        ? (draft.steps || []).map((step) => toAdminStepDraft(step, config))
        : [emptySeriesStep(allFactionIds)]);
    }
    setEditAttempted(false);
    setEditing(true);
  }

  function publishEdited(next: unknown) {
    void act("contributionPublish", JSON.stringify(next));
  }

  async function saveComment() {
    if (!currentId) return;
    setBusy(true);
    try {
      await ask("admin:action", { action: "contributionUpdateComment", id: currentId, kind, comment, password });
      onError("批注已保存");
      onChanged();
    } catch (e) {
      onError(e instanceof Error ? e.message : "保存失败");
    } finally {
      setBusy(false);
    }
  }

  const title = current ? (current.title || contributionDraftTitle(draft) || kindLabel[current.kind]) : "";
  const voteTail = current
    ? contributionVoteTail({ kind: current.kind, status: current.status, likeRatio: voteRatio(current.likeCount, current.downCount), completionRate: current.completion?.rate })
    : "";

  return (
    <div className="contribute-layout">
      <div className="contribute-sidebar">
        <input
          className="contribute-list-search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="搜索标题 / 投稿者"
        />
        <div className="contribute-list">
          {filteredItems.map((item) => (
            <button
              key={item.id}
              ref={(el) => { itemRefs.current[item.id] = el; }}
              type="button"
              className={`contribute-item${currentId === item.id ? " active" : ""}`}
              onClick={() => void openItem(item.id)}
            >
              <strong>{item.title || kindLabel[item.kind] || item.kind}</strong>
              <small className="hint">
                <ContributionStatusChip status={item.status} />{" "}
                {formatContributionListMeta({ kind: item.kind, status: item.status, updatedAt: item.updatedAt, likeRatio: voteRatio(item.likeCount, item.downCount), completionRate: item.completion?.rate, short: true })}
              </small>
            </button>
          ))}
          {items.length === 0 ? <p className="hint">{emptyText}</p> : null}
          {items.length > 0 && filteredItems.length === 0 ? <p className="hint">没有匹配的投稿</p> : null}
        </div>
      </div>
      <div className="contribute-main">
        {current ? (
          <div className="stack">
            <div className="contribute-detail-head">
              <strong>{title}</strong>
              <small className="hint">
                <ContributionStatusChip status={current.status} />{" "}
                {formatContributionDay(current.updatedAt)} ·{" "}
                <SubmitterHoverName
                  playerId={current.submitterId}
                  label={current.anonymous ? `${current.submitterName}（匿名）` : current.submitterName}
                />
                {voteTail ? ` · ${voteTail}` : null}
              </small>
            </div>
            {editing && current.kind === "task" ? (
              <form className="stack" noValidate onSubmit={(event) => {
                event.preventDefault();
                if (stepHasFactionOverlap(editTask) || !isValidOrder(editTask.order)) { setEditAttempted(true); return; }
                publishEdited({ ...editTask, inRandomPool: true });
              }}>
                <StepEditor
                  config={config}
                  value={editTask}
                  onChange={setEditTask}
                  onError={onError}
                  showRandomPoolToggle={false}
                  showErrors={editAttempted}
                />
                <div className="row">
                  <button type="button" disabled={busy} onClick={() => setEditing(false)}>取消编辑</button>
                  <button type="submit" className="primary" disabled={busy || stepHasFactionOverlap(editTask) || !isValidOrder(editTask.order)}>保存并立即发布</button>
                </div>
              </form>
            ) : editing && current.kind === "series" ? (
              <div className="stack">
                <ContributeSeriesForm
                  config={config}
                  name={editSeriesName}
                  anonymous={current.anonymous}
                  targetFactionIds={editSeriesTargets}
                  steps={editSeriesSteps}
                  busy={busy}
                  canSaveDraft={false}
                  onName={setEditSeriesName}
                  onAnonymous={() => undefined}
                  onTargets={setEditSeriesTargets}
                  onSteps={setEditSeriesSteps}
                  onSaveDraft={() => undefined}
                  onSubmit={(next) => publishEdited(next)}
                  onError={onError}
                  showAnonymous={false}
                  showSaveDraft={false}
                  submitLabel="保存并批准"
                />
                <div className="row">
                  <button type="button" disabled={busy} onClick={() => setEditing(false)}>取消编辑</button>
                </div>
              </div>
            ) : draft ? (
              <ContributionPreview draft={draft} kind={current.kind} factionLabel={factionLabel} tagLabel={tagLabel} />
            ) : <p className="notice">内容解析失败。</p>}
            {(current.likeCount || current.downCount) ? (
              <p className="hint">获赞 {current.likeCount ?? 0} · 点踩 {current.downCount ?? 0} · 点赞率 {voteRatio(current.likeCount, current.downCount)}%</p>
            ) : null}

            {!editing && current.status === "pending" ? (
              <>
                <label className="field-label"><span>批注</span><textarea value={comment} onChange={(e) => setComment(e.target.value)} placeholder="驳回时给投稿者看的说明（选填）" /></label>
                <div className="row">
                  <button type="button" disabled={busy} onClick={beginEditing}>编辑</button>
                  <button type="button" className="primary" disabled={busy} onClick={() => void act("contributionPublish")}>批准</button>
                  <button type="button" className="danger-button" disabled={busy} onClick={() => void act("contributionReject")}>驳回</button>
                </div>
              </>
            ) : null}

            {!editing && current.status === "approved" ? (
              <div className="row">
                <button type="button" disabled={busy} onClick={beginEditing}>编辑</button>
                <button type="button" className="danger-button" disabled={busy} onClick={() => void act("contributionUnpublish")}>下架</button>
              </div>
            ) : null}

            {!editing && current.status === "rejected" ? (
              <>
                <label className="field-label"><span>批注</span><textarea value={comment} onChange={(e) => setComment(e.target.value)} placeholder="驳回理由" /></label>
                <div className="row">
                  <button type="button" disabled={busy} onClick={() => void saveComment()}>保存批注</button>
                  <button type="button" className="primary" disabled={busy} onClick={() => void act("contributionRevertReject")}>撤销驳回</button>
                </div>
              </>
            ) : null}

            {!editing && current.status === "withdrawn" ? (
              <>
                <label className="field-label"><span>批注</span><textarea value={comment} onChange={(e) => setComment(e.target.value)} placeholder="取消撤回时可以给投稿者留言（选填）" /></label>
                <div className="row">
                  <button type="button" className="primary" disabled={busy} onClick={() => void act("contributionRevertWithdraw")}>取消撤回</button>
                </div>
              </>
            ) : null}
          </div>
        ) : <p className="hint">选择一条投稿</p>}
      </div>
    </div>
  );
}

/** 投稿者昵称：悬停/聚焦时懒加载并展示玩家详情浮窗（头像+称号等），复用主宠关系力导向图
    悬停节点同一套 PlayerAvatar/PlayerBadge + 药丸浮窗语言（petbond-graph-popover）。
    即便投稿是匿名的，管理员审核时仍看得到真实身份——匿名只对其他玩家生效。 */
function SubmitterHoverName({ playerId, label }: { playerId: string; label: string }) {
  const [player, setPlayer] = useState<PublicPlayer | null>(null);
  const [open, setOpen] = useState(false);
  const loadedRef = useRef(false);

  async function ensureLoaded() {
    if (loadedRef.current || !playerId) return;
    loadedRef.current = true;
    try {
      const res = await ask<{ player: PublicPlayer }>("player:get", { playerId });
      setPlayer(res.player || null);
    } catch {
      setPlayer(null);
    }
  }

  return (
    <span
      className="contribute-submitter"
      tabIndex={playerId ? 0 : undefined}
      onMouseEnter={() => { setOpen(true); void ensureLoaded(); }}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => { setOpen(true); void ensureLoaded(); }}
      onBlur={() => setOpen(false)}
    >
      {label}
      {open && player ? (
        <span className="contribute-submitter-popover">
          <PlayerAvatar player={player} size={36} />
          <PlayerBadge player={player} compact />
        </span>
      ) : null}
    </span>
  );
}
