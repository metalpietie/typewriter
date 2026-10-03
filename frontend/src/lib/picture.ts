/** Shrink a picture file so it stays small in the data file. Returns a webp data URL. */
export async function shrinkPicture(file: File, opts: { size?: number; square?: boolean } = {}): Promise<string> {
  const size = opts.size ?? 480;
  const bmp = await createImageBitmap(file);
  let sx = 0, sy = 0, sw = bmp.width, sh = bmp.height;
  let w: number, h: number;
  if (opts.square) {
    // cut the middle square out, so a face stays in the middle
    const s = Math.min(bmp.width, bmp.height);
    sx = (bmp.width - s) / 2; sy = (bmp.height - s) / 2; sw = sh = s;
    w = h = Math.min(size, s);
  } else {
    const k = Math.min(1, size / Math.max(bmp.width, bmp.height));
    w = Math.round(bmp.width * k); h = Math.round(bmp.height * k);
  }
  const c = document.createElement('canvas');
  c.width = w; c.height = h;
  c.getContext('2d')!.drawImage(bmp, sx, sy, sw, sh, 0, 0, w, h);
  bmp.close();
  return c.toDataURL('image/webp', 0.85);
}
