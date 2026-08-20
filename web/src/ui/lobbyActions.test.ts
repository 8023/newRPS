import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

describe("lobby actions", () => {
  it("places contribute button left of create room in Lobby markup", () => {
    const src = readFileSync(new URL("./AppViews.tsx", import.meta.url), "utf8");
    const contribute = src.indexOf(">参与共建<");
    const create = src.indexOf(">创建房间<");
    expect(contribute).toBeGreaterThan(0);
    expect(create).toBeGreaterThan(contribute);
    const contributeBtn = src.slice(src.lastIndexOf("<button", contribute), src.indexOf(">", contribute) + 1);
    const createBtn = src.slice(src.lastIndexOf("<button", create), src.indexOf(">", create) + 1);
    expect(contributeBtn).toContain("small");
    expect(createBtn).toContain("small");
  });
});

describe("admin contribution review wiring", () => {
  it("wraps the pending list as items and shows queue counts in the subtitle helper", () => {
    const src = readFileSync(new URL("./AdminContributionReview.tsx", import.meta.url), "utf8");
    expect(src).toContain("res.items");
    expect(src).toContain("onCounts");
    expect(src).not.toContain("<textarea value={content}");
    expect(src).toContain("ContributionPreview");
  });

  it("labels liar's dice stakes as per-piece and others as per-game", () => {
    const src = readFileSync(new URL("./AdminViews.tsx", import.meta.url), "utf8");
    expect(src).toContain("（每子）");
    expect(src).toContain("（每局）");
    expect(src).toContain("formatContributionReviewSubtitle(contributionCounts)");
  });
});

describe("contribute keepalive across admin", () => {
  it("keeps the contribute page mounted while visiting admin", () => {
    const src = readFileSync(new URL("../App.tsx", import.meta.url), "utf8");
    expect(src).toContain("keepContribute");
    expect(src).toContain('hidden={view !== "contribute"}');
  });
});
