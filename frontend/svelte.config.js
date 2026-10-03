import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess(),
  onwarn(warning, handler) {
    // Labels sit next to their inputs here, so this warning is only noise.
    if (warning.code === 'a11y_label_has_associated_control') return
    handler(warning)
  },
}
