import { describe, expect, it } from "vitest";
import { normalizeRoundHistoryItem } from "./normalize";
import { formatHistoryTime } from "./format";
import type { RoomSnapshot } from "../shared/types";

type HistoryItem = RoomSnapshot["roundHistory"][number];

/** 只填测试关心的字段，其余交给 normalizeRoundHistoryItem 兜底。 */
function itemWithAt(at: unknown): HistoryItem {
  return { id: "r1", round: 1, at } as unknown as HistoryItem;
}

describe("对局记录时间戳", () => {
  // 回归钉子：protojson 按规范把 int64 编成十进制字符串，房间频道的 Struct/DELTA 路径
  // 没有 decodeEnvelope 那层 longs:Number 兜底，at 会以 "1788002325326" 这种字符串上来。
  // new Date(该字符串) 是按「日期字符串」解析的，得到 Invalid Date——不是按纪元毫秒。
  it("把 protojson 传来的十进制字符串收成数字", () => {
    const normalized = normalizeRoundHistoryItem(itemWithAt("1788002325326"));
    expect(normalized.at).toBe(1788002325326);
    expect(typeof normalized.at).toBe("number");
  });

  it("数字时间戳原样保留", () => {
    expect(normalizeRoundHistoryItem(itemWithAt(1788002325326)).at).toBe(1788002325326);
  });

  it("缺失/非法时间戳归零，不产生 NaN", () => {
    expect(normalizeRoundHistoryItem(itemWithAt(undefined)).at).toBe(0);
    expect(normalizeRoundHistoryItem(itemWithAt("abc")).at).toBe(0);
  });

  it("formatHistoryTime 永不吐出 Invalid Date", () => {
    for (const bad of [0, -1, NaN, undefined as unknown as number, "abc" as unknown as number]) {
      expect(formatHistoryTime(bad)).toBe("时间未知");
    }
    // 合法值仍走本地化时间；只断言"不是 Invalid Date"，避免依赖运行环境的 locale/时区。
    expect(formatHistoryTime(1788002325326)).not.toContain("Invalid");
    expect(formatHistoryTime("1788002325326" as unknown as number)).not.toContain("Invalid");
  });
});
