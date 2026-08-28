<script lang="ts">
  // 证明图：blob 立刻显示；远端图先占位，滚入视口附近（IntersectionObserver）才挂载 <img>，
  // 避免对局记录瀑布流一次性发起大量图片请求。处理缓存图 onLoad 不触发导致一直「加载中」。
  // 源：ui/AppViews.tsx:2841-2913
  import { untrack } from "svelte";

  let { src, alt, className = "" }: { src: string; alt: string; className?: string } = $props();

  const isLocal = $derived(src.startsWith("blob:") || src.startsWith("data:"));

  // 只取一次初始值：src 变化时的重置由下面第一个 $effect 统一处理，不靠这里的响应式追踪。
  let loaded = $state(untrack(() => isLocal));
  let failed = $state(false);
  let visible = $state(untrack(() => isLocal));
  let imgEl: HTMLImageElement | null = $state(null);
  let wrapEl: HTMLSpanElement | null = $state(null);

  $effect(() => {
    const local = src.startsWith("blob:") || src.startsWith("data:");
    void local; // 依赖追踪：src 变化时重置三个状态
    failed = false;
    loaded = local;
    visible = local;
  });

  $effect(() => {
    if (isLocal || visible) return;
    const el = wrapEl;
    if (!el || typeof IntersectionObserver === "undefined") {
      visible = true;
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          visible = true;
          observer.disconnect();
        }
      },
      { rootMargin: "400px" }
    );
    observer.observe(el);
    return () => observer.disconnect();
  });

  $effect(() => {
    if (!visible) return;
    // 下一帧检查：缓存命中时 complete 已为 true，不会再触发 onLoad
    const id = window.requestAnimationFrame(() => {
      const el = imgEl;
      if (el && el.complete && el.naturalWidth > 0) loaded = true;
    });
    return () => window.cancelAnimationFrame(id);
  });
</script>

{#if isLocal}
  <img class={className} {src} {alt} />
{:else}
  <span bind:this={wrapEl} class={`proof-image-wrap ${loaded ? "is-ready" : ""} ${className}`}>
    {#if !loaded && !failed}
      <span class="proof-image-loading">
        <span>图片加载中…</span>
        <span class="proof-image-loading-bar" aria-hidden="true"><i></i></span>
      </span>
    {/if}
    {#if failed}
      <span class="proof-image-loading">图片加载失败</span>
    {:else if visible}
      <img
        bind:this={imgEl}
        {src}
        {alt}
        loading="lazy"
        decoding="async"
        class={loaded ? "is-loaded" : ""}
        onload={() => (loaded = true)}
        onerror={() => (failed = true)}
      />
    {/if}
  </span>
{/if}
