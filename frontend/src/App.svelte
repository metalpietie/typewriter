<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { settle } from './lib/backupReminder';
  import { db } from './lib/db.svelte';
  import { nav, type Tab } from './lib/nav.svelte';
  import { motion, applyMotion, cycleLevel, installRipple, animatePage, fadeWindow } from './lib/motion.svelte';
  import { LABEL } from './lib/motion-core';
  import Search from './lib/Search.svelte';
  import CalendarEditor from './lib/CalendarEditor.svelte';
  import NameGenerator from './lib/NameGenerator.svelte';
  import Projects from './lib/Projects.svelte';
  import Bookshelf from './lib/Bookshelf.svelte';
  import DataFolder from './lib/DataFolder.svelte';
  import UndoToast from './lib/UndoToast.svelte';
  import PomodoroTimer from './lib/PomodoroTimer.svelte';
  import PomodoroAlert from './lib/PomodoroAlert.svelte';
  import { migrateChapters } from './lib/scenes';
  import { currentTheme, setTheme, type Theme } from './lib/theme';
  import Dashboard from './views/Dashboard.svelte';
  import Draft from './views/Draft.svelte';
  import Write from './views/Write.svelte';
  import Board from './views/Board.svelte';
  import Arcs from './views/Arcs.svelte';
  import Glossary from './views/Glossary.svelte';
  import Characters from './views/Characters.svelte';
  import Relationships from './views/Relationships.svelte';
  import World from './views/World.svelte';
  import Factions from './views/Factions.svelte';
  import Items from './views/Items.svelte';
  import FamilyTree from './views/FamilyTree.svelte';
  import Checks from './views/Checks.svelte';
  import Statistics from './views/Statistics.svelte';
  import { findConflicts, openConflicts } from './lib/plotholes';
  import { has, menu as projectMenu, tabOn } from './lib/projecttype.svelte';
  import ProjectSettings from './lib/ProjectSettings.svelte';
  import Maps from './views/Maps.svelte';
  import Timeline from './views/Timeline.svelte';
  import Export from './views/Export.svelte';
  import Sources from './views/Sources.svelte';
  import Ideas from './views/Ideas.svelte';

  // The menu comes from the type of the project: first a tab with no heading, then groups. Each group has a heading.
  const menu = $derived(projectMenu());
  // a tab of a tool that was switched off is not opened (for example after the type changed)
  $effect(() => { if (db.ready && !tabOn(nav.tab)) nav.tab = 'home'; });

  // the number of open plot holes shows next to the Checks tab
  const holes = $derived(db.ready && has('characters') ? openConflicts(findConflicts().conflicts).length : 0);

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

  // Motion: ripples on buttons, a menu highlight that glides, pages that arrive, a fade for focus mode.
  applyMotion();
  onMount(() => installRipple());
  let navEl = $state<HTMLElement>();
  let mainEl = $state<HTMLElement>();
  let appEl = $state<HTMLElement>();
  let pill = $state({ y: 0, h: 0, show: false, glide: false });
  function placePill() {
    const on = navEl?.querySelector<HTMLElement>('button.on');
    if (!on) { pill.show = false; return; }
    const y = on.offsetTop, h = on.offsetHeight;
    if (!pill.show || y !== pill.y || h !== pill.h) pill = { y, h, show: true, glide: pill.show }; // the first place is set without gliding
  }
  $effect(() => { nav.tab; db.ready; tick().then(placePill); });
  onMount(() => {
    const watch = new MutationObserver(placePill); // something above moved the items, for example the timer card
    if (navEl) watch.observe(navEl, { childList: true, subtree: true });
    window.addEventListener('resize', placePill);
    return () => { watch.disconnect(); window.removeEventListener('resize', placePill); };
  });
  let shownTab = '';
  $effect(() => {
    const t = nav.tab;
    if (!db.ready || t === shownTab) return;
    shownTab = t;
    tick().then(() => animatePage(mainEl ?? null));
  });
  let seenFocus = nav.focus;
  $effect(() => { const f = nav.focus; if (f !== seenFocus) { seenFocus = f; tick().then(() => fadeWindow(appEl ?? null)); } });

  let error = $state('');
  let theme = $state<Theme>(currentTheme());
  function toggleTheme() { theme = theme === 'dark' ? 'light' : 'dark'; setTheme(theme); }

  // The window is closing: save what is waiting, then tell the app that it may close.
  // a phone changed some notes: read them again, after saving what is waiting
  onMount(() => Events.On('typewriter:synced', async () => { db.flush(); await db.load(); }));
  onMount(() => Events.On('typewriter:flush', async () => { try { await settle(); } finally { Events.Emit('typewriter:flushed'); } }));
  onMount(() => { db.load().then(migrateChapters).catch((e) => (error = String(e))); });
</script>

<svelte:window onbeforeunload={() => db.flush()} onpagehide={() => db.flush()} onkeydown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") { e.preventDefault(); nav.searchOpen = !nav.searchOpen; } }} />
<Search />
<UndoToast />
<PomodoroAlert />
{#if nav.calendar}<CalendarEditor />{/if}
{#if nav.names}<NameGenerator />{/if}
{#if nav.shelf}<Bookshelf />{/if}
{#if nav.projects}<Projects />{/if}
{#if nav.projectSettings}<ProjectSettings />{/if}
{#if nav.dataFolder}<DataFolder />{/if}

<div class="app" class:focus={nav.focus} bind:this={appEl}>
  <nav class="nav" bind:this={navEl}>
    <span class="navpill" class:show={pill.show} class:glide={pill.glide} style:transform="translateY({pill.y}px)" style:height="{pill.h}px" aria-hidden="true"></span>
    <div class="brand"><img src="/logo.svg" alt="" />Typewriter</div>
    <button class="project-btn" title="Switch or add a project (a book or series)" onclick={() => (nav.projects = true)}>▾ {db.project || 'Projects'}</button>
    <button class="shelf-btn" title="Show all projects and books on the bookshelf" onclick={() => (nav.shelf = true)}>▤ Bookshelf</button>
    <PomodoroTimer />
    <button class="search-btn" onclick={() => (nav.searchOpen = true)}>Search <kbd>Ctrl+K</kbd></button>
    {#each menu as group}
      {#if 'title' in group}<div class="group">{group.title}</div>{/if}
      {#each group.items as [id, label]}
        <button class:on={nav.tab === id} onclick={() => (nav.tab = id as Tab)}>{label}{#if id === 'checks' && holes}<span class="count">{holes}</span>{/if}</button>
      {/each}
    {/each}
    <div class="foot">
      <button onclick={toggleTheme}>{theme === 'dark' ? '☀ Light theme' : '☾ Dark theme'}</button>
      <button class="small" title="How much the screen moves: Off, Calm, Smooth or Lively. Click to change." onclick={cycleLevel}>Animations: {LABEL[motion.level]}</button>
      Data file:<br />{db.dataPath}
      {#if db.folderProblem}<div><button class="small danger" title={db.folderProblem} onclick={() => (nav.dataFolder = true)}>⚠ Data folder problem</button></div>{/if}
      <div><button class="small" title="Keep your data in another folder" onclick={() => (nav.dataFolder = true)}>Change folder…</button></div>
      <div class="ver">Typewriter {db.version}</div></div>
  </nav>
  <main class="main" bind:this={mainEl}>
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
    {:else if nav.tab === 'draft'}<Draft />
    {:else if nav.tab === 'write'}<Write />
    {:else if nav.tab === 'board'}<Board />
    {:else if nav.tab === 'arcs'}<Arcs />
    {:else if nav.tab === 'characters'}<Characters />
    {:else if nav.tab === 'relationships'}<Relationships />
    {:else if nav.tab === 'world'}<World />
    {:else if nav.tab === 'glossary'}<Glossary />
    {:else if nav.tab === 'factions'}<Factions />
    {:else if nav.tab === 'items'}<Items />
    {:else if nav.tab === 'family'}<FamilyTree />
    {:else if nav.tab === 'maps'}<Maps />
    {:else if nav.tab === 'timeline'}<Timeline />
    {:else if nav.tab === 'checks'}<Checks />
    {:else if nav.tab === 'stats'}<Statistics />
    {:else if nav.tab === 'sources'}<Sources />
    {:else if nav.tab === 'ideas'}<Ideas />
    {:else if nav.tab === 'export'}<Export />
    {/if}
  </main>
</div>
