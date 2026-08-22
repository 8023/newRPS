import { describe, expect, it } from "vitest";
import { seriesFactionWarning, seriesFactionWarningFor } from "./seriesFaction";

const config = {
  genderFactions: [
    { id: "f1", label: "甲营", taskGroup: "default", textColor: "", backgroundColor: "", borderColor: "" },
    { id: "f2", label: "乙营", taskGroup: "default", textColor: "", backgroundColor: "", borderColor: "" },
  ],
  punishmentSeriesSummaries: [
    { id: "s1", name: "试炼", stepCount: 12, targetFactionIds: ["f1"] },
  ],
};

describe("seriesFactionWarning", () => {
  it("returns null when the player faction is covered", () => {
    expect(seriesFactionWarning(
      { settings: { punishmentSource: "series", punishmentSeriesId: "s1" } } as never,
      config,
      { factionId: "f1" },
    )).toBeNull();
  });

  it("warns when entering a series room that does not cover the player", () => {
    const msg = seriesFactionWarning(
      { settings: { punishmentSource: "series", punishmentSeriesId: "s1" } } as never,
      config,
      { factionId: "f2" },
    );
    expect(msg).toContain("甲营");
    expect(msg).toContain("剧情会断");
  });

  it("uses the same warning for the flat lobby-room summary before joining", () => {
    const msg = seriesFactionWarningFor("series", "s1", config, { factionId: "f2" });
    expect(msg).toContain("甲营");
    expect(msg).toContain("仍然要进入吗");
  });

  it("does not warn before joining a random-task room", () => {
    expect(seriesFactionWarningFor("random", "s1", config, { factionId: "f2" })).toBeNull();
  });
});
