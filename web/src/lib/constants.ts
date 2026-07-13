export const tokenKey = "rps-online-token";
export const playerIdKey = "rps-player-id";
export const playerSecretKey = "rps-player-secret";
export const dailyAnnouncementKey = "rps-online-daily-announcement";
export const defaultRoomName = "新的锤子剪刀布房间";
export const defaultOthelloRoomName = "新的黑白棋房间";
export const defaultTicTacToeRoomName = "新的井字棋房间";
export const maxOriginalImageBytes = 10 * 1024 * 1024;
export const maxProofUploadBytes = 2 * 1024 * 1024;
export const maxProofPixels = 4_000_000;
export const maxAspectRatio = 21 / 9;
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
export const sponsorLinks = [
  { id: "x", label: "X", title: "关注 X 账号", description: "看更新、吐槽和临时公告。", href: "https://x.com/home", icon: "𝕏", tone: "#111827" },
  { id: "telegram", label: "TG", title: "加入 TG 群", description: "一起聊天、反馈 bug、催新玩法。", href: "https://t.me/+X1Jr4GPxgIwzOWY1", icon: "✈", tone: "#229ed9" },
  { id: "afdian", label: "爱发电", title: "爱发电支持", description: "国内赞助入口，支持一点服务器电费。", href: "https://afdian.com/a/doumiaojiang", icon: "⚡", tone: "#946cff" },
  { id: "patreon", label: "Patreon", title: "Patreon", description: "海外赞助入口，适合长期支持。", href: "https://www.patreon.com/customize?step=navigation", icon: "P", tone: "#ff424d" },
  { id: "coffee", label: "Coffee", title: "来一杯咖啡", description: "请作者喝杯咖啡，继续加玩法。", href: "https://buymeacoffee.com/doumiaojiang", icon: "☕", tone: "#f2b84b" }
] as const;
