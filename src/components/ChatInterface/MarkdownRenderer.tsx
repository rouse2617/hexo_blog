/**
 * MarkdownRenderer Component
 * Requirements: 1.3
 * 
 * Renders Markdown content with support for common syntax.
 * TODO: Integrate react-markdown and remark-gfm when dependencies are installed.
 */

import React from 'react';
import './MarkdownRenderer.css';

export interface MarkdownRendererProps {
  content: string;
}

/**
 * MarkdownRenderer Component
 * 
 * Requirement 1.3: Render Markdown content correctly
 * 
 * This is a basic implementation. Will be replaced with react-markdown
 * once dependencies are installed.
 */
const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ content }) => {
  /**
   * Basic Markdown parsing (temporary implementation)
   * This will be replaced with react-markdown + remark-gfm
   */
  const parseMarkdown = (text: string): React.ReactNode => {
    // Split by code blocks first
    const codeBlockRegex = /```(\w+)?\n([\s\S]*?)```/g;
    const parts: React.ReactNode[] = [];
    let lastIndex = 0;
    let match;

    while ((match = codeBlockRegex.exec(text)) !== null) {
      // Add text before code block
      if (match.index > lastIndex) {
        parts.push(parseInlineMarkdown(text.slice(lastIndex, match.index)));
      }

      // Add code block
      const language = match[1] || 'text';
      const code = match[2];
      parts.push(
        <pre key={match.index} className="code-block">
          <code className={`language-${language}`}>{code}</code>
        </pre>
      );

      lastIndex = match.index + match[0].length;
    }

    // Add remaining text
    if (lastIndex < text.length) {
      parts.push(parseInlineMarkdown(text.slice(lastIndex)));
    }

    return parts.length > 0 ? parts : parseInlineMarkdown(text);
  };

  /**
   * Parse inline Markdown (bold, italic, links, etc.)
   */
  const parseInlineMarkdown = (text: string): React.ReactNode => {
    const lines = text.split('\n');
    return lines.map((line, index) => {
      // Headers
      if (line.startsWith('# ')) {
        return <h1 key={index}>{line.slice(2)}</h1>;
      }
      if (line.startsWith('## ')) {
        return <h2 key={index}>{line.slice(3)}</h2>;
      }
      if (line.startsWith('### ')) {
        return <h3 key={index}>{line.slice(4)}</h3>;
      }

      // Lists
      if (line.startsWith('- ') || line.startsWith('* ')) {
        return <li key={index}>{parseInlineFormatting(line.slice(2))}</li>;
      }

      // Blockquote
      if (line.startsWith('> ')) {
        return <blockquote key={index}>{parseInlineFormatting(line.slice(2))}</blockquote>;
      }

      // Regular paragraph
      if (line.trim()) {
        return <p key={index}>{parseInlineFormatting(line)}</p>;
      }

      return <br key={index} />;
    });
  };

  /**
   * Parse inline formatting (bold, italic, code, links)
   */
  const parseInlineFormatting = (text: string): React.ReactNode => {
    const parts: React.ReactNode[] = [];
    let remaining = text;
    let key = 0;

    while (remaining.length > 0) {
      // Bold (**text**)
      const boldMatch = remaining.match(/\*\*(.*?)\*\*/);
      if (boldMatch && boldMatch.index !== undefined) {
        if (boldMatch.index > 0) {
          parts.push(remaining.slice(0, boldMatch.index));
        }
        parts.push(<strong key={key++}>{boldMatch[1]}</strong>);
        remaining = remaining.slice(boldMatch.index + boldMatch[0].length);
        continue;
      }

      // Italic (*text*)
      const italicMatch = remaining.match(/\*(.*?)\*/);
      if (italicMatch && italicMatch.index !== undefined) {
        if (italicMatch.index > 0) {
          parts.push(remaining.slice(0, italicMatch.index));
        }
        parts.push(<em key={key++}>{italicMatch[1]}</em>);
        remaining = remaining.slice(italicMatch.index + italicMatch[0].length);
        continue;
      }

      // Inline code (`code`)
      const codeMatch = remaining.match(/`(.*?)`/);
      if (codeMatch && codeMatch.index !== undefined) {
        if (codeMatch.index > 0) {
          parts.push(remaining.slice(0, codeMatch.index));
        }
        parts.push(<code key={key++} className="inline-code">{codeMatch[1]}</code>);
        remaining = remaining.slice(codeMatch.index + codeMatch[0].length);
        continue;
      }

      // Link ([text](url))
      const linkMatch = remaining.match(/\[(.*?)\]\((.*?)\)/);
      if (linkMatch && linkMatch.index !== undefined) {
        if (linkMatch.index > 0) {
          parts.push(remaining.slice(0, linkMatch.index));
        }
        parts.push(
          <a key={key++} href={linkMatch[2]} target="_blank" rel="noopener noreferrer">
            {linkMatch[1]}
          </a>
        );
        remaining = remaining.slice(linkMatch.index + linkMatch[0].length);
        continue;
      }

      // No more matches, add remaining text
      parts.push(remaining);
      break;
    }

    return parts.length > 0 ? parts : text;
  };

  return (
    <div className="markdown-renderer">
      {parseMarkdown(content)}
    </div>
  );
};

export default MarkdownRenderer;
