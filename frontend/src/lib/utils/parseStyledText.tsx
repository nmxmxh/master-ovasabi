import type { JSX } from "react";

const TAG_MAP: Record<string, string> = {
  light: "light-light",
  dark: "dark-light",
  "no-split": "no-split",
};

export function parseStyledText(text: string): (JSX.Element | string)[] {
  const elements: (JSX.Element | string)[] = [];
  const regex = /<br\s*\/?>|\[([a-zA-Z-]+)\](.*?)\[\/\1\]/g;

  let lastIndex = 0;
  let match: RegExpExecArray | null;
  let key = 0;

  while ((match = regex.exec(text)) !== null) {
    const matchStart = match.index;

    // Add plain text before the match
    if (lastIndex < matchStart) {
      elements.push(text.slice(lastIndex, matchStart));
    }

    // Handle <br />
    if (match[0].startsWith("<br")) {
      elements.push(<br key={key++} />);
    }
    // Handle [tag]text[/tag]
    else if (match[1] && match[2]) {
      const className = TAG_MAP[match[1]];
      if (className) {
        elements.push(
          <span className={className} key={key++}>
            {match[2]}
          </span>
        );
      } else {
        elements.push(match[2]); // fallback to raw if tag is unrecognized
      }
    }

    lastIndex = regex.lastIndex;
  }

  // Remaining plain text
  if (lastIndex < text.length) {
    elements.push(text.slice(lastIndex));
  }

  return elements;
}
