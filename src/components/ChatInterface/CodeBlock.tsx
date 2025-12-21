/**
 * CodeBlock Component
 * Requirements: 1.4
 * 
 * Displays code with syntax highlighting.
 * TODO: Integrate react-syntax-highlighter when dependencies are installed.
 */

import React from 'react';
import './CodeBlock.css';

export interface CodeBlockProps {
  code: string;
  language?: string;
}

/**
 * CodeBlock Component
 * 
 * Requirement 1.4: Syntax highlight code blocks
 * 
 * This is a basic implementation. Will be replaced with react-syntax-highlighter
 * once dependencies are installed.
 */
const CodeBlock: React.FC<CodeBlockProps> = ({ code, language = 'text' }) => {
  /**
   * Copy code to clipboard
   */
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      // TODO: Show success toast
    } catch (err) {
      console.error('Failed to copy code:', err);
    }
  };

  return (
    <div className="code-block-container">
      <div className="code-block-header">
        <span className="code-language">{language}</span>
        <button
          className="copy-button"
          onClick={handleCopy}
          aria-label="Copy code"
          title="Copy code"
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
          >
            <rect x="4" y="4" width="8" height="8" strokeWidth="1.5" rx="1" />
            <path d="M2 6V2h4M10 2h4v4M14 10v4h-4M6 14H2v-4" strokeWidth="1.5" />
          </svg>
        </button>
      </div>
      <pre className="code-block-pre">
        <code className={`language-${language}`}>
          {code}
        </code>
      </pre>
    </div>
  );
};

export default CodeBlock;
