export interface OutlineHeading {
  level: number;
  text: string;
  line: number;
}

export interface OutlineNode extends OutlineHeading {
  children: OutlineNode[];
}

/**
 * Convert a flat list of headings into a tree structure based on their levels.
 */
export function nestOutline(headings: OutlineHeading[]): OutlineNode[] {
  const roots: OutlineNode[] = [];
  const stack: OutlineNode[] = [];
  for (const heading of headings) {
    const node: OutlineNode = { ...heading, children: [] };
    while (stack.length > 0 && stack[stack.length - 1]!.level >= node.level) {
      stack.pop();
    }
    const parent = stack[stack.length - 1];
    if (parent) {
      parent.children.push(node);
    } else {
      roots.push(node);
    }
    stack.push(node);
  }
  return roots;
}

const HEADING = /^(={1,6})[ \t]+(.+?)[ \t]*$/;
const FENCE = /^\s*```/;

export function parseOutline(source: string): OutlineHeading[] {
  const out: OutlineHeading[] = [];
  let inFence = false;
  const lines = source.split('\n');
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i] ?? '';
    if (FENCE.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) {
      continue;
    }
    const match = HEADING.exec(line);
    const marks = match?.[1];
    const text = match?.[2];
    if (marks && text) {
      out.push({ level: marks.length, text, line: i + 1 });
    }
  }
  return out;
}
