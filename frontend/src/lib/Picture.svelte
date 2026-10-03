<script lang="ts">
  import { shrinkPicture } from './picture';
  import ConfirmButton from './ConfirmButton.svelte';

  let { value = '', square = false, label = 'Picture', onchange }: {
    value?: string; square?: boolean; label?: string; onchange: (v: string) => void;
  } = $props();
  let error = $state('');

  async function pick(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = '';
    if (!file) return;
    error = '';
    try { onchange(await shrinkPicture(file, { square })); }
    catch { error = 'I could not read that file. Pick a PNG, JPG or WEBP picture.'; }
  }
</script>

<div class="pic">
  {#if value}
    <img src={value} alt={label} class:square />
  {:else}
    <div class="blank" class:square>No picture</div>
  {/if}
  <div class="btns">
    <label class="small btn">{value ? 'Change' : 'Add'} {label.toLowerCase()}
      <input type="file" accept="image/*" onchange={pick} style="display:none" /></label>
    {#if value}<ConfirmButton onconfirm={() => onchange('')} />{/if}
    {#if error}<span class="muted">{error}</span>{/if}
  </div>
</div>

<style>
  .pic { display: flex; gap: .8rem; align-items: flex-end; margin: .4rem 0 .8rem; }
  img { max-width: 220px; max-height: 220px; border-radius: 8px; border: 1px solid var(--line); display: block; }
  img.square { width: 140px; height: 140px; object-fit: cover; }
  .blank { width: 140px; height: 140px; border: 1px dashed var(--line); border-radius: 8px; display: grid; place-items: center; color: var(--muted); font-size: .8rem; }
  .btns { display: flex; flex-direction: column; gap: .3rem; align-items: flex-start; }
  .btn { cursor: pointer; display: inline-block; }
</style>
