import type { Item } from './db.svelte';

export type FamRole = 'parent' | 'partner' | 'sibling';
export const FAM_ROLES: { role: FamRole; label: string; short: string }[] = [
  { role: 'parent', label: 'A is a parent of B', short: 'parent of' },
  { role: 'partner', label: 'A and B are partners', short: 'partner' },
  { role: 'sibling', label: 'A and B are siblings', short: 'sibling' },
];
const AUTO_LABELS = new Set(['', 'family', ...FAM_ROLES.map((r) => r.short)]);

/** Give a family link its role. The graph label follows, unless you wrote your own. */
export function setFamRole(r: Item, role: FamRole | '') {
  r.fam = role;
  if (AUTO_LABELS.has(r.label ?? '')) r.label = FAM_ROLES.find((x) => x.role === role)?.short ?? '';
  r.directed = role === 'parent';
}

export type FNode = { id: number; x: number; y: number; w: number };
export type FPath = { d: string; kind: 'child' | 'partner' | 'sibling' };
export type FLayout = { nodes: FNode[]; paths: FPath[]; minX: number; maxX: number; maxY: number };
export type FamEdges = { parents: [number, number][]; partners: [number, number][]; siblings: [number, number][] }; // parents: [parent, child]

export const NODE_H = 46;
const PARTNER_GAP = 30; // between partners
const SIBLING_GAP = 34; // between family branches
const ROOT_GAP = 70; // between separate families
const ROW = 140;

/**
 * Lay out a family tree. Partners sit side by side. Children hang under their parents.
 * Each branch is as wide as everything below it, so no two branches overlap.
 */
export function layoutFamily(ids: number[], width: (id: number) => number, e: FamEdges): FLayout {
  const inSet = new Set(ids);
  const ok = ([a, b]: [number, number]) => inSet.has(a) && inSet.has(b) && a !== b;
  const parents = e.parents.filter(ok), partners = e.partners.filter(ok), siblings = e.siblings.filter(ok);

  // partners (and partners of partners) form one unit
  const up = new Map<number, number>();
  const find = (x: number): number => { let r = x; while (up.has(r)) r = up.get(r)!; return r; };
  const union = (a: number, b: number) => { const ra = find(a), rb = find(b); if (ra !== rb) up.set(rb, ra); };
  for (const [a, b] of partners) union(a, b);

  const parentsOf = new Map<number, number[]>();
  for (const [p, c] of parents) parentsOf.set(c, [...(parentsOf.get(c) ?? []), p]);

  const units = new Map<number, number[]>(); // unit → people, left to right
  for (const id of ids) { const u = find(id); units.set(u, [...(units.get(u) ?? []), id]); }
  for (const [u, members] of units) units.set(u, [...members].sort((a, b) => +!parentsOf.has(a) - +!parentsOf.has(b)));

  // each unit hangs under one parent unit (the first one found), never under itself or its own descendants
  const under = new Map<number, number>();
  const below = (u: number, of: number) => { let x = of; while (true) { if (x === u) return true; if (!under.has(x)) return false; x = under.get(x)!; } };
  for (const [u, members] of units) {
    search: for (const m of members) {
      for (const p of parentsOf.get(m) ?? []) {
        const pu = find(p);
        if (pu !== u && !below(u, pu)) { under.set(u, pu); break search; }
      }
    }
  }
  const kids = new Map<number, number[]>();
  for (const u of units.keys()) if (under.has(u)) kids.set(under.get(u)!, [...(kids.get(under.get(u)!) ?? []), u]);

  // families with no known parents: siblings stay next to each other
  const roots = [...units.keys()].filter((u) => !under.has(u));
  const g = new Map<number, number>();
  const gfind = (x: number): number => { let r = x; while (g.has(r)) r = g.get(r)!; return r; };
  for (const [a, b] of siblings) {
    const ua = find(a), ub = find(b);
    if (roots.includes(ua) && roots.includes(ub) && gfind(ua) !== gfind(ub)) g.set(gfind(ub), gfind(ua));
  }
  const order = new Map<number, number>();
  roots.forEach((r) => { if (!order.has(gfind(r))) order.set(gfind(r), order.size); });
  roots.sort((a, b) => order.get(gfind(a))! - order.get(gfind(b))!);

  const unitW = (u: number) => units.get(u)!.reduce((n, id) => n + width(id), 0) + PARTNER_GAP * (units.get(u)!.length - 1);
  const kidsW = (u: number) => { const k = kids.get(u) ?? []; return k.reduce((n, x) => n + branchW(x), 0) + SIBLING_GAP * Math.max(0, k.length - 1); };
  const memo = new Map<number, number>();
  const branchW = (u: number): number => {
    if (!memo.has(u)) memo.set(u, Math.max(unitW(u), kidsW(u)));
    return memo.get(u)!;
  };

  const pos = new Map<number, FNode>();
  const place = (u: number, left: number, gen: number) => {
    const bw = branchW(u);
    let x = left + bw / 2 - unitW(u) / 2;
    for (const id of units.get(u)!) {
      pos.set(id, { id, x: x + width(id) / 2, y: gen * ROW + NODE_H / 2, w: width(id) });
      x += width(id) + PARTNER_GAP;
    }
    let cx = left + (bw - kidsW(u)) / 2;
    for (const k of kids.get(u) ?? []) { place(k, cx, gen + 1); cx += branchW(k) + SIBLING_GAP; }
  };
  let cursor = 0;
  roots.forEach((r, i) => {
    place(r, cursor, 0);
    const sameGroup = i + 1 < roots.length && gfind(roots[i + 1]) === gfind(r);
    cursor += branchW(r) + (sameGroup ? SIBLING_GAP : ROOT_GAP);
  });

  // lines
  const paths: FPath[] = [];
  const at = (id: number) => pos.get(id)!;
  for (const [a, b] of partners) {
    const p = at(a), q = at(b);
    const [l, r] = p.x < q.x ? [p, q] : [q, p];
    paths.push({ kind: 'partner', d: `M${l.x + l.w / 2},${l.y} H${r.x - r.w / 2}` });
  }
  const sharedParent = (a: number, b: number) => (parentsOf.get(a) ?? []).some((p) => (parentsOf.get(b) ?? []).includes(p));
  for (const [child, ps] of parentsOf) {
    const c = pos.get(child);
    if (!c) continue;
    const byUnit = new Map<number, number[]>();
    for (const p of ps) byUnit.set(find(p), [...(byUnit.get(find(p)) ?? []), p]);
    for (const people of byUnit.values()) {
      const pp = people.map(at);
      const ax = pp.reduce((n, p) => n + p.x, 0) / pp.length;
      const py = pp[0].y;
      const together = pp.length > 1; // a couple: the line leaves from between them
      const ay = together ? py : py + NODE_H / 2;
      const midY = py + NODE_H / 2 + 34;
      paths.push({ kind: 'child', d: `M${ax},${ay} V${midY} H${c.x} V${c.y - NODE_H / 2}` });
    }
  }
  for (const [a, b] of siblings) {
    if (sharedParent(a, b)) continue;
    const p = at(a), q = at(b);
    const top = Math.min(p.y, q.y) - NODE_H / 2;
    paths.push({ kind: 'sibling', d: `M${p.x},${p.y - NODE_H / 2} C${p.x},${top - 34} ${q.x},${top - 34} ${q.x},${q.y - NODE_H / 2}` });
  }

  const nodes = [...pos.values()];
  return {
    nodes, paths,
    minX: Math.min(0, ...nodes.map((n) => n.x - n.w / 2)),
    maxX: Math.max(0, ...nodes.map((n) => n.x + n.w / 2)),
    maxY: Math.max(0, ...nodes.map((n) => n.y + NODE_H / 2)),
  };
}
