package server

// BuildID 由构建时 -ldflags -X 注入（git 短哈希，见根目录 package.json 的
// build:server 脚本），未注入时保持 "dev"，与前端 vite.config.ts 未注入
// __APP_BUILD_ID__ 时的兜底值对齐，确保本地开发环境（npm run dev）两端
// 恒相等、不会误报版本不一致。
//
// 用途：随连接建立时的 server:hello 推送、以及心跳 ping 应答一起下发给
// 前端，供前端与自身构建时嵌入的 buildId 比对，提示用户刷新页面。这是
// 一个提示性质的只读标识，不参与任何鉴权/限流判断，客户端无法影响它。
var BuildID = "dev"
