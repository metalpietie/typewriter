<script lang="ts">
  import { onMount } from 'svelte';
  import { db } from './lib/db.svelte';
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
    ['export', 'Export'],
  ] as const;

  let tab = $state<(typeof tabs)[number][0]>('write');
  let error = $state('');
  let theme = $state<Theme>(currentTheme());
  function toggleTheme() { theme = theme === 'dark' ? 'light' : 'dark'; setTheme(theme); }

  onMount(() => { db.load().catch((e) => (error = String(e))); });
</script>

<div class="app">
  <nav class="nav">
    <div class="brand"><img src="/logo.svg" alt="" />Typewriter</div>
    {#each tabs as [id, label]}
      <button class:on={tab === id} onclick={() => (tab = id)}>{label}</button>
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
    {:else if tab === 'write'}<Write />
    {:else if tab === 'characters'}<Characters />
    {:else if tab === 'relationships'}<Relationships />
    {:else if tab === 'world'}<World />
    {:else if tab === 'maps'}<Maps />
    {:else if tab === 'timeline'}<Timeline />
    {:else if tab === 'export'}<Export />
    {/if}
  </main>
</div>
