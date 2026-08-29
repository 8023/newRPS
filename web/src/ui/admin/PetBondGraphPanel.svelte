<script module lang="ts">
  import type { SimulationNodeDatum } from "d3-force";
  import type { PublicPlayer } from "../../shared/types";

  type PetBondGraphEdge = { masterId: string; petId: string; petTitle?: string; createdAt: number };
  type PetBondGraphData = { nodes: PublicPlayer[]; edges: PetBondGraphEdge[] };
  type PlayerHit = { id: string; name: string; connected: boolean };
  type SimPlayerNode = PublicPlayer & SimulationNodeDatum;
  // source/target 初始是字符串 id；forceLink 的 .id() 访问器在模拟第一次 initialize 时
  // 会把它们原地替换成对应的节点对象引用（与 tickNodes 里的对象同一实例），之后每次 tick
  // 读到的就是带最新 x/y 的节点——直接从这两个字段取端点坐标，不需要额外维护 linkPositions。
  type SimLink = { source: string | SimPlayerNode; target: string | SimPlayerNode; masterId: string; petId: string; petTitle?: string };

  const HEIGHT = 690;
  const NODE_RADIUS = 18;
  const AVATAR_SIZE = 36;
  const DRAG_CLICK_THRESHOLD = 5;

  /** 以 rootId 为中心，把主/宠边当无向图做 BFS，收集 depth 跳以内可达的玩家 id。
   * depth=2 时会自然覆盖"主人的主人""主人的其它宠物""宠物的其它主人"等场景——
   * 因为这些节点都通过共同的中间节点落在 2 跳以内。 */
  function collectPetBondNeighbors(graph: PetBondGraphData, rootId: string, depth: number): Set<string> {
    const adjacency = new Map<string, string[]>();
    for (const edge of graph.edges) {
      if (!adjacency.has(edge.masterId)) adjacency.set(edge.masterId, []);
      if (!adjacency.has(edge.petId)) adjacency.set(edge.petId, []);
      adjacency.get(edge.masterId)!.push(edge.petId);
      adjacency.get(edge.petId)!.push(edge.masterId);
    }
    const visited = new Set<string>([rootId]);
    let frontier = [rootId];
    for (let hop = 0; hop < depth && frontier.length > 0; hop++) {
      const next: string[] = [];
      for (const id of frontier) {
        for (const neighbor of adjacency.get(id) || []) {
          if (!visited.has(neighbor)) {
            visited.add(neighbor);
            next.push(neighbor);
          }
        }
      }
      frontier = next;
    }
    return visited;
  }
</script>

<script lang="ts">
  /**
   * 「主宠关系」力导向图：原版是手写 O(n²) 排斥力 + 弹簧力的 requestAnimationFrame 动画循环，
   * 现改用 LayerChart 的 ForceSimulation 组件（对 d3-force 的声明式封装）驱动物理引擎——
   * charge/link/collide/center 四种力替代原手写公式，拖拽节点时通过临时抬高 alpha「重新加热」
   * 模拟（标准 d3-drag 手法），松开后固定点 fx/fy 清空，交回给力引导自然回弹。
   * 节点位置沿用同一批 SimPlayerNode 对象在多次 tick 间保持引用不变以维持连续性；仅当
   * 可见节点集合（图数据变化 / 聚焦筛选变化）真正改变时才重新构造 data.nodes 传入模拟器，
   * 此时从 #positions 位置缓存里回填已存在节点的坐标，新节点按环形分布给初始位置——
   * 与原版"节点集合变化时补齐新节点初始位置"的语义一致。
   * 源：ui/PetBondGraphPanel.tsx。
   */
  import { untrack } from "svelte";
  import { forceManyBody, forceLink, forceCenter, forceCollide } from "d3-force";
  import { ForceSimulation } from "layerchart/force";
  import { ask } from "../../lib/rpc";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";
  import AdminSectionHeader from "./AdminSectionHeader.svelte";

  let { onError }: { onError: (message: string) => void } = $props();

  let graph = $state<PetBondGraphData>({ nodes: [], edges: [] });
  let loading = $state(false);
  let adding = $state(false);
  let selectedId = $state<string | null>(null);

  let masterQuery = $state("");
  let masterHits = $state<PlayerHit[]>([]);
  let masterPicked = $state<PlayerHit | null>(null);
  let petQuery = $state("");
  let petHits = $state<PlayerHit[]>([]);
  let petPicked = $state<PlayerHit | null>(null);

  let filterQuery = $state("");
  let filterHits = $state<PlayerHit[]>([]);
  let filterPicked = $state<PlayerHit | null>(null);
  let filterDepth = $state(10);

  // 关系图不再是固定尺寸：宽度随卡片实际渲染宽度变化，撑满 petbond-graph-svg；高度固定。
  let svgWidth = $state(720);
  let svgEl = $state<SVGSVGElement | null>(null);

  // 筛选：选定一位玩家后，只保留与其在 filterDepth 跳以内可达的主宠关系；未选人时不筛选。
  const visibleIds = $derived.by(() => {
    if (!filterPicked) return null;
    if (!graph.nodes.some((n) => n.id === filterPicked!.id)) return new Set<string>();
    return collectPetBondNeighbors(graph, filterPicked.id, Math.max(0, filterDepth));
  });

  const displayGraph = $derived.by<PetBondGraphData>(() => {
    if (!visibleIds) return graph;
    return {
      nodes: graph.nodes.filter((n) => visibleIds.has(n.id)),
      edges: graph.edges.filter((e) => visibleIds.has(e.masterId) && visibleIds.has(e.petId))
    };
  });

  // 位置缓存：节点集合变化时为已有节点保留坐标、为新节点分配环形初始位置——不用 $state，
  // 纯粹的模拟私有簿记，变化不需要触发 Svelte 重渲染（渲染由 ForceSimulation 的 tick 驱动）。
  const positions = new Map<string, { x: number; y: number }>();

  const simNodes = $derived.by<SimPlayerNode[]>(() => {
    const nodes = displayGraph.nodes;
    return nodes.map((player, index) => {
      const prev = positions.get(player.id);
      let x: number, y: number;
      if (prev) {
        ({ x, y } = prev);
      } else {
        const angle = (index / Math.max(1, nodes.length)) * Math.PI * 2;
        x = svgWidth / 2 + Math.cos(angle) * 150 + (Math.random() - 0.5) * 30;
        y = HEIGHT / 2 + Math.sin(angle) * 150 + (Math.random() - 0.5) * 30;
      }
      return Object.assign({}, player, { x, y }) as SimPlayerNode;
    });
  });
  // 只保留两端节点都在场的边：d3 的 forceLink 在 initialize 时按 id 查节点，查不到会直接
  // 抛 "node not found" 把整个后台面板打白屏。原 React 版是手写循环 `if (!a || !b) continue`
  // 自然跳过，这里要显式补回同样的容错，防止服务端返回不自洽的 nodes/edges 时炸掉。
  const simLinks = $derived.by<SimLink[]>(() => {
    const present = new Set(simNodes.map((n) => n.id));
    return displayGraph.edges
      .filter((edge) => present.has(edge.masterId) && present.has(edge.petId))
      .map((edge) => ({ source: edge.masterId, target: edge.petId, masterId: edge.masterId, petId: edge.petId, petTitle: edge.petTitle }));
  });
  const simData = $derived({ nodes: simNodes, links: simLinks });
  const forces = $derived({
    charge: forceManyBody().strength(-260).distanceMax(420),
    link: forceLink(simLinks).id((d: any) => d.id).distance(170).strength(0.35),
    center: forceCenter(svgWidth / 2, HEIGHT / 2).strength(0.02),
    collide: forceCollide(NODE_RADIUS + 6)
  });

  let simAlpha = $state(1);
  let tickNodes = $state<SimPlayerNode[]>([]);
  let tickLinks = $state<SimLink[]>([]);

  function onTick(e: { nodes: SimPlayerNode[]; links: (SimLink | undefined)[] }) {
    const maxX = Math.max(AVATAR_SIZE, svgWidth - AVATAR_SIZE);
    for (const node of e.nodes) {
      if (node.fx == null) node.x = Math.min(maxX, Math.max(AVATAR_SIZE, node.x ?? 0));
      if (node.fy == null) node.y = Math.min(HEIGHT - AVATAR_SIZE - 16, Math.max(AVATAR_SIZE, node.y ?? 0));
      positions.set(node.id, { x: node.x ?? 0, y: node.y ?? 0 });
    }
    tickNodes = e.nodes;
    tickLinks = e.links.filter((l): l is SimLink => l != null);
  }

  // 节点/边集合变化后必须给模拟"重新加热"：d3 的 alpha 会随时间衰减到 alphaMin 以下并
  // 停表，此后 LayerChart 的 resumeDynamicSimulation() 会因 `alpha < alphaMin` 直接返回，
  // 光换 data/forces 是不会重新开跑的——表现为添加/解除关系、切换聚焦筛选之后，新节点
  // 僵在初始环形位置上不动。原 React 版是永不停歇的 rAF 循环，没有这个状态，所以要在这里
  // 显式补上。只读 simData（不读 simAlpha）避免与 ForceSimulation 每 tick 回写 alpha 形成循环。
  $effect(() => {
    void simData;
    untrack(() => { if (simAlpha < 0.3) simAlpha = 0.6; });
  });

  $effect(() => {
    if (selectedId && visibleIds && !visibleIds.has(selectedId)) selectedId = null;
  });

  async function load() {
    loading = true;
    try {
      const result = await ask<PetBondGraphData>("admin:petBondGraph", {});
      graph = { nodes: result.nodes || [], edges: result.edges || [] };
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载主宠关系失败");
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void load();
  });

  function searchPlayers(keyword: string, apply: (hits: PlayerHit[]) => void) {
    if (!keyword.trim()) {
      apply([]);
      return;
    }
    ask<{ players?: Array<{ id: string; name: string; connected: boolean }> }>("admin:listPlayers", { keyword: keyword.trim(), limit: 8 })
      .then((result) => apply((result.players || []).map((p) => ({ id: p.id, name: p.name, connected: p.connected }))))
      .catch(() => {
        // 搜索联想失败不打断输入，静默忽略。
      });
  }

  function useSearchEffect(getQuery: () => string, apply: (hits: PlayerHit[]) => void) {
    $effect(() => {
      const query = getQuery();
      const handle = setTimeout(() => searchPlayers(query, apply), 250);
      return () => clearTimeout(handle);
    });
  }
  useSearchEffect(() => masterQuery, (hits) => (masterHits = hits));
  useSearchEffect(() => petQuery, (hits) => (petHits = hits));
  useSearchEffect(() => filterQuery, (hits) => (filterHits = hits));

  async function addRelation() {
    if (!masterPicked || !petPicked) return;
    if (masterPicked.id === petPicked.id) {
      onError("不能选择同一个人");
      return;
    }
    adding = true;
    try {
      const result = await ask<PetBondGraphData>("admin:petBondAdd", { masterId: masterPicked.id, petId: petPicked.id });
      graph = { nodes: result.nodes || [], edges: result.edges || [] };
      masterPicked = null;
      petPicked = null;
      masterQuery = "";
      petQuery = "";
      onError("已添加主宠关系");
    } catch (error) {
      onError(error instanceof Error ? error.message : "添加关系失败");
    } finally {
      adding = false;
    }
  }

  async function removeRelation(masterId: string, petId: string) {
    if (!window.confirm("确定解除这条主宠关系吗？")) return;
    try {
      const result = await ask<PetBondGraphData>("admin:petBondRemove", { masterId, petId });
      graph = { nodes: result.nodes || [], edges: result.edges || [] };
      onError("已解除主宠关系");
    } catch (error) {
      onError(error instanceof Error ? error.message : "解除关系失败");
    }
  }

  function toSvgPoint(clientX: number, clientY: number) {
    const svg = svgEl;
    if (!svg) return { x: svgWidth / 2, y: HEIGHT / 2 };
    const screenMatrix = svg.getScreenCTM();
    if (screenMatrix) {
      const point = svg.createSVGPoint();
      point.x = clientX;
      point.y = clientY;
      const transformed = point.matrixTransform(screenMatrix.inverse());
      return { x: transformed.x, y: transformed.y };
    }
    const rect = svg.getBoundingClientRect();
    return { x: ((clientX - rect.left) / Math.max(1, rect.width)) * svgWidth, y: ((clientY - rect.top) / Math.max(1, rect.height)) * HEIGHT };
  }

  let dragId: string | null = null;
  let dragMoved = false;
  let dragStart = { x: 0, y: 0 };
  let dragOffset = { x: 0, y: 0 };

  function findNode(id: string) {
    return tickNodes.find((n) => n.id === id) ?? simNodes.find((n) => n.id === id);
  }

  function onNodePointerDown(playerId: string, event: PointerEvent) {
    (event.currentTarget as Element).setPointerCapture(event.pointerId);
    dragId = playerId;
    dragMoved = false;
    dragStart = { x: event.clientX, y: event.clientY };
    const node = findNode(playerId);
    if (node) {
      const point = toSvgPoint(event.clientX, event.clientY);
      dragOffset = { x: (node.x ?? 0) - point.x, y: (node.y ?? 0) - point.y };
      node.fx = node.x;
      node.fy = node.y;
      simAlpha = Math.max(simAlpha, 0.3);
    }
  }

  function onSvgPointerMove(event: PointerEvent) {
    if (!dragId) return;
    const node = findNode(dragId);
    if (!node) return;
    const dx = event.clientX - dragStart.x;
    const dy = event.clientY - dragStart.y;
    if (Math.abs(dx) + Math.abs(dy) > DRAG_CLICK_THRESHOLD) dragMoved = true;
    const point = toSvgPoint(event.clientX, event.clientY);
    node.fx = point.x + dragOffset.x;
    node.fy = point.y + dragOffset.y;
    simAlpha = Math.max(simAlpha, 0.3);
  }

  function releaseDrag() {
    const draggedId = dragId;
    if (draggedId) {
      const node = findNode(draggedId);
      if (node) {
        node.fx = null;
        node.fy = null;
      }
      if (!dragMoved) {
        // 没有明显移动视为一次点击：选中/取消选中该节点，展示详细信息气泡。
        selectedId = selectedId === draggedId ? null : draggedId;
      }
    }
    dragId = null;
  }

  function onBackgroundPointerDown(event: PointerEvent) {
    if (event.target === event.currentTarget) selectedId = null;
  }

  const selectedNode = $derived(selectedId ? tickNodes.find((n) => n.id === selectedId) || null : null);
  const filtering = $derived(Boolean(filterPicked));
</script>

<AdminSectionHeader title="主宠关系" subtitle="拖动节点调整布局，点击头像查看详细资料，点击连线上的 × 可解除关系。" />
<div class="admin-preview-card">
  <span>概况</span>
  <p>
    {filtering
      ? `筛选后 ${displayGraph.nodes.length} 位玩家 · ${displayGraph.edges.length} 条关系（共 ${graph.nodes.length} 位玩家 · ${graph.edges.length} 条主宠关系）`
      : `${graph.nodes.length} 位玩家 · ${graph.edges.length} 条主宠关系`}
    {loading ? "（刷新中…）" : ""}
  </p>
</div>
<div class="petbond-graph-add">
  <div class="petbond-picker">
    <span class="field-label-caption">主人</span>
    {#if masterPicked}
      <div class="petbond-picker-picked">
        <span>{masterPicked.name}</span>
        <button type="button" onclick={() => (masterPicked = null)} aria-label="清除已选主人">×</button>
      </div>
    {:else}
      <input value={masterQuery} oninput={(event) => (masterQuery = event.currentTarget.value)} placeholder="搜索玩家昵称" />
      {#if masterHits.length > 0}
        <div class="petbond-picker-hits">
          {#each masterHits as hit (hit.id)}
            <button type="button" onclick={() => (masterPicked = hit)}>
              <span>{hit.name}</span>
              <small>{hit.connected ? "在线" : "离线"}</small>
            </button>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
  <span class="petbond-graph-arrow">→ 认养 →</span>
  <div class="petbond-picker">
    <span class="field-label-caption">宠物</span>
    {#if petPicked}
      <div class="petbond-picker-picked">
        <span>{petPicked.name}</span>
        <button type="button" onclick={() => (petPicked = null)} aria-label="清除已选宠物">×</button>
      </div>
    {:else}
      <input value={petQuery} oninput={(event) => (petQuery = event.currentTarget.value)} placeholder="搜索玩家昵称" />
      {#if petHits.length > 0}
        <div class="petbond-picker-hits">
          {#each petHits as hit (hit.id)}
            <button type="button" onclick={() => (petPicked = hit)}>
              <span>{hit.name}</span>
              <small>{hit.connected ? "在线" : "离线"}</small>
            </button>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
  <button type="button" class="primary" disabled={!masterPicked || !petPicked || adding} onclick={addRelation}>添加关系</button>
  <button type="button" onclick={load} disabled={loading}>刷新</button>
</div>
<div class="petbond-graph-filter">
  <div class="petbond-picker">
    <span class="field-label-caption">聚焦玩家</span>
    {#if filterPicked}
      <div class="petbond-picker-picked">
        <span>{filterPicked.name}</span>
        <button type="button" onclick={() => (filterPicked = null)} aria-label="清除已选聚焦玩家">×</button>
      </div>
    {:else}
      <input value={filterQuery} oninput={(event) => (filterQuery = event.currentTarget.value)} placeholder="搜索玩家昵称" />
      {#if filterHits.length > 0}
        <div class="petbond-picker-hits">
          {#each filterHits as hit (hit.id)}
            <button type="button" onclick={() => (filterPicked = hit)}>
              <span>{hit.name}</span>
              <small>{hit.connected ? "在线" : "离线"}</small>
            </button>
          {/each}
        </div>
      {/if}
    {/if}
  </div>
  <label class="petbond-graph-filter-depth">
    <span class="field-label-caption">关系深度</span>
    <input type="number" min="0" max="50" value={filterDepth} oninput={(event) => (filterDepth = Math.max(0, Math.min(50, Number(event.currentTarget.value) || 0)))} />
  </label>
  {#if filtering}
    <button type="button" onclick={() => { filterPicked = null; filterQuery = ""; }}>清除筛选</button>
  {/if}
  <span class="petbond-graph-filter-hint">
    {filtering ? "只显示与该玩家在设定深度内可达的主宠关系链" : "选定玩家后可按关系深度隐藏其余无关的主宠关系"}
  </span>
</div>
<div class="petbond-graph-wrap" bind:clientWidth={svgWidth}>
  <ForceSimulation data={simData} {forces} bind:alpha={simAlpha} onTick={onTick}>
    {#snippet children()}
      <svg
        bind:this={svgEl}
        viewBox={`0 0 ${svgWidth} ${HEIGHT}`}
        class="petbond-graph-svg"
        onpointermove={onSvgPointerMove}
        onpointerup={releaseDrag}
        onpointerleave={releaseDrag}
        onpointerdown={onBackgroundPointerDown}
      >
        <defs>
          <marker id="petbond-arrow" markerWidth="8" markerHeight="8" refX="6" refY="4" orient="auto">
            <path d="M0,0 L8,4 L0,8 Z" />
          </marker>
        </defs>
        {#each tickLinks as link (link.masterId + ">" + link.petId)}
          {@const a = typeof link.source === "object" ? link.source : null}
          {@const b = typeof link.target === "object" ? link.target : null}
          {#if a && b}
            {@const dx = (b.x ?? 0) - (a.x ?? 0)}
            {@const dy = (b.y ?? 0) - (a.y ?? 0)}
            {@const dist = Math.sqrt(dx * dx + dy * dy) || 1}
            {@const ux = dx / dist}
            {@const uy = dy / dist}
            {@const x1 = (a.x ?? 0) + ux * (NODE_RADIUS + 2)}
            {@const y1 = (a.y ?? 0) + uy * (NODE_RADIUS + 2)}
            {@const x2 = (b.x ?? 0) - ux * (NODE_RADIUS + 4)}
            {@const y2 = (b.y ?? 0) - uy * (NODE_RADIUS + 4)}
            {@const mx = ((a.x ?? 0) + (b.x ?? 0)) / 2}
            {@const my = ((a.y ?? 0) + (b.y ?? 0)) / 2}
            <g>
              <line {x1} {y1} {x2} {y2} class="petbond-graph-edge" marker-end="url(#petbond-arrow)" />
              <g transform={`translate(${mx}, ${my})`} class="petbond-graph-edge-label" onclick={() => removeRelation(link.masterId, link.petId)}>
                <title>{link.petTitle ? `称号：${link.petTitle} · 点击解除` : "点击解除关系"}</title>
                <rect x={-11} y={-11} width={22} height={22} rx={11} />
                <text x={0} y={4} text-anchor="middle">×</text>
              </g>
            </g>
          {/if}
        {/each}
        {#each tickNodes as node (node.id)}
          <foreignObject
            x={(node.x ?? svgWidth / 2) - 46}
            y={(node.y ?? HEIGHT / 2) - AVATAR_SIZE / 2 - 4}
            width={92}
            height={AVATAR_SIZE + 26}
            class={`petbond-graph-node${node.id === selectedId ? " selected" : ""}`}
          >
            <div class="petbond-graph-node-inner" onpointerdown={(event) => onNodePointerDown(node.id, event)}>
              <PlayerAvatar player={node} size={AVATAR_SIZE} />
              <span class="petbond-graph-node-label">{node.name}</span>
            </div>
          </foreignObject>
        {/each}
        {#if selectedNode}
          <foreignObject
            x={Math.min(Math.max(4, svgWidth - 300), Math.max(4, (selectedNode.x ?? 0) - 150))}
            y={Math.max(4, (selectedNode.y ?? 0) - AVATAR_SIZE - 74)}
            width={300}
            height={70}
            class="petbond-graph-popover-wrap"
          >
            <div class="petbond-graph-popover">
              <PlayerBadge player={selectedNode} compact />
            </div>
          </foreignObject>
        {/if}
      </svg>
    {/snippet}
  </ForceSimulation>
  {#if displayGraph.nodes.length === 0 && !loading}
    <p class="empty">{filtering ? "没有符合筛选条件的主宠关系，试试调大关系深度或换个玩家" : "暂无主宠关系，先在上方搜索建立一条吧"}</p>
  {/if}
</div>
