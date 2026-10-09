// Tiptap extensions shared by the composer and the signature editor.
import { TextStyle, Color } from '@tiptap/extension-text-style'
import { Table, TableCell, TableHeader } from '@tiptap/extension-table'

/**
 * Extended TextStyle to handle legacy <font> tags from signatures/pasted content
 */
export const ExtendedTextStyle = TextStyle.extend({
  parseHTML() {
    return [
      { tag: 'span' },
      { tag: 'font' },
    ]
  },
})

/**
 * Extended Color to handle legacy <font color="..."> tags
 */
export const ExtendedColor = Color.extend({
  addGlobalAttributes() {
    return [
      {
        types: this.options.types,
        attributes: {
          color: {
            default: null,
            parseHTML: (element: HTMLElement) => {
              const styleColor = element.style.color?.replace(/['"]+/g, '')
              if (styleColor) return styleColor
              if (element.tagName === 'FONT') {
                return element.getAttribute('color')
              }
              return null
            },
            renderHTML: (attributes: Record<string, string>) => {
              if (!attributes.color) {
                return {}
              }
              return {
                style: `color: ${attributes.color}`,
              }
            },
          },
        },
      },
    ]
  },
})

/**
 * Extended Table extensions to preserve inline style attributes
 */
export const ExtendedTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('style'),
        renderHTML: (attributes: Record<string, string>) => {
          if (!attributes.style) return {}
          return { style: attributes.style }
        },
      },
    }
  },
})

export const ExtendedTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('style'),
        renderHTML: (attributes: Record<string, string>) => {
          if (!attributes.style) return {}
          return { style: attributes.style }
        },
      },
    }
  },
})

export const ExtendedTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        default: null,
        parseHTML: (element: HTMLElement) => element.getAttribute('style'),
        renderHTML: (attributes: Record<string, string>) => {
          if (!attributes.style) return {}
          return { style: attributes.style }
        },
      },
    }
  },
})
