<script lang="ts">
  import { onMount } from 'svelte';
  import { db } from './lib/db.svelte';
  import { nav } from './lib/nav.svelte';
  import Search from './lib/Search.svelte';
  import UndoToast from './lib/UndoToast.svelte';
  import { migrateChapters } from './lib/scenes';
  import { currentTheme, setTheme, type Theme } from './lib/theme';
  import Write from './views/Write.svelte';
  import Characters from './views/Characters.svelte';
  import Relationships from './views/Relationships.svelte';
  import World from './views/World.svelte';
  import Maps from './views/Maps.svelte';
  import Timeline from './views/Timeline.svelte';
  import Export from './views/Export.svelte';

  const tabs = [
    ['write', 'Write'],
    ['characters', 'Characters'],
    ['relationships', 'Relationships'],
    ['world', 'World'],
    ['maps', 'Maps & Paths'],
    ['timeline', 'Timeline'],
    ['export', 'Export & Backup'],
  ] as const;

  let error = $state('');
  let theme = $state<Theme>(currentTheme());
  function toggleTheme() { theme = theme === 'dark' ? 'light' : 'dark'; setTheme(theme); }

  onMount(() => { db.load().then(migrateChapters).catch((e) => (error = String(e))); });
</script>

<svelte:window onbeforeunload={() => db.flush()} onpagehide={() => db.flush()} onkeydown={(e) => { if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") { e.preventDefault(); nav.searchOpen = !nav.searchOpen; } }} />
<Search />
<UndoToast />

<div class="app">
  <nav class="nav">
    <div class="brand"><img src="/logo.svg" alt="" />Typewriter</div>
    <button class="search-btn" onclick={() => (nav.searchOpen = true)}>Search <kbd>Ctrl+K</kbd></button>
    {#each tabs as [id, label]}
      <button class:on={nav.tab === id} onclick={() => (nav.tab = id)}>{label}</button>
    {/each}
    <div class="foot">
      <button onclick={toggleTheme}>{theme === 'dark' ? '☀ Light theme' : '☾ Dark theme'}</button>
      Data file:<br />{db.dataPath}</div>
  </nav>
  <main class="main">
    {#if error}
      <div class="pane">Could not open the database: {error}</div>
    {:else if !db.ready}
      <div class="pane muted">Loading…</div>
    {:else if nav.tab === 'write'}<Write />
    {:else if nav.tab === 'characters'}<Characters />
    {:else if nav.tab === 'relationships'}<Relationships />
    {:else if nav.tab === 'world'}<World />
    {:else if nav.tab === 'maps'}<Maps />
    {:else if nav.tab === 'timeline'}<Timeline />
    {:else if nav.tab === 'export'}<Export />
    {/if}
  </main>
</div>
