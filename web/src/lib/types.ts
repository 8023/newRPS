import type { PublicPlayer, RoomSnapshot } from "../shared/types";

export type MeState = { player: PublicPlayer; token: string; roomId?: string; room?: RoomSnapshot; reissuedSecret?: string };
export type AnnouncementPayload = { id: string; message: string; durationMs: number; createdAt: number };
