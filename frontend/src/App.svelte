<script lang="ts">
  import { onMount } from 'svelte';
  import { db } from './lib/db.svelte';
  import { nav } from './lib/nav.svelte';
  import Search from './lib/Search.svelte';
  import CalendarEditor from './lib/CalendarEditor.svelte';
  import NameGenerator from './lib/NameGenerator.svelte';
  import Projects from './lib/Projects.svelte';
  import DataFolder from './lib/DataFolder.svelte';
  import UndoToast from './lib/UndoToast.svelte';
  import { migrateChapters } from './lib/scenes';
  import { currentTheme, setTheme, type Theme } from './lib/theme';
  import Dashboard from './views/Dashboard.svelte';
  import Write from './views/Write.svelte';
  import Board from './views/Board.svelte';
  import Characters from './views/Characters.svelte';
  import Relationships from './views/Relationships.svelte';
  import World from './views/World.svelte';
  import Factions from './views/Factions.svelte';
  import Items from './views/Items.svelte';
  import FamilyTree from './views/FamilyTree.svelte';
  import Checks from './views/Checks.svelte';
  import Statistics from './views/Statistics.svelte';
  import { findConflicts, openConflicts } from './lib/plotholes';
  import Maps from './views/Maps.svelte';
  import Timeline from './views/Timeline.svelte';
  import Export from './views/Export.svelte';

  const tabs = [
    ['home', 'Dashboard'],
    ['write', 'Write'],
    ['board', 'Board'],
    ['characters', 'Characters'],
    ['relationships', 'Relationships'],
    ['world', 'World'],
    ['factions', 'Factions'],
    ['items', 'Items'],
    ['family', 'Family tree'],
    ['maps', 'Maps & Paths'],
    ['timeline', 'Timeline'],
    ['checks', 'Checks'],
    ['stats', 'Statistics'],
    ['export', 'Export & Backup'],
  ] as const;

  // the number of open plot holes shows next to the Checks tab
  const holes = $derived(db.ready ? openConflicts(findConflicts().conflicts).length : 0);

  // the width of the list on the left of each screen, kept between runs
  const SIDE_KEY = 'typewriter-side-width', SIDE_MIN = 180, SIDE_MAX = 640, SIDE_DEFAULT = 260;
  const clampSide = (n: number) => Math.min(SIDE_MAX, Math.max(SIDE_MIN, Math.round(n)));
  let sideW = $state(SIDE_DEFAULT);
  try { const v = Number(localStorage.getItem(SIDE_KEY)); if (v) sideW = clampSide(v); } catch { /* storage may be blocked */ }
  $effect(() => {
    document.documentElement.style.setProperty('--side-w', sideW + 'px');
    try { localStorage.setItem(SIDE_KEY, String(sideW)); } catch { /* storage may be blocked */ }
  });
  const hasSide = $derived(!['home', 'board', 'checks', 'stats', 'export'].includes(nav.tab));
  let dragging = $state(false);
  function startResize(e: PointerEvent) {
    const bar = e.currentTarget as HTMLElement;
    const left = (bar.parentElement as HTMLElement).getBoundingClientRect().left;
    bar.setPointerCapture?.(e.pointerId);
    dragging = true;
    const move = (ev: PointerEvent) => (sideW = clampSide(ev.clientX - left));
    const stop = () => { dragging = false; bar.removeEventListener('pointermove', move); bar.removeEventListener('pointerup', stop); bar.removeEventListener('pointercancel', stop); };
    bar.addEventListener('pointermove', move); bar.addEventListener('pointerup', stop); bar.addEventListener('pointercancel', stop);
  }
  function resizeKey(e: KeyboardEvent) {
    if (e.key === 'ArrowLeft') { sideW = clampSide(sideW - 20); e.preventDefault(); }
    else if (e.key === 'ArrowRight') { sideW = clampSide(sideW + 20); e.preventDefault(); }
  }

  let error = $state('');
  let theme = $state<Theme>(currentTheme());
  function toggleTheme() { theme = theme === 'dark' ? 'light' : 'dark'; setTheme(theme); }

  onMount(() => { db.load().then(migrateChapters).catch((e) => (error = String(e))); });
</script>

<svelte:window onbeforeunload={() => db.flush()} onpagehide={() => db.flush()} onkeydown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") { e.preventDefault(); nav.searchOpen = !nav.searchOpen; } }} />
<Search />
<UndoToast />
{#if nav.calendar}<CalendarEditor />{/if}
{#if nav.names}<NameGenerator />{/if}
{#if nav.projects}<Projects />{/if}
{#if nav.dataFolder}<DataFolder />{/if}

<div class="app" class:focus={nav.focus}>
  <nav class="nav">
    <div class="brand"><img src="/logo.svg" alt="" />Typewriter</div>
    <button class="project-btn" title="Switch or add a project (a book or series)" onclick={() => (nav.projects = true)}>▾ {db.project || 'Projects'}</button>
    <button class="search-btn" onclick={() => (nav.searchOpen = true)}>Search <kbd>Ctrl+K</kbd></button>
    {#each tabs as [id, label]}
      <button class:on={nav.tab === id} onclick={() => (nav.tab = id)}>{label}{#if id === 'checks' && holes}<span class="count">{holes}</span>{/if}</button>
    {/each}
    <div class="foot">
      <button onclick={toggleTheme}>{theme === 'dark' ? '☀ Light theme' : '☾ Dark theme'}</button>
      Data file:<br />{db.dataPath}
      {#if db.folderProblem}<div><button class="small danger" title={db.folderProblem} onclick={() => (nav.dataFolder = true)}>⚠ Data folder problem</button></div>{/if}
      <div><button class="small" title="Keep your data in another folder" onclick={() => (nav.dataFolder = true)}>Change folder…</button></div>
      <div class="ver">Typewriter {db.version}</div></div>
  </nav>
  <main class="main">
    {#if hasSide && !nav.focus && db.ready}
      <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
      <div class="resizer" class:drag={dragging} role="separator" aria-orientation="vertical" aria-label="Resize the list on the left. Drag, or use the left and right arrow keys. Double-click to reset."
        aria-valuenow={sideW} aria-valuemin={SIDE_MIN} aria-valuemax={SIDE_MAX} tabindex="0" title="Drag to resize. Double-click to reset."
        onpointerdown={startResize} onkeydown={resizeKey} ondblclick={() => (sideW = SIDE_DEFAULT)}></div>
    {/if}
    {#if error}
      <div class="pane">Could not open the database: {error}</div>
    {:else if !db.ready}
      <div class="pane muted">Loading…</div>
    {:else if nav.tab === 'home'}<Dashboard />
    {:else if nav.tab === 'write'}<Write />
    {:else if nav.tab === 'board'}<Board />
    {:else if nav.tab === 'characters'}<Characters />
    {:else if nav.tab === 'relationships'}<Relationships />
    {:else if nav.tab === 'world'}<World />
    {:else if nav.tab === 'factions'}<Factions />
    {:else if nav.tab === 'items'}<Items />
    {:else if nav.tab === 'family'}<FamilyTree />
    {:else if nav.tab === 'maps'}<Maps />
    {:else if nav.tab === 'timeline'}<Timeline />
    {:else if nav.tab === 'checks'}<Checks />
    {:else if nav.tab === 'stats'}<Statistics />
    {:else if nav.tab === 'export'}<Export />
    {/if}
  </main>
</div>
