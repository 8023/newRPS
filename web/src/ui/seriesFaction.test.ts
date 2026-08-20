import { describe, expect, it } from "vitest";
import { seriesFactionWarning } from "./seriesFaction";

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

  it("warns when sitting into a series that does not cover the player", () => {
    const msg = seriesFactionWarning(
      { settings: { punishmentSource: "series", punishmentSeriesId: "s1" } } as never,
      config,
      { factionId: "f2" },
    );
    expect(msg).toContain("甲营");
    expect(msg).toContain("剧情会断");
  });
});
