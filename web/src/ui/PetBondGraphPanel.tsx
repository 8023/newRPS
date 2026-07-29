import { useEffect, useRef, useState } from "react";
import type { PublicPlayer } from "../shared/types";
import { ask } from "../lib/rpc";
import { PlayerAvatar, PlayerBadge } from "./AppViews";
import { AdminSectionHeader } from "./AdminViews";

type PetBondGraphEdge = {
  masterId: string;
  petId: string;
  petTitle?: string;
  createdAt: number;
};

type PetBondGraphData = { nodes: PublicPlayer[]; edges: PetBondGraphEdge[] };

type PlayerHit = { id: string; name: string; connected: boolean };

type SimNode = { x: number; y: number; vx: number; vy: number; fx?: number; fy?: number };

const WIDTH = 720;
const HEIGHT = 460;
const NODE_RADIUS = 18;
const AVATAR_SIZE = 36;
const DRAG_CLICK_THRESHOLD = 5;

/** 力导向图节点/边不受 React state 频繁重渲染影响：物理状态放 ref，动画帧只用 tick 触发重绘。 */
function usePetBondSimulation(graph: PetBondGraphData) {
  const simRef = useRef<Record<string, SimNode>>({});
  const graphRef = useRef(graph);
  const [, setTick] = useState(0);

  useEffect(() => {
    graphRef.current = graph;
  }, [graph]);

  // 节点集合变化时补齐新节点的初始位置（环形分布），清掉已消失节点的模拟状态。
  useEffect(() => {
    const sim = simRef.current;
    const ids = new Set(graph.nodes.map((n) => n.id));
    for (const id of Object.keys(sim)) {
      if (!ids.has(id)) delete sim[id];
    }
    graph.nodes.forEach((node, index) => {
      if (sim[node.id]) return;
      const angle = (index / Math.max(1, graph.nodes.length)) * Math.PI * 2;
      sim[node.id] = {
        x: WIDTH / 2 + Math.cos(angle) * 150 + (Math.random() - 0.5) * 30,
        y: HEIGHT / 2 + Math.sin(angle) * 150 + (Math.random() - 0.5) * 30,
        vx: 0, vy: 0
      };
    });
  }, [graph.nodes]);

  useEffect(() => {
    let alive = true;
    let raf = 0;
    function step() {
      if (!alive) return;
      const sim = simRef.current;
      const current = graphRef.current;
      const ids = Object.keys(sim);
      for (const id of ids) {
        const node = sim[id];
        let fx = 0;
        let fy = 0;
        for (const otherId of ids) {
          if (otherId === id) continue;
          const other = sim[otherId];
          const dx = node.x - other.x;
          const dy = node.y - other.y;
          const distSq = Math.max(500, dx * dx + dy * dy);
          const dist = Math.sqrt(distSq);
          const force = 7200 / distSq;
          fx += (dx / dist) * force;
          fy += (dy / dist) * force;
        }
        // 向心力，避免节点飘出画布
        fx += (WIDTH / 2 - node.x) * 0.0025;
        fy += (HEIGHT / 2 - node.y) * 0.0025;
        node.vx = (node.vx + fx) * 0.85;
        node.vy = (node.vy + fy) * 0.85;
      }
      for (const edge of current.edges) {
        const a = sim[edge.masterId];
        const b = sim[edge.petId];
        if (!a || !b) continue;
        const dx = b.x - a.x;
        const dy = b.y - a.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const springForce = (dist - 170) * 0.018;
        const ux = dx / dist;
        const uy = dy / dist;
        a.vx += ux * springForce;
        a.vy += uy * springForce;
        b.vx -= ux * springForce;
        b.vy -= uy * springForce;
      }
      for (const id of ids) {
        const node = sim[id];
        if (node.fx !== undefined && node.fy !== undefined) {
          node.x = node.fx;
          node.y = node.fy;
          node.vx = 0;
          node.vy = 0;
          continue;
        }
        node.x = Math.min(WIDTH - AVATAR_SIZE, Math.max(AVATAR_SIZE, node.x + node.vx));
        node.y = Math.min(HEIGHT - AVATAR_SIZE - 16, Math.max(AVATAR_SIZE, node.y + node.vy));
      }
      setTick((t) => (t + 1) % 1_000_000);
      raf = requestAnimationFrame(step);
    }
    raf = requestAnimationFrame(step);
    return () => {
      alive = false;
      cancelAnimationFrame(raf);
    };
    // 只在挂载时起一次循环，物理状态和最新的 graph 都从 ref 读取，不需要重启。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return simRef;
}

function PlayerPicker({
  label, query, onQuery, hits, picked, onPick
}: {
  label: string;
  query: string;
  onQuery: (value: string) => void;
  hits: PlayerHit[];
  picked: PlayerHit | null;
  onPick: (player: PlayerHit | null) => void;
}) {
  return (
    <div className="petbond-picker">
      <span className="field-label-caption">{label}</span>
      {picked ? (
        <div className="petbond-picker-picked">
          <span>{picked.name}</span>
          <button type="button" onClick={() => onPick(null)} aria-label={`清除已选${label}`}>×</button>
        </div>
      ) : (
        <>
          <input
            value={query}
            onChange={(event) => onQuery(event.target.value)}
            placeholder="搜索玩家昵称"
          />
          {hits.length > 0 && (
            <div className="petbond-picker-hits">
              {hits.map((hit) => (
                <button type="button" key={hit.id} onClick={() => onPick(hit)}>
                  <span>{hit.name}</span>
                  <small>{hit.connected ? "在线" : "离线"}</small>
                </button>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

export function PetBondGraphPanel({ onError }: { onError: (message: string) => void }) {
  const [graph, setGraph] = useState<PetBondGraphData>({ nodes: [], edges: [] });
  const [loading, setLoading] = useState(false);
  const [adding, setAdding] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const [masterQuery, setMasterQuery] = useState("");
  const [masterHits, setMasterHits] = useState<PlayerHit[]>([]);
  const [masterPicked, setMasterPicked] = useState<PlayerHit | null>(null);
  const [petQuery, setPetQuery] = useState("");
  const [petHits, setPetHits] = useState<PlayerHit[]>([]);
  const [petPicked, setPetPicked] = useState<PlayerHit | null>(null);

  const simRef = usePetBondSimulation(graph);
  const dragIdRef = useRef<string | null>(null);
  const dragMovedRef = useRef(false);
  const dragStartRef = useRef({ x: 0, y: 0 });
  const dragOffsetRef = useRef({ x: 0, y: 0 });
  const svgRef = useRef<SVGSVGElement | null>(null);

  async function load() {
    setLoading(true);
    try {
      const result = await ask<PetBondGraphData>("admin:petBondGraph", {});
      setGraph({ nodes: result.nodes || [], edges: result.edges || [] });
    } catch (error) {
      onError(error instanceof Error ? error.message : "加载主宠关系失败");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, []);

  function searchPlayers(keyword: string, setHits: (hits: PlayerHit[]) => void) {
    if (!keyword.trim()) {
      setHits([]);
      return;
    }
    ask<{ players?: Array<{ id: string; name: string; connected: boolean }> }>(
      "admin:listPlayers",
      { keyword: keyword.trim(), limit: 8 }
    ).then((result) => {
      setHits((result.players || []).map((p) => ({ id: p.id, name: p.name, connected: p.connected })));
    }).catch(() => {
      // 搜索联想失败不打断输入，静默忽略。
    });
  }

  useEffect(() => {
    const handle = setTimeout(() => searchPlayers(masterQuery, setMasterHits), 250);
    return () => clearTimeout(handle);
  }, [masterQuery]);

  useEffect(() => {
    const handle = setTimeout(() => searchPlayers(petQuery, setPetHits), 250);
    return () => clearTimeout(handle);
  }, [petQuery]);

  async function addRelation() {
    if (!masterPicked || !petPicked) return;
    if (masterPicked.id === petPicked.id) {
      onError("不能选择同一个人");
      return;
    }
    setAdding(true);
    try {
      const result = await ask<PetBondGraphData>("admin:petBondAdd", { masterId: masterPicked.id, petId: petPicked.id });
      setGraph({ nodes: result.nodes || [], edges: result.edges || [] });
      setMasterPicked(null);
      setPetPicked(null);
      setMasterQuery("");
      setPetQuery("");
      onError("已添加主宠关系");
    } catch (error) {
      onError(error instanceof Error ? error.message : "添加关系失败");
    } finally {
      setAdding(false);
    }
  }

  async function removeRelation(masterId: string, petId: string) {
    if (!window.confirm("确定解除这条主宠关系吗？")) return;
    try {
      const result = await ask<PetBondGraphData>("admin:petBondRemove", { masterId, petId });
      setGraph({ nodes: result.nodes || [], edges: result.edges || [] });
      onError("已解除主宠关系");
    } catch (error) {
      onError(error instanceof Error ? error.message : "解除关系失败");
    }
  }

  function toSvgPoint(clientX: number, clientY: number) {
    const svg = svgRef.current;
    if (!svg) return { x: WIDTH / 2, y: HEIGHT / 2 };
    const screenMatrix = svg.getScreenCTM();
    if (screenMatrix) {
      const point = svg.createSVGPoint();
      point.x = clientX;
      point.y = clientY;
      const transformed = point.matrixTransform(screenMatrix.inverse());
      return { x: transformed.x, y: transformed.y };
    }
    const rect = svg.getBoundingClientRect();
    return {
      x: ((clientX - rect.left) / Math.max(1, rect.width)) * WIDTH,
      y: ((clientY - rect.top) / Math.max(1, rect.height)) * HEIGHT
    };
  }

  function onNodePointerDown(playerId: string, event: React.PointerEvent) {
    event.currentTarget.setPointerCapture(event.pointerId);
    dragIdRef.current = playerId;
    dragMovedRef.current = false;
    dragStartRef.current = { x: event.clientX, y: event.clientY };
    const node = simRef.current[playerId];
    if (node) {
      const point = toSvgPoint(event.clientX, event.clientY);
      dragOffsetRef.current = { x: node.x - point.x, y: node.y - point.y };
      node.fx = node.x;
      node.fy = node.y;
    }
  }

  function onSvgPointerMove(event: React.PointerEvent) {
    if (!dragIdRef.current) return;
    const node = simRef.current[dragIdRef.current];
    if (!node) return;
    const dx = event.clientX - dragStartRef.current.x;
    const dy = event.clientY - dragStartRef.current.y;
    if (Math.abs(dx) + Math.abs(dy) > DRAG_CLICK_THRESHOLD) dragMovedRef.current = true;
    const point = toSvgPoint(event.clientX, event.clientY);
    node.fx = point.x + dragOffsetRef.current.x;
    node.fy = point.y + dragOffsetRef.current.y;
  }

  function releaseDrag() {
    const draggedId = dragIdRef.current;
    if (draggedId) {
      const node = simRef.current[draggedId];
      if (node) {
        node.fx = undefined;
        node.fy = undefined;
      }
      if (!dragMovedRef.current) {
        // 没有明显移动视为一次点击：选中/取消选中该节点，展示详细信息气泡。
        setSelectedId((old) => (old === draggedId ? null : draggedId));
      }
    }
    dragIdRef.current = null;
  }

  function onBackgroundPointerDown(event: React.PointerEvent<SVGSVGElement>) {
    if (event.target === event.currentTarget) setSelectedId(null);
  }

  const selectedNode = selectedId ? graph.nodes.find((n) => n.id === selectedId) || null : null;
  const selectedPos = selectedId ? simRef.current[selectedId] : null;

  return (
    <>
      <AdminSectionHeader
        title="主宠关系"
        subtitle="力导向关系图：拖动节点调整布局；点击节点头像查看详细资料，点击连线上的 × 可直接解除关系；下方搜索建立新关系不受在线限制。"
      />
      <div className="admin-preview-card">
        <span>概况</span>
        <p>{graph.nodes.length} 位玩家 · {graph.edges.length} 条主宠关系{loading ? "（刷新中…）" : ""}</p>
      </div>
      <div className="petbond-graph-add">
        <PlayerPicker label="主人" query={masterQuery} onQuery={setMasterQuery} hits={masterHits} picked={masterPicked} onPick={setMasterPicked} />
        <span className="petbond-graph-arrow">→ 认养 →</span>
        <PlayerPicker label="宠物" query={petQuery} onQuery={setPetQuery} hits={petHits} picked={petPicked} onPick={setPetPicked} />
        <button type="button" className="primary" disabled={!masterPicked || !petPicked || adding} onClick={addRelation}>
          添加关系
        </button>
        <button type="button" onClick={load} disabled={loading}>刷新</button>
      </div>
      <div className="petbond-graph-wrap">
        <svg
          ref={svgRef}
          viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
          className="petbond-graph-svg"
          onPointerMove={onSvgPointerMove}
          onPointerUp={releaseDrag}
          onPointerLeave={releaseDrag}
          onPointerDown={onBackgroundPointerDown}
        >
          <defs>
            <marker id="petbond-arrow" markerWidth="8" markerHeight="8" refX="6" refY="4" orient="auto">
              <path d="M0,0 L8,4 L0,8 Z" />
            </marker>
          </defs>
          {graph.edges.map((edge) => {
            const a = simRef.current[edge.masterId];
            const b = simRef.current[edge.petId];
            if (!a || !b) return null;
            const dx = b.x - a.x;
            const dy = b.y - a.y;
            const dist = Math.sqrt(dx * dx + dy * dy) || 1;
            const ux = dx / dist;
            const uy = dy / dist;
            const x1 = a.x + ux * (NODE_RADIUS + 2);
            const y1 = a.y + uy * (NODE_RADIUS + 2);
            const x2 = b.x - ux * (NODE_RADIUS + 4);
            const y2 = b.y - uy * (NODE_RADIUS + 4);
            const mx = (a.x + b.x) / 2;
            const my = (a.y + b.y) / 2;
            return (
              <g key={`${edge.masterId}>${edge.petId}`}>
                <line x1={x1} y1={y1} x2={x2} y2={y2} className="petbond-graph-edge" markerEnd="url(#petbond-arrow)" />
                <g
                  transform={`translate(${mx}, ${my})`}
                  className="petbond-graph-edge-label"
                  onClick={() => removeRelation(edge.masterId, edge.petId)}
                >
                  <title>{edge.petTitle ? `称号：${edge.petTitle} · 点击解除` : "点击解除关系"}</title>
                  <rect x={-11} y={-11} width={22} height={22} rx={11} />
                  <text x={0} y={4} textAnchor="middle">×</text>
                </g>
              </g>
            );
          })}
          {graph.nodes.map((node) => {
            const pos = simRef.current[node.id] || { x: WIDTH / 2, y: HEIGHT / 2 };
            return (
              <foreignObject
                key={node.id}
                x={pos.x - 46}
                y={pos.y - AVATAR_SIZE / 2 - 4}
                width={92}
                height={AVATAR_SIZE + 26}
                className={`petbond-graph-node${node.id === selectedId ? " selected" : ""}`}
              >
                <div
                  className="petbond-graph-node-inner"
                  onPointerDown={(event) => onNodePointerDown(node.id, event)}
                >
                  <PlayerAvatar player={node} size={AVATAR_SIZE} />
                  <span className="petbond-graph-node-label">{node.name}</span>
                </div>
              </foreignObject>
            );
          })}
          {selectedNode && selectedPos && (
            <foreignObject
              x={Math.min(WIDTH - 300, Math.max(4, selectedPos.x - 150))}
              y={Math.max(4, selectedPos.y - AVATAR_SIZE - 74)}
              width={300}
              height={70}
              className="petbond-graph-popover-wrap"
            >
              <div className="petbond-graph-popover">
                <PlayerBadge player={selectedNode} compact />
              </div>
            </foreignObject>
          )}
        </svg>
        {graph.nodes.length === 0 && !loading && <p className="empty">暂无主宠关系，先在上方搜索建立一条吧</p>}
      </div>
    </>
  );
}
