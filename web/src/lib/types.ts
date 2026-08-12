import type { PublicPlayer, RoomSnapshot } from "../shared/types";

// punishmentTagPrefs：仅本人可见的惩罚标签偏好（tagId -> include|exclude），开房面板预填用。
// 服务端刻意不放进 PublicPlayer（会被广播给房间里/大厅其他人），只在 player:join 应答里
// 作为独立字段下发，见后端 onPlayerJoin 注释。
export type MeState = { player: PublicPlayer; token: string; roomId?: string; room?: RoomSnapshot; reissuedSecret?: string; punishmentTagPrefs?: Record<string, string> };
export type AnnouncementPayload = { id: string; message: string; durationMs: number; createdAt: number };
