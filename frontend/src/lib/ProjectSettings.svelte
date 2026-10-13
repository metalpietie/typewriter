<script lang="ts">
  import { veil, dialog } from './motion.svelte';
  import { db } from './db.svelte';
  import { nav } from './nav.svelte';
  import { has, projectType, setTool, setType } from './projecttype.svelte';
  import { TOOLS, TYPES, type ProjectType, type Tool } from './projecttype-core';

  // The type of the open project, and which tools show. Nothing is deleted when a tool is hidden.
  let message = $state('');
  const type = $derived(projectType());

  const close = () => { db.flush(); nav.projectSettings = false; };
  async function pickType(t: ProjectType) {
    message = '';
    try { await setType(t); } catch (e: any) { message = e?.message ?? String(e); }
  }
  async function tick(tool: Tool, on: boolean) {
    message = '';
    try { await setTool(tool, on); } catch (e: any) { message = e?.message ?? String(e); }
  }
</script>

<svelte:window onkeydown={(e) => { if (e.key === 'Escape') close(); }} />

<div class="overlay" transition:veil|global role="presentation" onmousedown={(e) => { if (e.target === e.currentTarget) close(); }}>
  <div transition:dialog|global class="palette box" role="dialog" aria-label="Project settings">
    <div class="head"><h2 style="margin:0">Project settings</h2><button class="small" onclick={close}>Close</button></div>
    <p class="muted" style="margin-top:.4rem">These choices are for the project “{db.project}”. They go with the project in a backup.</p>

    <h3>Type of project</h3>
    {#each TYPES as [t, name, about]}
      <label class="type" class:on={type === t}>
        <input type="radio" name="ptype" style="width:auto" checked={type === t} onchange={() => pickType(t)} />
        <span><b>{name}</b><br /><small class="muted">{about}</small></span>
      </label>
    {/each}
    <p class="muted" style="font-size:.85rem">When you change the type, the ticks below go back to what that type usually shows.</p>

    <h3>Tools</h3>
    <p class="muted" style="margin-top:0">
      Nothing is deleted when you hide a tool. Your characters, places and maps stay in the project. Tick the tool again and they are back.
    </p>
    <div class="tools">
      {#each TOOLS as [tool, name]}
        <label class="tool"><input type="checkbox" style="width:auto" checked={has(tool)} onchange={(e) => tick(tool, e.currentTarget.checked)} />{name}</label>
      {/each}
    </div>
    {#if message}<p style="color:var(--danger)">{message}</p>{/if}
  </div>
</div>

<style>
  .box { width: min(560px, 94vw); max-height: 85vh; overflow: auto; }
  .type { display: flex; gap: .6rem; align-items: flex-start; padding: .5rem .6rem; margin: .3rem 0; border: 1px solid var(--line); border-radius: 6px; color: var(--text); cursor: pointer; }
  .type.on { border-color: var(--accent); }
  .tools { display: grid; grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr)); gap: .1rem .8rem; }
  .tool { display: flex; gap: .5rem; align-items: center; color: var(--text); margin: .15rem 0; }
</style>
