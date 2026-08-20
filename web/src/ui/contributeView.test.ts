import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("contribute view markup", () => {
  it("keeps save-draft to the left of submit", () => {
    const src = readFileSync(new URL("./ContributeView.tsx", import.meta.url), "utf8");
    const save = src.indexOf(">保存草稿<");
    const submit = src.indexOf(">提交审批<");
    expect(save).toBeGreaterThan(0);
    expect(submit).toBeGreaterThan(save);
    expect(src).toContain("内容尚未保存");
    expect(src).toContain("contribute-unsaved-card");
    expect(src).toContain(">放弃<");
    expect(src).toContain("requestLeave(onBack)");
  });

  it("puts gender name and faction on one split row", () => {
    const src = readFileSync(new URL("./ContributeView.tsx", import.meta.url), "utf8");
    const formStart = src.indexOf('tab === "gender" &&');
    const gender = src.indexOf("性别名称", formStart);
    const faction = src.indexOf("归属阵营", formStart);
    const split = src.lastIndexOf("contribute-split-row", faction);
    expect(formStart).toBeGreaterThan(0);
    expect(gender).toBeGreaterThan(formStart);
    expect(faction).toBeGreaterThan(gender);
    expect(split).toBeGreaterThan(formStart);
    expect(split).toBeLessThan(gender);
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
    const save = src.indexOf(">保存草稿<");
    const submit = src.indexOf("{submitLabel}", save);
    expect(save).toBeGreaterThan(0);
    expect(submit).toBeGreaterThan(save);
    expect(src).toContain('submitLabel = "提交审批"');
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
