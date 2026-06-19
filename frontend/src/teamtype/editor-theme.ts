import type { Extension } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags } from '@lezer/highlight';
import { typst } from 'codemirror-lang-typst';

export const editorTheme = EditorView.theme(
  {
    '&': {
      height: '100%',
      color: 'var(--color-text)',
      backgroundColor: 'var(--color-bg-card)',
      fontSize: '13px',
    },
    '.cm-scroller': {
      fontFamily: 'var(--font-mono)',
      lineHeight: '1.6',
    },
    '.cm-content': {
      caretColor: 'var(--color-primary)',
      padding: 'var(--space-3) 0',
    },
    '.cm-gutters': {
      backgroundColor: 'var(--color-bg-card)',
      color: 'var(--color-code-line-num)',
      border: 'none',
    },
    '.cm-lineNumbers .cm-gutterElement': {
      padding: '0 var(--space-3) 0 var(--space-4)',
      minWidth: '2.75rem',
    },
    '.cm-activeLine': {
      backgroundColor: 'var(--color-accent-50)'
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'var(--color-accent-50)',
      color: 'var(--color-text-secondary)',
    },
    '&.cm-focused .cm-cursor': {
      borderLeftColor: 'var(--color-primary)'
    },
    '.cm-cursor': {
      borderLeftColor: 'var(--color-primary)'
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: 'var(--color-accent-100)',
    },
  },
  { dark: false },
);

/**
 * Maps the Lezer tags emitted by codemirror-lang-typst's Typst parser onto the
 * `--color-code-*` design tokens.
 */
export const typstHighlightStyle = HighlightStyle.define([
  { tag: tags.comment, color: 'var(--color-code-comment)', fontStyle: 'italic' },
  {
    tag: [
      tags.controlKeyword,
      tags.definitionKeyword,
      tags.moduleKeyword,
      tags.operatorKeyword,
      tags.processingInstruction,
    ],
    color: 'var(--color-code-keyword)',
    fontWeight: '500',
  },
  { tag: [tags.literal, tags.bool], color: 'var(--color-code-keyword)' },
  { tag: tags.string, color: 'var(--color-code-string)' },
  { tag: [tags.variableName, tags.labelName], color: 'var(--color-code-function)' },
  { tag: [tags.number, tags.integer, tags.float], color: 'var(--color-code-number)' },
  {
    tag: [
      tags.brace,
      tags.bracket,
      tags.squareBracket,
      tags.angleBracket,
      tags.paren,
      tags.punctuation,
      tags.separator,
      tags.derefOperator,
    ],
    color: 'var(--color-text-secondary)',
  },
  {
    tag: [
      tags.operator,
      tags.arithmeticOperator,
      tags.compareOperator,
      tags.updateOperator,
      tags.definitionOperator,
      tags.controlOperator,
      tags.typeOperator,
      tags.logicOperator,
    ],
    color: 'var(--color-text)',
  },
  { tag: tags.heading, fontWeight: '600' },
  { tag: tags.emphasis, fontStyle: 'italic' },
  { tag: tags.strong, fontWeight: '700' },
  { tag: tags.invalid, color: 'var(--color-error)' },
]);

/**
 * Presentation extensions shared by every document, regardless of file type.
 */
export const editorPresentation: Extension = [editorTheme, syntaxHighlighting(typstHighlightStyle)];

/**
 * Returns the grammar for a given file, based on its extension. Currently
 * only supports `.typ` files.
 */
export function languageForFile(file: string | null): Extension {
  if (file && file.toLowerCase().endsWith('.typ')) {
    return typst();
  }
  return [];
}
