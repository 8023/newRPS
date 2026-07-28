// libheif-js 的 Emscripten glue 里有仅 Node 端才会走到的 require('fs'/'path'/'crypto')
// 分支（浏览器端 typeof require === 'undefined' 时不会执行），别名到这里让 Vite 不再警告。
export {};
