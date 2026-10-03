<script lang="ts">
  import { db, type Kind } from '../lib/db.svelte';
  import { nav, openItem, openScene } from '../lib/nav.svelte';
  import ProgressBar from '../lib/ProgressBar.svelte';
  import { sceneLabel, sceneWords } from '../lib/scenes';
  import { dailyGoal, streak, wordsToday } from '../lib/progress';
  import { findConflicts, openConflicts } from '../lib/plotholes';
  import { findGaps, openGaps } from '../lib/consistency';

  let text = $state('');

  const goal = $derived(dailyGoal());
  const today = $derived(wordsToday());
  const run = $derived(streak());
  const total = $derived(db.lists.scene.reduce((a, s) => a + sceneWords(s), 0));
  const holes = $derived(openConflicts(findConflicts().conflicts).length);
  const gaps = $derived(openGaps(findGaps()).length);

  type Recent = { kind: Kind; id: number; label: string; what: string; at: string };
  const WHAT: Partial<Record<Kind, string>> = { scene: 'scene', character: 'character', location: 'place', note: 'lore note', faction: 'faction', item: 'item', event: 'event' };
  const labelOf = (kind: Kind, i: any): string =>
    kind === 'scene' ? sceneLabel(i) : kind === 'note' || kind === 'event' ? i.title || 'Untitled' : i.name || 'Unnamed';

  const recent = $derived.by(() => {
    const all: Recent[] = [];
    for (const kind of Object.keys(WHAT) as Kind[]) {
      for (const i of db.lists[kind]) all.push({ kind, id: i.id, label: labelOf(kind, i), what: WHAT[kind]!, at: i.updatedAt });
    }
    return all.sort((a, b) => (a.at < b.at ? 1 : a.at > b.at ? -1 : 0)).slice(0, 8);
  });
  // the scene you touched last: the "continue writing" button opens it
  const last = $derived(recent.find((r) => r.kind === 'scene'));

  const todos = $derived([...db.lists.todo].sort((a, b) => a.id - b.id));
  const openTodos = $derived(todos.filter((t) => !t.done).length);

  const ago = (iso: string): string => {
    const d = new Date(iso);
    if (isNaN(+d)) return '';
    const days = Math.floor((Date.now() - +d) / 864e5);
    if (days <= 0 && new Date().toDateString() === d.toDateString()) return 'today';
    if (days <= 1) return 'yesterday';
    return days < 7 ? `${days} days ago` : d.toLocaleDateString();
  };
  const open = (r: Recent) => (r.kind === 'scene' ? openScene(r.id) : openItem(r.kind, r.id));

  async function add() {
    const t = text.trim();
    if (!t) return;
    text = '';
    await db.save('todo', { text: t, done: false });
  }
  const toggle = (t: any) => { t.done = !t.done; db.queue('todo', t); };
  const edit = (t: any) => db.queue('todo', t);
  const clearDone = async () => { for (const t of todos.filter((x) => x.done)) await db.remove('todo', t.id, false); };
  const n = (v: number) => v.toLocaleString();
</script>

<section class="pane">
  <h2>Dashboard</h2>

  <div class="cols">
    <div class="card">
      <h3>Today</h3>
      <p style="margin:.2rem 0 .4rem"><b>{n(today)}</b>{goal ? ` / ${n(goal)} words` : ' words written'}{goal && today >= goal ? ' · goal reached' : ''}</p>
      <ProgressBar value={today} {goal} />
      {#if !goal}<p class="muted" style="margin:.4rem 0 0">No daily goal yet. <button class="link" onclick={() => (nav.tab = 'stats')}>Set one</button></p>{/if}
      <p class="muted" style="margin:.6rem 0 0">Streak: <b>{run.current}</b> day{run.current === 1 ? '' : 's'} · Best: <b>{run.best}</b> · All text: <b>{n(total)}</b> words</p>
      <div style="margin-top:.8rem">
        {#if last}
          <button class="primary" onclick={() => openScene(last.id)}>Continue writing</button>
          <span class="muted"> {last.label}</span>
        {:else}
          <button class="primary" onclick={() => (nav.tab = 'write')}>Start writing</button>
        {/if}
      </div>
    </div>

    <div class="card">
      <h3>Needs attention</h3>
      <button class="row-link" onclick={() => (nav.tab = 'checks')}>
        <span class="chip" class:bad={holes > 0}>{holes}</span><span>plot hole{holes === 1 ? '' : 's'}</span>
      </button>
      <button class="row-link" onclick={() => (nav.tab = 'checks')}>
        <span class="chip">{gaps}</span><span>unfinished thing{gaps === 1 ? '' : 's'}</span>
      </button>
      <button class="row-link" onclick={() => (nav.tab = 'checks')}>
        <span class="chip">{db.lists.comment.filter((c) => !c.resolved).length}</span><span>open comment{db.lists.comment.filter((c) => !c.resolved).length === 1 ? '' : 's'}</span>
      </button>
      <p class="muted" style="margin:.5rem 0 0">Plot holes and unfinished things are on the Checks tab.</p>
    </div>
  </div>

  <div class="cols">
    <div class="card">
      <h3>Recent work</h3>
      {#each recent as r (r.kind + r.id)}
        <button class="row-link" onclick={() => open(r)}>
          <span class="chip">{r.what}</span><span class="t">{r.label}</span><span class="muted">{ago(r.at)}</span>
        </button>
      {:else}
        <div class="empty" style="padding:.6rem 0">Nothing yet. Start in the Write tab.</div>
      {/each}
    </div>

    <div class="card">
      <div class="head">
        <h3 style="margin:0">To-do{openTodos ? ` · ${openTodos}` : ''}</h3>
        {#if todos.some((t) => t.done)}<button class="small" onclick={clearDone}>Clear done</button>{/if}
      </div>
      <form class="row" onsubmit={(e) => { e.preventDefault(); add(); }}>
        <input placeholder="Add a to-do, then press Enter" bind:value={text} />
        <button class="small" type="submit">Add</button>
      </form>
      {#each todos as t (t.id)}
        <div class="todo" class:done={t.done}>
          <input type="checkbox" checked={t.done} onchange={() => toggle(t)} aria-label="Done" />
          <input class="txt" bind:value={t.text} oninput={() => edit(t)} />
          <button class="small danger" title="Delete" onclick={() => db.remove('todo', t.id, false)}>✕</button>
        </div>
      {:else}
        <div class="empty" style="padding:.6rem 0">No to-dos. Write down what you want to fix or add.</div>
      {/each}
    </div>
  </div>
</section>

<style>
  .cols { display: grid; grid-template-columns: repeat(auto-fit, minmax(22rem, 1fr)); gap: 0 1rem; align-items: start; }
  .row-link { display: flex; align-items: center; gap: .5rem; width: 100%; text-align: left; background: none; border: 1px solid transparent; padding: .3rem .4rem; }
  .row-link:hover { background: var(--panel-2); border-color: var(--line); }
  .row-link .t { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row-link .chip { margin: 0; }
  .chip.bad { background: var(--danger); border-color: var(--danger); color: #fff; }
  .link { background: none; border: 0; padding: 0; color: var(--accent); text-decoration: underline; }
  form.row { margin: .5rem 0; }
  .todo { display: flex; align-items: center; gap: .5rem; padding: .15rem 0; }
  .todo input[type='checkbox'] { width: auto; flex: 0 0 auto; }
  .todo .txt { border-color: transparent; background: none; }
  .todo .txt:focus { border-color: var(--accent); background: var(--bg); }
  .todo.done .txt { text-decoration: line-through; color: var(--muted); }
</style>
