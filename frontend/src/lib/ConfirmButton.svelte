<script lang="ts">
  let { label = 'Delete', onconfirm }: { label?: string; onconfirm: () => void } = $props();
  let armed = $state(false);
  let timer: ReturnType<typeof setTimeout>;
  function click() {
    if (armed) { clearTimeout(timer); armed = false; onconfirm(); return; }
    armed = true;
    timer = setTimeout(() => (armed = false), 3000);
  }
</script>

<button class="danger small" class:armed onclick={click}>{armed ? 'Sure?' : label}</button>
