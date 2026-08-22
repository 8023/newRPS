import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("contribute view markup", () => {
  it("keeps withdraw, save, submit in that order inside the submit row", () => {
    const src = readFileSync(new URL("./ContributeView.tsx", import.meta.url), "utf8");
    const submitRow = src.indexOf("contribute-submit-row");
    const withdraw = src.indexOf("{withdrawButton}", submitRow);
    const save = src.indexOf(">保存<", withdraw);
    const submit = src.indexOf(">提交<", save);
    expect(submitRow).toBeGreaterThan(0);
    expect(withdraw).toBeGreaterThan(submitRow);
    expect(save).toBeGreaterThan(withdraw);
    expect(submit).toBeGreaterThan(save);
    // 系列表单的撤回按钮同样通过 extraSubmitActions 传给 ContributeSeriesForm，
    // 而不是单独浮在编辑器上方。
    expect(src).toContain("extraSubmitActions={withdrawButton}");
    expect(src).toContain("内容尚未保存");
    expect(src).toContain("contribute-unsaved-card");
    expect(src).toContain(">放弃<");
    expect(src).toContain("requestLeave(onBack)");
  });

  it("no longer offers a gender contribution tab (feature removed)", () => {
    const src = readFileSync(new URL("./ContributeView.tsx", import.meta.url), "utf8");
    expect(src).not.toContain('"gender"');
    expect(src).not.toContain("性别共建");
  });

  it("shows the selected title and a meta line led by the shared status chip", () => {
    const src = readFileSync(new URL("./ContributeView.tsx", import.meta.url), "utf8");
    expect(src).toContain("contribute-detail-head");
    expect(src).toContain("contributionDraftTitle(selectedDraft)");
    // 状态药丸徽标现在和后台共建审核复用同一个 ContributionStatusChip 组件，
    // 放在列表/详情第二行开头（如"待审批 · 26/08/20"），不再各写一份 <span className="status-chip">。
    expect(src).toContain("ContributionStatusChip");
    expect(src).not.toContain("status-chip");
  });
});

describe("series step toolbar markup", () => {
  it("puts move/delete/add on each step and drops the old add/delete buttons", () => {
    const src = readFileSync(new URL("./ContributeSeriesForm.tsx", import.meta.url), "utf8");
    expect(src).toContain(">上移<");
    expect(src).toContain(">下移<");
    expect(src).toContain(">删除<");
    expect(src).toContain(">添加<");
    expect(src).not.toContain("删除这步");
    expect(src).not.toContain("添加步骤");
    const extra = src.indexOf("{extraSubmitActions}");
    const save = src.indexOf("{saveDraftLabel}", extra);
    const submit = src.indexOf("{submitLabel}", save);
    expect(extra).toBeGreaterThan(0);
    expect(save).toBeGreaterThan(extra);
    expect(submit).toBeGreaterThan(save);
    expect(src).toContain('submitLabel = "提交审批"');
    // 玩家端「参与共建」通过 saveDraftLabel/submitLabel 覆盖成"保存"/"提交"；
    // extraSubmitActions 用来插入撤回按钮，两者都是可选 prop，不影响后台共建审核的默认调用。
    expect(src).toContain('saveDraftLabel = "保存草稿"');
    expect(src).toContain("extraSubmitActions?:");
  });
});

describe("step editor field order", () => {
  it("renders punishment tags above difficulty and cover", () => {
    const src = readFileSync(new URL("./StepEditor.tsx", import.meta.url), "utf8");
    const tags = src.indexOf("<legend>惩罚标签</legend>");
    const difficulty = src.indexOf("难度 1-99");
    const cover = src.indexOf("<span>封面图</span>");
    expect(tags).toBeGreaterThan(0);
    expect(difficulty).toBeGreaterThan(tags);
    expect(cover).toBeGreaterThan(difficulty);
    expect(src).toContain("选取文件");
    expect(src).toContain("cover-thumb");
    expect(src).not.toContain("任务封面预览");
  });
});
