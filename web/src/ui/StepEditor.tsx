import { useEffect, useRef, useState, type ReactNode } from "react";
import type { AppConfig } from "../shared/types";
import { tokenKey } from "../lib/constants";
import { prepareProofImageForUpload } from "../lib/proofImage";
import { OptionalNumberField } from "./NumberField";
import {
  DIFFICULTY_GUIDE,
  MAX_STEP_VARIANTS,
  TAG_GUIDE,
  isValidOrder,
  overlappingFactionIds,
  toggleId,
  type StepDraft,
} from "./contributeSeries";

export function StepEditor({
  config,
  value,
  onChange,
  onError,
  showRandomPoolToggle,
  rightOfPreview,
  stepActions,
  showErrors,
}: {
  config: AppConfig;
  value: StepDraft;
  onChange: (next: StepDraft | ((prev: StepDraft) => StepDraft)) => void;
  onError: (message: string) => void;
  showRandomPoolToggle: boolean;
  /** 与「封面图预览」同一行、分布在左侧的附加控件（如「匿名贡献」）；仅单份编辑场景
      （随机任务）传入，系列每一步各自没有独立的匿名开关，不传。 */
  rightOfPreview?: ReactNode;
  /** 系列每一步的上移/下移/删除/添加；随机任务不传。传入且这一步要进随机池时启用两栏
      布局：左栏惩罚标签，右栏提示语/难度/封面图 + 这四个按钮；未勾选「同时发布到随机
      任务」时退化为单栏，封面图与这四个按钮合并一行（见 stepActions 传入分支）。 */
  stepActions?: ReactNode;
  /** 点击过一次「提交审批」但校验没通过：把还没填好的文案输入框标红，
      与阵营选重复时的 .conflict 标红是同一套视觉语言。不传时按 false 处理。 */
  showErrors?: boolean;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [uploadedPreview, setUploadedPreview] = useState<{ path: string; objectURL: string } | null>(null);
  const factionIds = config.genderFactions.map((f) => f.id);
  const maxVariants = Math.min(MAX_STEP_VARIANTS, Math.max(1, factionIds.length || MAX_STEP_VARIANTS));
  const showPoolFields = !showRandomPoolToggle || value.inRandomPool;
  const overlapFactions = new Set(overlappingFactionIds(value.variants));

  useEffect(() => () => {
    if (uploadedPreview) URL.revokeObjectURL(uploadedPreview.objectURL);
  }, [uploadedPreview]);

  // 一律用函数式更新（读最新的 prev，而不是闭包捕获的 value）：uploadImage 是异步的，
  // 上传期间用户可能已经编辑/删除了别的步骤，patch 若直接拿渲染时闭包的 value 展开，
  // 上传完成后回填就会用过期快照覆盖掉这段时间内的其它修改（见 CHANGELOG 相关修复）。
  function patch(next: Partial<StepDraft>) {
    onChange((prev) => ({ ...prev, ...next }));
  }

  function patchVariant(i: number, next: Partial<StepDraft["variants"][number]>) {
    patch({ variants: value.variants.map((item, idx) => idx === i ? { ...item, ...next } : item) });
  }

  async function uploadImage(file: File) {
    const prepared = await prepareProofImageForUpload(file);
    const body = new FormData();
    body.append("image", prepared);
    body.append("token", localStorage.getItem(tokenKey) || "");
    const res = await fetch("/api/contribution-image", { method: "POST", body });
    const data = await res.json() as { imageUrl?: string; message?: string };
    if (!res.ok || !data.imageUrl) throw new Error(data.message || "上传失败");
    // 新上传的共建图在草稿真正保存、background_image 落库前按服务端规则不可公开访问，
    // 直接拿远端 URL 做 <img> 会先得到 404。用已压缩文件的本地 object URL 预览；保存
    // 后重新进入详情时再自然改用远端 URL，服务端访问控制无需为预览开口子。
    setUploadedPreview({ path: data.imageUrl, objectURL: URL.createObjectURL(prepared) });
    patch({ backgroundImage: data.imageUrl });
  }

  const tagsFieldset = (
    <fieldset className="checkbox-fieldset">
      <legend>惩罚标签</legend>
      <p className="hint">{TAG_GUIDE}</p>
      <div className="checkbox-pill-row">
        {(config.punishmentTags || []).map((tag) => (
          <button
            key={tag.id}
            type="button"
            className={`checkbox-pill${value.tagIds.includes(tag.id) ? " active" : ""}`}
            onClick={() => patch({ tagIds: toggleId(value.tagIds, tag.id) })}
          >
            {tag.name}
          </button>
        ))}
      </div>
    </fieldset>
  );

  const difficultyField = (
    <label className="field-label">
      <span>难度 1-99</span>
      <OptionalNumberField
        min={1}
        max={99}
        value={value.order}
        onChange={(order) => patch({ order })}
        invalid={showErrors && !isValidOrder(value.order)}
      />
    </label>
  );

  const coverPicker = (
    <label className="field-label">
      <span>封面图</span>
      <div className="cover-picker">
        <button type="button" onClick={() => fileInputRef.current?.click()}>选取文件</button>
        {value.backgroundImage ? (
          <>
            <button type="button" onClick={() => { setUploadedPreview(null); patch({ backgroundImage: "" }); }}>移除封面</button>
            <img src={uploadedPreview?.path === value.backgroundImage ? uploadedPreview.objectURL : value.backgroundImage} alt="" className="cover-thumb" />
          </>
        ) : null}
        <input
          ref={fileInputRef}
          className="cover-picker-input"
          type="file"
          accept="image/*,.heic,.heif"
          onChange={(e) => {
            const file = e.target.files?.[0];
            e.target.value = "";
            if (file) uploadImage(file).catch((err: unknown) => onError(err instanceof Error ? err.message : "上传失败"));
          }}
        />
      </div>
    </label>
  );

  return (
    <div className="stack step-editor">
      {value.variants.map((variant, i) => (
        <fieldset key={i} className="step-variant">
          <legend>文案{value.variants.length > 1 ? ` ${i + 1}` : ""}</legend>
          <label>
            <textarea
              className={showErrors && !variant.text.trim() ? "field-invalid" : undefined}
              value={variant.text}
              onChange={(e) => patchVariant(i, { text: e.target.value })}
              required
              placeholder="请输入惩罚任务文案，可使用 {loser} 和 {winner} （含花括号）替代输家和赢家的昵称。"
            />
          </label>
          <fieldset className="checkbox-fieldset">
            <legend>适用阵营</legend>
            <div className="checkbox-pill-row">
              {config.genderFactions.map((f) => {
                const selected = variant.factionIds.includes(f.id);
                const conflict = selected && overlapFactions.has(f.id);
                return (
                  <button
                    key={f.id}
                    type="button"
                    className={`checkbox-pill${selected ? " active" : ""}${conflict ? " conflict" : ""}`}
                    onClick={() => patchVariant(i, { factionIds: toggleId(variant.factionIds, f.id) })}
                  >
                    {f.label}
                  </button>
                );
              })}
            </div>
          </fieldset>
          {value.variants.length > 1 ? (
            <button type="button" className="small" onClick={() => patch({ variants: value.variants.filter((_, idx) => idx !== i) })}>删除这份</button>
          ) : null}
        </fieldset>
      ))}
      {overlapFactions.size > 0 ? <p className="notice">多份文案不能勾同一个阵营，点红色的再取消。</p> : null}
      <button
        type="button"
        className="small"
        disabled={value.variants.length >= maxVariants}
        onClick={() => patch({ variants: [...value.variants, { text: "", factionIds: [] }] })}
      >
        + 添加特定阵营文案
      </button>
      {showRandomPoolToggle ? (
        <label className="checkbox-inline">
          <input
            type="checkbox"
            checked={value.inRandomPool}
            onChange={(e) => patch({ inRandomPool: e.target.checked })}
          />
          同时发布到随机任务
        </label>
      ) : null}
      {stepActions ? (
        showPoolFields ? (
          // 系列每一步、且这一步要进随机池：两栏布局——左栏惩罚标签，右栏是提示语 +
          // 难度/封面图（合并一行，各占右栏一半宽度）+ 紧跟在下面的上移/下移/删除/添加。
          <div className="step-columns">
            <div className="step-column">
              {tagsFieldset}
            </div>
            <div className="step-column">
              <p className="hint">{DIFFICULTY_GUIDE}</p>
              <div className="contribute-split-row">
                {difficultyField}
                {coverPicker}
              </div>
              <div className="series-step-actions">{stepActions}</div>
            </div>
          </div>
        ) : (
          // 未勾选「同时发布到随机任务」：没有标签/难度可放，退化成单栏——封面图选择器占
          // 左半，上移/下移/删除/添加四个按钮占右半（各占整行八分之一），且都对齐到「选取
          // 文件」按钮所在的那一行，而不是上面「封面图」三个字那一行（.contribute-split-row
          // 的 align-items:end 保证这一点，与「难度/封面图」合并一行是同一套写法）。
          <div className="contribute-split-row">
            {coverPicker}
            <div className="series-step-actions">{stepActions}</div>
          </div>
        )
      ) : (
        <>
          {showPoolFields ? tagsFieldset : null}
          {showPoolFields ? <p className="hint">{DIFFICULTY_GUIDE}</p> : null}
          <div className="contribute-split-row">
            {showPoolFields ? difficultyField : null}
            {coverPicker}
          </div>
        </>
      )}
      {rightOfPreview}
    </div>
  );
}
