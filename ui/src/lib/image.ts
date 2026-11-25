// Phone photos are huge — scale them down before upload (keeps GIFs untouched).
const MAX = 1600

export async function shrink(file: File): Promise<File> {
  if (!file.type.startsWith('image/') || file.type === 'image/gif') return file
  try {
    const bmp = await createImageBitmap(file, { imageOrientation: 'from-image' })
    const scale = Math.min(1, MAX / Math.max(bmp.width, bmp.height))
    if (scale === 1 && file.size < 1.5e6) { bmp.close(); return file }
    const c = document.createElement('canvas')
    c.width = Math.round(bmp.width * scale); c.height = Math.round(bmp.height * scale)
    c.getContext('2d')!.drawImage(bmp, 0, 0, c.width, c.height)
    bmp.close()
    const blob = await new Promise<Blob | null>(r => c.toBlob(r, 'image/jpeg', 0.86))
    if (!blob) return file
    return new File([blob], file.name.replace(/\.\w+$/, '') + '.jpg', { type: 'image/jpeg' })
  } catch { return file }
}
