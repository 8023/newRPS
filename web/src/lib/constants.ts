export const tokenKey = "rps-online-token";
export const playerIdKey = "rps-player-id";
export const playerSecretKey = "rps-player-secret";
export const securityDisclaimerKey = "rps-online-security-disclaimer";
export const punishmentTagPrefsKey = "rps-punishment-tag-prefs";
export const defaultRoomName = "新的锤子剪刀布房间";
export const defaultOthelloRoomName = "新的黑白棋房间";
export const defaultTicTacToeRoomName = "新的井字棋房间";
export const defaultLiarsDiceRoomName = "新的大话骰房间";
export const defaultGomokuRoomName = "新的五子棋房间";
export const defaultJungleRoomName = "新的斗兽棋房间";
export const defaultChessRoomName = "新的国际象棋房间";
export const maxOriginalImageBytes = 10 * 1024 * 1024;
export const maxProofUploadBytes = 2 * 1024 * 1024;
export const maxProofPixels = 4_000_000;
export const maxAspectRatio = 21 / 9;
export const maxAvatarOriginalBytes = 3 * 1024 * 1024;
export const maxAvatarUploadBytes = 20 * 1024;
export const maxAvatarSquareDiffPx = 20;
export const avatarSquareSize = 200;
export const leaderboardRefreshMs = 10 * 60 * 1000;
export const othelloBoardThemes = [
  { id: "classic", name: "经典绿", description: "传统棋盘，最清楚耐看。", board: "#2f8a64", cell: "#38a474", line: "rgba(18, 72, 52, 0.55)", hover: "#45b883", border: "#2f7a5c", blackDisc: "radial-gradient(circle at 32% 28%, #5f6670, #10151a 64%)", whiteDisc: "radial-gradient(circle at 32% 28%, #ffffff, #d8e1e8 70%)", blackRing: "#e3eef5", whiteRing: "#2b4f40" },
  { id: "pastel", name: "粉蓝白", description: "柔和一点，适合夜里轻松玩。", board: "#d8f0ff", cell: "#f8d7e9", line: "rgba(81, 124, 155, 0.35)", hover: "#e9f7ff", border: "#8fc7e8", blackDisc: "radial-gradient(circle at 32% 28%, #526070, #101821 66%)", whiteDisc: "radial-gradient(circle at 32% 28%, #ffffff, #f2f7ff 72%)", blackRing: "#ffffff", whiteRing: "#6f8aa4" },
  { id: "midnight", name: "深夜蓝", description: "暗色棋盘，不刺眼。", board: "#172339", cell: "#24395d", line: "rgba(159, 190, 255, 0.24)", hover: "#2f4a78", border: "#6b8dd6", blackDisc: "radial-gradient(circle at 32% 28%, #707b90, #090d16 66%)", whiteDisc: "radial-gradient(circle at 32% 28%, #ffffff, #d9ecff 72%)", blackRing: "#8fb2ff", whiteRing: "#ffffff" },
  { id: "wood", name: "木纹棕", description: "温暖桌游感。", board: "#9a6a3d", cell: "#b8844d", line: "rgba(78, 46, 20, 0.45)", hover: "#c89459", border: "#7a4e2a", blackDisc: "radial-gradient(circle at 32% 28%, #695b50, #17100b 66%)", whiteDisc: "radial-gradient(circle at 32% 28%, #fff9ec, #ead6b9 72%)", blackRing: "#f0d09d", whiteRing: "#6d4324" },
  { id: "neon", name: "霓虹紫", description: "更游戏感，适合整活。", board: "#24133e", cell: "#43206f", line: "rgba(244, 157, 255, 0.34)", hover: "#5b2b94", border: "#f49dff", blackDisc: "radial-gradient(circle at 32% 28%, #7f6d94, #0e0718 66%)", whiteDisc: "radial-gradient(circle at 32% 28%, #ffffff, #f4d7ff 72%)", blackRing: "#f49dff", whiteRing: "#ffffff" }
] as const;
export type OthelloBoardThemeId = typeof othelloBoardThemes[number]["id"];
export const tictactoeBoardThemes = [
  { id: "paper", name: "纸面白", description: "干净清楚，像便签纸。", board: "#f0d18f", cell: "#fffaf0", line: "#d6aa55", hover: "#fff2cf", border: "#c68b32", x: "#2f6f9f", o: "#9d3860", win: "#ffe082" },
  { id: "mint", name: "薄荷绿", description: "清爽一点，不刺眼。", board: "#8bd7bc", cell: "#effdf7", line: "#58b395", hover: "#dcf8ee", border: "#3a9c7e", x: "#176d86", o: "#b64268", win: "#bdf3cd" },
  { id: "midnight", name: "夜间蓝", description: "晚上玩更舒服。", board: "#172339", cell: "#223553", line: "#48628d", hover: "#2f466c", border: "#6b8dd6", x: "#74c7ff", o: "#ff9fc2", win: "#4c416c" },
  { id: "candy", name: "糖果粉", description: "偏可爱，适合轻松局。", board: "#f7b6d2", cell: "#fff4fb", line: "#ea86b5", hover: "#ffe4f2", border: "#d65c98", x: "#4f83c7", o: "#c93674", win: "#ffe6a8" },
  { id: "arcade", name: "街机紫", description: "更亮一点，有游戏感。", board: "#24133e", cell: "#351a5b", line: "#7a42c8", hover: "#43206f", border: "#f49dff", x: "#6ff7ff", o: "#ff78d2", win: "#64427f" }
] as const;
export type TicTacToeBoardThemeId = typeof tictactoeBoardThemes[number]["id"];
/** 五子棋与黑白棋共用同一套棋盘/棋子配色主题 */
export const gomokuBoardThemes = othelloBoardThemes;
export type GomokuBoardThemeId = typeof gomokuBoardThemes[number]["id"];
/** 斗兽棋棋盘主题：草地/水域/陷阱/兽穴/棋子色 */
export const jungleBoardThemes = [
  { id: "forest", name: "密林绿", description: "经典草地，清晰好认。", board: "#3d7a4a", land: "#4e9a5e", water: "#3a7eb8", trap: "#c4a35a", den: "#6b3f2a", border: "#2f5c3a", aPiece: "#f0f4f8", bPiece: "#1a1f2a", aText: "#1a2a1f", bText: "#f0f4f8", hover: "rgba(255,255,255,0.18)" },
  { id: "bamboo", name: "竹影青", description: "清爽竹林感。", board: "#5a9a6e", land: "#7bb88a", water: "#4a90b8", trap: "#d4b86a", den: "#7a5030", border: "#3d6b4a", aPiece: "#fffaf0", bPiece: "#243028", aText: "#243028", bText: "#fffaf0", hover: "rgba(255,255,255,0.2)" },
  { id: "river", name: "溪畔蓝", description: "偏冷色，夜晚也舒服。", board: "#2d5a6e", land: "#3d7a8e", water: "#1e6a9a", trap: "#b89a5a", den: "#5a3828", border: "#4a90a8", aPiece: "#eef6ff", bPiece: "#121820", aText: "#1a3040", bText: "#eef6ff", hover: "rgba(180,220,255,0.18)" },
  { id: "dusk", name: "暮色橙", description: "温暖桌游感。", board: "#8a5a3a", land: "#a87848", water: "#3a7090", trap: "#d4a050", den: "#5a3020", border: "#6a4028", aPiece: "#fff6e8", bPiece: "#1c1410", aText: "#3a2818", bText: "#fff6e8", hover: "rgba(255,220,160,0.16)" },
  { id: "night", name: "夜林紫", description: "暗色棋盘，不刺眼。", board: "#1e2438", land: "#2a3450", water: "#1a4068", trap: "#6a5838", den: "#3a2820", border: "#6b8dd6", aPiece: "#e8f0ff", bPiece: "#0a0e16", aText: "#1a2438", bText: "#e8f0ff", hover: "rgba(140,170,255,0.16)" }
] as const;
export type JungleBoardThemeId = typeof jungleBoardThemes[number]["id"];
export const chessBoardThemes = [
  { id: "classic", name: "经典褐", description: "最常见的木质棋盘。", light: "#f0d9b5", dark: "#b58863", board: "#8b5a2b", border: "#6b4220", hover: "rgba(255,255,255,0.22)", last: "rgba(246, 224, 94, 0.45)", check: "rgba(220, 38, 38, 0.42)" },
  { id: "wood", name: "深木纹", description: "更深一点，对比更强。", light: "#e8c39e", dark: "#8b5a2b", board: "#6b3f1f", border: "#4a2a12", hover: "rgba(255,255,255,0.18)", last: "rgba(255, 210, 80, 0.4)", check: "rgba(220, 38, 38, 0.4)" },
  { id: "midnight", name: "深夜蓝", description: "暗色棋盘，不刺眼。", light: "#4a6080", dark: "#1a2a40", board: "#121820", border: "#6b8dd6", hover: "rgba(180,210,255,0.16)", last: "rgba(96, 165, 250, 0.38)", check: "rgba(248, 113, 113, 0.45)" },
  { id: "marble", name: "大理石", description: "灰白棋盘，干净清楚。", light: "#e8e8e8", dark: "#6a6a6a", board: "#4a4a4a", border: "#2a2a2a", hover: "rgba(255,255,255,0.2)", last: "rgba(250, 204, 21, 0.38)", check: "rgba(220, 38, 38, 0.4)" },
  { id: "green", name: "赛场绿", description: "比赛常用的绿格。", light: "#eeeed2", dark: "#769656", board: "#4a6a38", border: "#3a5530", hover: "rgba(255,255,255,0.2)", last: "rgba(250, 204, 21, 0.42)", check: "rgba(220, 38, 38, 0.42)" }
] as const;
export type ChessBoardThemeId = typeof chessBoardThemes[number]["id"];
/** 黑白棋/五子棋/斗兽棋/国际象棋建房「每子时长」下拉选项（秒），0 = 不限时 */
export const moveSecondsOptions = [0, 30, 45, 60, 90, 120, 180] as const;
/** 黑白棋/五子棋/斗兽棋/国际象棋建房「每局时长」下拉选项（分钟），0 = 不限时 */
export const gameMinutesOptions = [0, 5, 10, 15, 20, 30, 45, 60] as const;

export const doumiaoLinks = [
  { id: "x", label: "X", title: "关注 X 账号", description: "看更新、吐槽和临时公告。", href: "https://x.com/doumiaojiang", icon: "𝕏", tone: "#111827" },
  { id: "telegram", label: "TG", title: "加入 TG 群", description: "一起聊天、反馈 bug、催新玩法。", href: "https://t.me/+X1Jr4GPxgIwzOWY1", icon: "✈", tone: "#229ed9" },
  { id: "afdian", label: "爱发电", title: "爱发电支持", description: "国内赞助入口，支持一点服务器电费。", href: "https://afdian.com/a/doumiaojiang", icon: "⚡", tone: "#946cff" },
  { id: "patreon", label: "Patreon", title: "Patreon", description: "海外赞助入口，适合长期支持。", href: "https://www.patreon.com/customize?step=navigation", icon: "P", tone: "#ff424d" },
  { id: "coffee", label: "Coffee", title: "来一杯咖啡", description: "请作者喝杯咖啡，继续加玩法。", href: "https://buymeacoffee.com/doumiaojiang", icon: "☕", tone: "#f2b84b" }
] as const;

export const luv4uLinks = [
  { id: "x", label: "X", title: "关注 X 账号", description: "看看腐竹的日常", href: "https://x.com/0x8023", icon: "𝕏", tone: "#111827" },
  { id: "x", label: "X", title: "关注 X 账号", description: "看看腐竹的色图。", href: "https://x.com/YukiHoshikawa", icon: "𝕏", tone: "#111827" },
  { id: "telegram", label: "TG", title: "TG 私聊", description: "与腐竹说点悄悄话", href: "https://t.me/+X1Jr4GPxgIwzOWY1", icon: "✈", tone: "#229ed9" }
] as const;