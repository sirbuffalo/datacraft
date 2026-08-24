(() => {
  const encoder = new TextEncoder();
  const decoder = new TextDecoder();

  const crcTable = new Uint32Array(256);
  for (let index = 0; index < 256; index += 1) {
    let value = index;
    for (let bit = 0; bit < 8; bit += 1) value = (value >>> 1) ^ ((value & 1) ? 0xedb88320 : 0);
    crcTable[index] = value >>> 0;
  }

  function crc32(bytes) {
    let value = 0xffffffff;
    for (const byte of bytes) value = (value >>> 8) ^ crcTable[(value ^ byte) & 0xff];
    return (value ^ 0xffffffff) >>> 0;
  }

  function write16(view, offset, value) { view.setUint16(offset, value, true); }
  function write32(view, offset, value) { view.setUint32(offset, value >>> 0, true); }

  function create(files) {
    const entries = Object.entries(files).map(([name, value]) => ({
      name: encoder.encode(name.replace(/^\/+/, '')),
      data: typeof value === 'string' ? encoder.encode(value) : value,
    }));
    const localSize = entries.reduce((size, entry) => size + 30 + entry.name.length + entry.data.length, 0);
    const centralSize = entries.reduce((size, entry) => size + 46 + entry.name.length, 0);
    const bytes = new Uint8Array(localSize + centralSize + 22);
    const view = new DataView(bytes.buffer);
    let localOffset = 0;
    for (const entry of entries) {
      entry.offset = localOffset;
      entry.crc = crc32(entry.data);
      write32(view, localOffset, 0x04034b50);
      write16(view, localOffset + 4, 20);
      write16(view, localOffset + 6, 0x0800);
      write16(view, localOffset + 8, 0);
      write32(view, localOffset + 14, entry.crc);
      write32(view, localOffset + 18, entry.data.length);
      write32(view, localOffset + 22, entry.data.length);
      write16(view, localOffset + 26, entry.name.length);
      bytes.set(entry.name, localOffset + 30);
      bytes.set(entry.data, localOffset + 30 + entry.name.length);
      localOffset += 30 + entry.name.length + entry.data.length;
    }
    let centralOffset = localOffset;
    for (const entry of entries) {
      write32(view, centralOffset, 0x02014b50);
      write16(view, centralOffset + 4, 20);
      write16(view, centralOffset + 6, 20);
      write16(view, centralOffset + 8, 0x0800);
      write16(view, centralOffset + 10, 0);
      write32(view, centralOffset + 16, entry.crc);
      write32(view, centralOffset + 20, entry.data.length);
      write32(view, centralOffset + 24, entry.data.length);
      write16(view, centralOffset + 28, entry.name.length);
      write32(view, centralOffset + 42, entry.offset);
      bytes.set(entry.name, centralOffset + 46);
      centralOffset += 46 + entry.name.length;
    }
    write32(view, centralOffset, 0x06054b50);
    write16(view, centralOffset + 8, entries.length);
    write16(view, centralOffset + 10, entries.length);
    write32(view, centralOffset + 12, centralSize);
    write32(view, centralOffset + 16, localSize);
    return bytes;
  }

  async function inflate(bytes) {
    if (typeof DecompressionStream !== 'function') throw new Error('This browser cannot open compressed ZIP entries.');
    let stream;
    try {
      stream = new Blob([bytes]).stream().pipeThrough(new DecompressionStream('deflate-raw'));
    } catch (_) {
      throw new Error('This browser cannot open deflated ZIP entries.');
    }
    return new Uint8Array(await new Response(stream).arrayBuffer());
  }

  async function open(buffer) {
    const bytes = new Uint8Array(buffer);
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    let end = bytes.length - 22;
    const minimum = Math.max(0, bytes.length - 65557);
    while (end >= minimum && view.getUint32(end, true) !== 0x06054b50) end -= 1;
    if (end < minimum) throw new Error('Not a valid ZIP file.');
    const count = view.getUint16(end + 10, true);
    let offset = view.getUint32(end + 16, true);
    const files = new Map();
    for (let index = 0; index < count; index += 1) {
      if (view.getUint32(offset, true) !== 0x02014b50) throw new Error('Invalid ZIP directory.');
      const method = view.getUint16(offset + 10, true);
      const compressedSize = view.getUint32(offset + 20, true);
      const nameLength = view.getUint16(offset + 28, true);
      const extraLength = view.getUint16(offset + 30, true);
      const commentLength = view.getUint16(offset + 32, true);
      const localOffset = view.getUint32(offset + 42, true);
      const name = decoder.decode(bytes.slice(offset + 46, offset + 46 + nameLength));
      const localNameLength = view.getUint16(localOffset + 26, true);
      const localExtraLength = view.getUint16(localOffset + 28, true);
      const dataOffset = localOffset + 30 + localNameLength + localExtraLength;
      const compressed = bytes.slice(dataOffset, dataOffset + compressedSize);
      let data;
      if (method === 0) data = compressed;
      else if (method === 8) data = await inflate(compressed);
      else throw new Error(`Unsupported ZIP compression method ${method}.`);
      if (!name.endsWith('/')) files.set(name, data);
      offset += 46 + nameLength + extraLength + commentLength;
    }
    return files;
  }

  globalThis.DataCraftZIP = { create, open, decode: bytes => decoder.decode(bytes) };
})();
