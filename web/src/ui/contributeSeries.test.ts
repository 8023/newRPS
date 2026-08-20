import { describe, expect, it } from "vitest";
import {
  MAX_SERIES_STEPS,
  buildSeriesContent,
  contributionDraftTitle,
  contributionHasPublishedVersion,
  contributionStatusLabel,
  contributionVoteTail,
  effectiveMaxSeriesSteps,
  effectiveMinSeriesSteps,
  emptySeriesStep,
  formatContributionDay,
  formatContributionListMeta,
  formatContributionReviewSubtitle,
  insertStepAfter,
  missingTargetFactions,
  moveStep,
  overlappingFactionIds,
  removeStep,
  seriesCoverageGaps,
  seriesDraftHasContent,
  stepHasContent,
} from "./contributeSeries";

describe("buildSeriesContent", () => {
  it("emits the new step/variant shape without refs or names", () => {
    const first = emptySeriesStep(["f1", "f2"]);
    first.variants = [{ text: "第一步", factionIds: ["f1", "f2"] }];
    first.order = 10;
    const second = emptySeriesStep(["f1"]);
    second.variants = [{ text: "第二步", factionIds: ["f1"] }];
    second.order = 20;
    const content = buildSeriesContent("共建系列", ["f1"], [first, second]);
    expect(content.steps).toHaveLength(2);
    expect(content.targetFactionIds).toEqual(["f1"]);
    expect(content.steps[0]).not.toHaveProperty("name");
    expect(content.steps[1]?.variants).toEqual([{ text: "第二步", factionIds: ["f1"] }]);
    expect("tasks" in content).toBe(false);
  });

  it("keeps the documented step ceiling and default min", () => {
    expect(MAX_SERIES_STEPS).toBe(1000);
    expect(effectiveMinSeriesSteps(0)).toBe(10);
    expect(effectiveMinSeriesSteps(12)).toBe(12);
    expect(effectiveMinSeriesSteps(5000)).toBe(1000);
  });

  it("lets configured max steps take effect within the defensive hard ceiling", () => {
    expect(effectiveMaxSeriesSteps(0)).toBe(20);
    expect(effectiveMaxSeriesSteps(undefined)).toBe(20);
    expect(effectiveMaxSeriesSteps(8)).toBe(8);
    expect(effectiveMaxSeriesSteps(500)).toBe(500);
    expect(effectiveMaxSeriesSteps(5000)).toBe(1000);
  });
});

describe("variant faction overlap", () => {
  it("lists factions selected by more than one variant", () => {
    expect(overlappingFactionIds([
      { text: "甲", factionIds: ["f1", "f2"] },
      { text: "乙", factionIds: ["f2"] },
    ])).toEqual(["f2"]);
    expect(overlappingFactionIds([
      { text: "甲", factionIds: ["f1"] },
    ])).toEqual([]);
  });
});

describe("series coverage", () => {
  it("lists missing target factions per step", () => {
    const step = emptySeriesStep(["f1"]);
    step.variants = [{ text: "只给甲", factionIds: ["f1"] }];
    expect(missingTargetFactions(step, ["f1", "f2"])).toEqual(["f2"]);
    expect(seriesCoverageGaps([step], ["f1", "f2"])).toEqual([{ index: 0, missing: ["f2"] }]);
  });
});

describe("step reorder helpers", () => {
  it("moves a written step up and down", () => {
    expect(moveStep(["a", "b", "c"], 1, -1)).toEqual(["b", "a", "c"]);
    expect(moveStep(["a", "b", "c"], 1, 1)).toEqual(["a", "c", "b"]);
    expect(moveStep(["a", "b", "c"], 0, -1)).toEqual(["a", "b", "c"]);
    expect(moveStep(["a", "b", "c"], 2, 1)).toEqual(["a", "b", "c"]);
  });

  it("inserts below the current step and refuses the ceiling", () => {
    expect(insertStepAfter(["a", "b"], 0, "x", 20)).toEqual(["a", "x", "b"]);
    expect(insertStepAfter(["a", "b"], 1, "x", 2)).toEqual(["a", "b"]);
  });

  it("deletes a step but keeps at least one", () => {
    expect(removeStep(["a", "b"], 0)).toEqual(["b"]);
    expect(removeStep(["a"], 0)).toEqual(["a"]);
  });
});

describe("draft content detection", () => {
  it("ignores a blank default step", () => {
    expect(stepHasContent(emptySeriesStep(["f1"]))).toBe(false);
    expect(seriesDraftHasContent("", [emptySeriesStep(["f1"])])).toBe(false);
  });

  it("counts text, tags or cover as content", () => {
    const tagged = emptySeriesStep(["f1"]);
    tagged.tagIds = ["truth"];
    expect(stepHasContent(tagged)).toBe(true);
    const withText = emptySeriesStep(["f1"]);
    withText.variants[0]!.text = "先做这个";
    expect(seriesDraftHasContent("", [withText])).toBe(true);
  });
});

describe("contribution list meta", () => {
  it("distinguishes live-content revision states without treating the old version as offline", () => {
    expect(contributionStatusLabel.revision_draft).toBe("修订草稿");
    expect(contributionStatusLabel.revision_pending).toBe("修订审核中");
    expect(contributionStatusLabel.revision_rejected).toBe("修订被驳回");
    expect(contributionHasPublishedVersion("revision_draft")).toBe(true);
    expect(contributionHasPublishedVersion("revision_pending")).toBe(true);
    expect(contributionHasPublishedVersion("revision_rejected")).toBe(true);
  });

  it("returns the bare day for unapproved items (status now lives in the chip)", () => {
    const day = new Date(2026, 7, 26, 12, 0, 0).getTime();
    expect(formatContributionDay(day)).toBe("26/08/26");
    expect(formatContributionListMeta({
      kind: "task",
      status: "pending",
      updatedAt: day,
    })).toBe("26/08/26");
  });

  it("shows date plus like and series completion for approved items", () => {
    const day = new Date(2026, 7, 20, 12, 0, 0).getTime();
    expect(formatContributionListMeta({
      kind: "task",
      status: "approved",
      updatedAt: day,
      likeRatio: 86,
    })).toBe("26/08/20 · 点赞 86%");
    expect(formatContributionListMeta({
      kind: "series",
      status: "approved",
      updatedAt: day,
      likeRatio: 86,
      completionRate: 100,
    })).toBe("26/08/20 · 点赞 86% · 完成 100%");
  });

  it("shows the old live version metrics during revision states", () => {
    expect(contributionVoteTail({ kind: "task", status: "pending" })).toBe("");
    expect(contributionVoteTail({ kind: "gender", status: "approved" })).toBe("");
    expect(contributionVoteTail({ kind: "task", status: "approved", likeRatio: 86 })).toBe("点赞 86%");
    expect(contributionVoteTail({ kind: "task", status: "revision_pending", likeRatio: 80 })).toBe("点赞 80%");
    expect(contributionVoteTail({ kind: "series", status: "approved", likeRatio: 86, completionRate: 100 })).toBe("点赞 86% · 完成 100%");
  });

  it("uses name as the contribution title", () => {
    expect(contributionDraftTitle({ name: "投稿任务", variants: [{ text: "正文" }] })).toBe("投稿任务");
    expect(contributionDraftTitle({ label: "狐狸" })).toBe("狐狸");
  });

  it("formats the admin review subtitle with queue counts", () => {
    expect(formatContributionReviewSubtitle({ pending: 9, revisionPending: 4, unpublishPending: 1 })).toBe("待审 9 · 复审 4 · 下架 1");
  });
});
