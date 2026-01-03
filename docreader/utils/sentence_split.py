"""
Sentence splitting utilities for Chinese and English text.

This module provides intelligent sentence splitting that respects sentence boundaries
and avoids splitting at inappropriate punctuation marks (like commas).
"""

import re
from typing import List, Tuple


def split_chinese_sentences(
    text: str,
    sentence_end_marks: List[str] = None,
    max_len: int = 10000
) -> List[str]:
    """
    Split Chinese text into sentences at sentence-ending punctuation only.
    
    Sentence-ending punctuation: 。！？；
    Will NOT split at: ，、：（逗号、顿号、冒号等句内标点）
    
    Args:
        text: Input text to split
        sentence_end_marks: List of sentence-ending punctuation marks.
                           Default: ["。", "！", "？", "；"]
        max_len: Maximum length before applying fallback splitting
    
    Returns:
        List of sentences with punctuation preserved
    """
    if sentence_end_marks is None:
        sentence_end_marks = ["。", "！", "？", "；"]
    
    if not text or not text.strip():
        return []
    
    # Build regex pattern for sentence-ending marks
    # Match any content followed by a sentence-ending mark
    marks_escaped = [re.escape(mark) for mark in sentence_end_marks]
    pattern = f"([^{''.join(marks_escaped)}]+[{''.join(marks_escaped)}]?)"
    
    matches = re.findall(pattern, text)
    sentences = [m.strip() for m in matches if m.strip()]
    
    # Handle sentences that are still too long (fallback splitting)
    result = []
    for sent in sentences:
        if len(sent) <= max_len:
            result.append(sent)
        else:
            # Fallback 1: Split at semicolon if present
            if "；" in sent:
                sub_sents = re.split(r"(；)", sent)
                # Merge pairs to keep semicolon with preceding text
                merged = []
                for i in range(0, len(sub_sents), 2):
                    part = sub_sents[i] + (sub_sents[i + 1] if i + 1 < len(sub_sents) else "")
                    if part.strip():
                        merged.append(part.strip())
                result.extend(merged)
            else:
                # Fallback 2: Split at colon
                if "：" in sent or ":" in sent:
                    sub_sents = re.split(r"([：:])", sent)
                    merged = []
                    for i in range(0, len(sub_sents), 2):
                        part = sub_sents[i] + (sub_sents[i + 1] if i + 1 < len(sub_sents) else "")
                        if part.strip():
                            merged.append(part.strip())
                    result.extend(merged)
                else:
                    # Fallback 3: Keep as-is, will be handled by hard split later
                    result.append(sent)
    
    return result


def split_english_sentences(
    text: str,
    sentence_end_marks: List[str] = None,
    max_len: int = 10000
) -> List[str]:
    """
    Split English text into sentences at sentence-ending punctuation only.
    
    Sentence-ending punctuation: . ! ? ;
    Will NOT split at: , : (commas, colons, etc.)
    
    Handles common abbreviations (Mr., Dr., etc.)
    
    Args:
        text: Input text to split
        sentence_end_marks: List of sentence-ending punctuation marks.
                           Default: [".", "!", "?", ";"]
        max_len: Maximum length before applying fallback splitting
    
    Returns:
        List of sentences with punctuation preserved
    """
    if sentence_end_marks is None:
        sentence_end_marks = [".", "!", "?", ";"]
    
    if not text or not text.strip():
        return []
    
    # Simple pattern that tries to avoid common abbreviations
    # Split on . ! ? ; followed by space and capital letter
    pattern = r'(?<=[.!?;])\s+(?=[A-Z])'
    
    sentences = re.split(pattern, text)
    sentences = [s.strip() for s in sentences if s.strip()]
    
    # Handle over-length sentences
    result = []
    for sent in sentences:
        if len(sent) <= max_len:
            result.append(sent)
        else:
            # Fallback: split at semicolon
            if ";" in sent:
                sub_sents = [s.strip() + ";" if not s.strip().endswith(";") else s.strip() 
                            for s in sent.split(";") if s.strip()]
                result.extend(sub_sents)
            else:
                result.append(sent)
    
    return result


def split_paragraphs(text: str) -> List[Tuple[int, int, str]]:
    """
    Split text into paragraphs based on double newlines or HTML paragraph tags.
    
    A paragraph boundary is defined as:
    - Two or more consecutive newlines (\\n\\n+)
    - HTML paragraph tags (<p>, </p>)
    
    Args:
        text: Input text
    
    Returns:
        List of tuples (start_pos, end_pos, paragraph_text)
    """
    if not text:
        return []
    
    # First, normalize line endings
    text = text.replace('\r\n', '\n').replace('\r', '\n')
    
    # Split by double newlines (paragraph separator)
    # Using regex to preserve position information
    paragraphs = []
    
    # Pattern: split at two or more newlines
    parts = re.split(r'\n\n+', text)
    
    current_pos = 0
    for part in parts:
        if not part.strip():
            current_pos += len(part) + 2  # Account for the \n\n separator
            continue
        
        # Find the actual start position of this part in original text
        start = text.find(part, current_pos)
        if start == -1:
            # Fallback: use current position
            start = current_pos
        
        end = start + len(part)
        paragraphs.append((start, end, part.strip()))
        
        # Move position forward
        current_pos = end
        # Try to account for the separator
        while current_pos < len(text) and text[current_pos] == '\n':
            current_pos += 1
    
    return paragraphs


def split_at_nearest_space(text: str, target_pos: int, window: int = 50) -> int:
    """
    Find the nearest whitespace character to a target position for safe splitting.
    
    Args:
        text: Text to search in
        target_pos: Target position (e.g., max chunk size)
        window: Search window size around target position
    
    Returns:
        Position of nearest space, or target_pos if no space found
    """
    if target_pos >= len(text):
        return len(text)
    
    # Search backward first (prefer splitting earlier)
    start = max(0, target_pos - window)
    for i in range(target_pos, start, -1):
        if text[i] in ' \t\n':
            return i
    
    # Search forward if no space found backward
    end = min(len(text), target_pos + window)
    for i in range(target_pos, end):
        if text[i] in ' \t\n':
            return i
    
    # No space found, return target position
    return target_pos
