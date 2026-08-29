<script lang="ts">
  // 组合根：只负责"挂载哪些 store 的生命周期效应 + 按当前视图渲染哪个组件"，不持有任何
  // 业务状态、不写业务逻辑——这些全部下放到 lib/stores/*.svelte.ts。对照原 React 版
  // App.tsx（一个文件揉合了会话状态、socket 订阅、路由、UI 状态、顶栏渲染、6+ 个弹窗，
  // 643 行），这里的拆分原则是：
  //   - sessionStore：会话/大厅/房间/配置，及其 socket 订阅（与既有 chatStore 同构）
  //   - routerStore：纯导航状态
  //   - uiStore：纯客户端 UI 状态（弹窗开关、主题、toast 触发文案）
  //   - 各视图/弹窗组件直接 import 需要的 store，不再经手 App 一层层转发 props
  // 新增视图/弹窗时，照这个模式加，不要把状态或逻辑塞回本文件。
  import { onMount, untrack, type Component } from "svelte";
  import { sessionStore } from "./lib/stores/sessionStore.svelte";
  import { routerStore } from "./lib/stores/routerStore.svelte";
  import { uiStore } from "./lib/stores/uiStore.svelte";
  import { startAnalytics, trackPageview } from "./lib/analytics";
  import { ask, isAdminRoute } from "./lib/rpc";
  import SecurityDisclaimer from "./ui/shell/SecurityDisclaimer.svelte";
  import TopBar from "./ui/shell/TopBar.svelte";
  import Toast from "./ui/shell/Toast.svelte";
  import AnnouncementPopup from "./ui/shell/AnnouncementPopup.svelte";
  import VersionUpdateBanner from "./ui/shell/VersionUpdateBanner.svelte";
  import RestoreKickPrompt from "./ui/shell/RestoreKickPrompt.svelte";
  import Login from "./ui/shell/Login.svelte";
  import Lobby from "./ui/lobby/Lobby.svelte";
  import Room from "./ui/room/Room.svelte";
  import ContributeView from "./ui/contribute/ContributeView.svelte";
  import AboutPanel from "./ui/about/AboutPanel.svelte";
  import HelpPanel from "./ui/about/HelpPanel.svelte";
  import ProfilePanel from "./ui/profile/ProfilePanel.svelte";
  import GlobalLeaderboardPanel from "./ui/social/GlobalLeaderboardPanel.svelte";

  onMount(() => {
    startAnalytics();
    sessionStore.connect();
  });

  // 与 React 版 useEffect([view]) 一致：初次挂载上报初始页，之后仅在主视图真正变化时
  // 上报。同值的 room:update 不会使依赖失效，因此不会把房态推送误算成页面浏览。
  $effect(() => {
    trackPageview(routerStore.view);
  });

  // 弹窗浏览同样保持 React 版的四个窄依赖 effect；无论从哪个入口打开，都只在
  // false -> true 时上报一次。
  $effect(() => {
    if (uiStore.profileOpen) trackPageview("profile");
  });
  $effect(() => {
    if (uiStore.leaderboardOpen) trackPageview("leaderboard");
  });
  $effect(() => {
    if (uiStore.aboutOpen) trackPageview("about");
  });
  $effect(() => {
    if (uiStore.helpOpen) trackPageview("help");
  });

  // 后台整体单独打包，普通玩家不会下载后台配置、用户管理和关系图等代码。
  let AdminPanelComponent = $state<Component | null>(null);
  $effect(() => {
    if (routerStore.view === "admin" && sessionStore.lobby && !AdminPanelComponent) {
      import("./ui/admin/AdminPanel.svelte").then((module) => {
        AdminPanelComponent = module.default;
      });
    }
  });

  // 各 store 的订阅式副作用：每个都只是"挂载一次、返回清理函数"，具体逻辑在对应 store
  // 方法里，这里只负责按 Svelte 组件生命周期把它们接上。
  $effect(() => sessionStore.wireSocketHandlers());
  $effect(() => sessionStore.startLeaderboardRefreshTimer());
  $effect(() => routerStore.wireHashAndKeyboard());

  $effect(() => {
    document.documentElement.dataset.theme = uiStore.theme;
  });

  $effect(() => {
    if (sessionStore.config && !sessionStore.config.securityDisclaimer.enabled) uiStore.releaseDisclaimerIfDisabled();
  });

  // 收敛依赖 sessionStore.syncMeFromLobby 内部的幂等比较（合并后不再变化就不再写 me），
  // 详见该方法注释。
  $effect(() => {
    void sessionStore.lobby;
    void sessionStore.me;
    sessionStore.syncMeFromLobby();
  });

  // 原 deps 是 [view, me?.player.id]：只在「视图切换」或「玩家身份变化」时才发订阅 RPC，
  // 玩家资料的其余字段变动（player:batch 高频推送）不应触发。meId 是 $derived，Svelte
  // 只在其输出值真正变化时才使下游失效，天然复现了这条窄依赖（见 plan.md §6.2）。
  let meId = $derived(sessionStore.me?.player.id);
  $effect(() => {
    const currentView = routerStore.view;
    const id = meId;
    if (!id || isAdminRoute()) return;
    untrack(() => ask(currentView === "room" ? "lobby:unsubscribe" : "lobby:subscribe", {}).catch(() => undefined));
  });
</script>

<!-- 声明页先于"正在连接服务器"展示：WS 连接与声明页的强制停留同时进行，不用先等连上
     服务器才弹声明，省掉两段等待叠加的时间。 -->
{#if !uiStore.disclaimerConfirmed}
  <SecurityDisclaimer onConfirm={() => uiStore.confirmDisclaimer()} />
{:else if !sessionStore.config}
  <div class="loading">正在连接服务器...</div>
{:else}
  <main>
    <TopBar />
    <Toast />
    <AnnouncementPopup />
    <VersionUpdateBanner />

    {#if routerStore.view === "login" && sessionStore.restoringSession && !sessionStore.restoreKickPending}
      <section class="panel">正在恢复登录状态...</section>
    {/if}
    {#if routerStore.view === "login" && sessionStore.restoreKickPending}
      <RestoreKickPrompt />
    {/if}
    {#if routerStore.view === "login" && !sessionStore.restoringSession && !sessionStore.restoreKickPending}
      <Login />
    {/if}
    {#if routerStore.view === "lobby" && sessionStore.me && sessionStore.lobby}
      <Lobby />
    {/if}
    {#if (routerStore.view === "contribute" || (routerStore.view === "admin" && routerStore.keepContribute)) && sessionStore.me && sessionStore.config}
      <div hidden={routerStore.view !== "contribute"}>
        <ContributeView />
      </div>
    {/if}
    {#if routerStore.view === "room" && sessionStore.me && sessionStore.room}
      <Room />
    {/if}
    {#if routerStore.view === "admin" && sessionStore.lobby}
      {#if AdminPanelComponent}
        <AdminPanelComponent />
      {:else}
        <div class="loading">正在加载后台管理…</div>
      {/if}
    {/if}
    {#if routerStore.view === "room" && !sessionStore.room}
      <section class="panel">你暂时不在房间里。</section>
    {/if}

    {#if uiStore.aboutOpen}<AboutPanel />{/if}
    {#if uiStore.helpOpen}<HelpPanel />{/if}
    {#if uiStore.profileOpen && sessionStore.me}<ProfilePanel />{/if}
    {#if uiStore.leaderboardOpen}<GlobalLeaderboardPanel players={sessionStore.leaderboardSource} />{/if}
  </main>
{/if}
