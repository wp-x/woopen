import DOMPurify from 'dompurify'
import { marked } from 'marked'

const UTF8_BOM = [0xef, 0xbb, 0xbf] as const
const UTF16_LE_BOM = [0xff, 0xfe] as const
const UTF16_BE_BOM = [0xfe, 0xff] as const

export function decodeTextContent(contentBase64: string): string {
  const bytes = decodeBase64(contentBase64)
  if (startsWith(bytes, UTF8_BOM)) {
    return new TextDecoder('utf-8').decode(bytes.slice(UTF8_BOM.length))
  }
  if (startsWith(bytes, UTF16_LE_BOM)) {
    return new TextDecoder('utf-16le').decode(bytes.slice(UTF16_LE_BOM.length))
  }
  if (startsWith(bytes, UTF16_BE_BOM)) {
    return new TextDecoder('utf-16be').decode(bytes.slice(UTF16_BE_BOM.length))
  }
  return decodeWithoutBom(bytes)
}

export function renderMarkdown(content: string): string {
  const html = marked.parse(content, { gfm: true, breaks: true, async: false })
  return DOMPurify.sanitize(String(html))
}

function decodeBase64(value: string): Uint8Array {
  const binary = atob(value)
  return Uint8Array.from(binary, character => character.charCodeAt(0))
}

function decodeWithoutBom(bytes: Uint8Array): string {
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch {
    return new TextDecoder('gb18030', { fatal: true }).decode(bytes)
  }
}

function startsWith(bytes: Uint8Array, prefix: readonly number[]): boolean {
  return prefix.every((value, index) => bytes[index] === value)
}
