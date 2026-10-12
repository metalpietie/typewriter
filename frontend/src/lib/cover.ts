import { Store } from '../../bindings/typewriter';
import { COVER_H, COVER_W, coverFit, fitText, scaledSize, textOn, type Spec } from './cover-core';

// The cover of a book. It is kept as a JPEG, one for each book, and read only when needed, because a picture is large.
// The rules for the layout are in cover-core.ts. This file draws and saves.

export type CoverDoc = {
  id?: number;
  book: number;
  image: string;            // the finished cover, a JPEG as a data URL
  kind: 'picture' | 'made'; // a picture you chose, or one made from a template
  spec?: Spec;              // for a made cover: the choices, so you can change them later
  photo?: string;           // for a made cover: the background picture, if any
};

export async function getCover(bookId: number): Promise<CoverDoc | null> {
  const docs = (await Store.ListBy('cover', 'book', bookId)) ?? [];
  if (!docs.length) return null;
  return { ...JSON.parse(docs[0].data), id: docs[0].id } as CoverDoc;
}

export async function saveCover(doc: CoverDoc): Promise<CoverDoc> {
  const { id, ...data } = doc;
  const saved = await Store.Save({ id: id ?? 0, kind: 'cover', data: JSON.stringify(data), updatedAt: '' });
  return { ...doc, id: saved.id };
}

export async function removeCover(id: number) {
  await Store.Delete('cover', id);
}

/** Load a data URL as a picture that can be drawn. */
export function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error('That picture could not be read.'));
    img.src = url;
  });
}

/** A picture file as a cover: a JPEG with the long side at most 2400, on a white page, nothing cut off. */
export async function pictureToCover(file: File): Promise<string> {
  const bmp = await createImageBitmap(file);
  const { w, h } = scaledSize(bmp.width, bmp.height);
  const c = document.createElement('canvas');
  c.width = w; c.height = h;
  const g = c.getContext('2d')!;
  g.fillStyle = '#ffffff';
  g.fillRect(0, 0, w, h);
  g.drawImage(bmp, 0, 0, w, h);
  bmp.close();
  return c.toDataURL('image/jpeg', 0.9);
}

/** A picture file as the background of a made cover: a JPEG, the long side at most 1600. */
export async function pictureToPhoto(file: File): Promise<string> {
  const bmp = await createImageBitmap(file);
  const { w, h } = scaledSize(bmp.width, bmp.height, 1600);
  const c = document.createElement('canvas');
  c.width = w; c.height = h;
  c.getContext('2d')!.drawImage(bmp, 0, 0, w, h);
  bmp.close();
  return c.toDataURL('image/jpeg', 0.85);
}

const SERIF = 'Georgia, "Times New Roman", serif';
const SANS = '"Segoe UI", Helvetica, Arial, sans-serif';

/** Draw a cover on a canvas of COVER_W by COVER_H. photo is the background picture, if there is one. */
export function drawCover(canvas: HTMLCanvasElement, spec: Spec, photo: HTMLImageElement | null) {
  canvas.width = COVER_W; canvas.height = COVER_H;
  const g = canvas.getContext('2d')!;
  const W = COVER_W, H = COVER_H;
  g.fillStyle = spec.bg;
  g.fillRect(0, 0, W, H);
  if (photo) {
    const f = coverFit(photo.naturalWidth, photo.naturalHeight, W, H);
    g.drawImage(photo, f.x, f.y, f.w, f.h);
    g.fillStyle = `rgba(0,0,0,${spec.dim})`;
    g.fillRect(0, 0, W, H);
  }
  const measure = (family: string, weight: string, style = '') => (s: string, size: number) => {
    g.font = `${style} ${weight} ${size}px ${family}`.trim();
    return g.measureText(s).width;
  };
  const lines = (list: string[], x: number, y: number, size: number, lh: number, align: CanvasTextAlign) => {
    g.textAlign = align;
    list.forEach((l, i) => g.fillText(l, x, y + i * size * lh));
  };
  g.textBaseline = 'alphabetic';

  if (spec.template === 'classic') {
    g.strokeStyle = spec.accent;
    g.lineWidth = 8; g.strokeRect(70, 70, W - 140, H - 140);
    g.lineWidth = 3; g.strokeRect(105, 105, W - 210, H - 210);
    const title = fitText(spec.title, W - 400, 760, measure(SERIF, 'bold'), { max: 230 });
    const top = 560;
    g.fillStyle = spec.fg; g.font = `bold ${title.size}px ${SERIF}`;
    lines(title.lines, W / 2, top + title.size * 0.9, title.size, 1.12, 'center');
    let y = top + title.size * 1.12 * title.lines.length + 40;
    g.fillStyle = spec.accent; g.fillRect(W / 2 - 160, y, 320, 6); y += 100;
    if (spec.subtitle) {
      const sub = fitText(spec.subtitle, W - 460, 280, measure(SERIF, 'normal', 'italic'), { max: 80, min: 40 });
      g.fillStyle = spec.fg; g.font = `italic ${sub.size}px ${SERIF}`;
      lines(sub.lines, W / 2, y + sub.size * 0.8, sub.size, 1.2, 'center');
    }
    if (spec.author) {
      const a = fitText(spec.author.toUpperCase(), W - 460, 200, measure(SERIF, 'normal'), { max: 92, min: 44 });
      g.fillStyle = spec.fg; g.font = `${a.size}px ${SERIF}`;
      lines(a.lines, W / 2, H - 300, a.size, 1.2, 'center');
    }
  } else if (spec.template === 'modern') {
    g.fillStyle = spec.accent; g.fillRect(0, 0, 56, H);
    const title = fitText(spec.title, W - 420, 1000, measure(SANS, 'bold'), { max: 250 });
    g.fillStyle = spec.fg; g.font = `bold ${title.size}px ${SANS}`;
    lines(title.lines, 170, 420 + title.size * 0.85, title.size, 1.08, 'left');
    let y = 420 + title.size * 1.08 * title.lines.length + 50;
    g.fillStyle = spec.accent; g.fillRect(170, y, 360, 14); y += 110;
    if (spec.subtitle) {
      const sub = fitText(spec.subtitle, W - 420, 300, measure(SANS, 'normal'), { max: 76, min: 40 });
      g.fillStyle = spec.fg; g.font = `${sub.size}px ${SANS}`;
      lines(sub.lines, 170, y + sub.size * 0.8, sub.size, 1.25, 'left');
    }
    if (spec.author) {
      const a = fitText(spec.author.toUpperCase(), W - 420, 220, measure(SANS, 'bold'), { max: 84, min: 44 });
      g.fillStyle = spec.fg; g.font = `bold ${a.size}px ${SANS}`;
      lines(a.lines, 170, H - 260, a.size, 1.2, 'left');
    }
  } else {
    const bandTop = 820, bandH = 700;
    g.fillStyle = spec.accent; g.fillRect(0, bandTop, W, bandH);
    const onBand = textOn(spec.accent);
    const title = fitText(spec.title, W - 300, bandH - 120, measure(SERIF, 'bold'), { max: 210 });
    g.fillStyle = onBand; g.font = `bold ${title.size}px ${SERIF}`;
    const block = title.size * 1.1 * title.lines.length;
    lines(title.lines, W / 2, bandTop + (bandH - block) / 2 + title.size * 0.85, title.size, 1.1, 'center');
    if (spec.subtitle) {
      const sub = fitText(spec.subtitle, W - 360, 220, measure(SERIF, 'normal', 'italic'), { max: 72, min: 40 });
      g.fillStyle = spec.fg; g.font = `italic ${sub.size}px ${SERIF}`;
      lines(sub.lines, W / 2, bandTop + bandH + 130, sub.size, 1.2, 'center');
    }
    if (spec.author) {
      const a = fitText(spec.author.toUpperCase(), W - 360, 200, measure(SERIF, 'normal'), { max: 92, min: 44 });
      g.fillStyle = spec.fg; g.font = `${a.size}px ${SERIF}`;
      lines(a.lines, W / 2, H - 240, a.size, 1.2, 'center');
    }
  }
}
