"""Token splitter.

This module provides text splitting functionality with support for:
- Configurable chunk size and overlap
- Protected regex patterns (e.g., math formulas, images, links, tables)
- Header tracking for context preservation
- Smart merging with overlap handling
- Paragraph-aware splitting (preserves paragraph integrity)
"""

import itertools
import logging
import re
from typing import Callable, Generic, List, Pattern, Tuple, TypeVar

from pydantic import BaseModel, Field, PrivateAttr

from docreader.splitter.header_hook import (
    HeaderTracker,
)
from docreader.utils.split import split_by_char, split_by_sep
from docreader.utils.sentence_split import (
    split_chinese_sentences,
    split_english_sentences,
    split_paragraphs,
    split_at_nearest_space,
)

# Default configuration for text chunking
DEFAULT_CHUNK_OVERLAP = 100  # Number of tokens to overlap between chunks
DEFAULT_CHUNK_SIZE = 512  # Maximum size of each chunk in tokens

T = TypeVar("T")

logger = logging.getLogger(__name__)


class TextSplitter(BaseModel, Generic[T]):
    """Text splitter with support for protected patterns and header tracking.

    This class splits text into chunks while:
    - Respecting chunk size and overlap constraints
    - Preserving protected patterns (formulas, tables, code blocks)
    - Tracking headers for context preservation
    - Maintaining text integrity with smart merging
    - Optionally preserving paragraph boundaries (paragraph_aware mode)
    """

    chunk_size: int = Field(description="The token chunk size for each chunk.")
    chunk_overlap: int = Field(
        description="The token overlap of each chunk when splitting."
    )
    separators: List[str] = Field(
        description="Default separators for splitting into words"
    )

    # Paragraph-aware chunking mode
    paragraph_aware: bool = Field(
        default=True,
        description="Enable paragraph-aware splitting to preserve paragraph integrity"
    )
    language: str = Field(
        default="zh",
        description="Primary language for sentence splitting (zh=Chinese, en=English)"
    )
    sentence_end_punctuation: List[str] = Field(
        default_factory=lambda: ["。", "！", "？", "；", ".", "!", "?", ";"],
        description="Punctuation marks that indicate sentence endings"
    )

    # Try to keep the matched characters as a whole.
    # If it's too long, the content will be further segmented.
    # 尝试将匹配的字符作为整体保留，如果太长则进一步分段
    protected_regex: List[str] = Field(
        description="Protected regex for splitting into words"
    )
    len_function: Callable[[str], int] = Field(description="The length function.")
    # Header tracking Hook related attributes
    # 标题跟踪钩子相关属性
    header_hook: HeaderTracker = Field(default_factory=HeaderTracker, exclude=True)

    # Compiled regex patterns for protected content
    _protected_fns: List[Pattern] = PrivateAttr()
    # Split functions for different separators
    _split_fns: List[Callable] = PrivateAttr()

    def __init__(
        self,
        chunk_size: int = DEFAULT_CHUNK_SIZE,
        chunk_overlap: int = DEFAULT_CHUNK_OVERLAP,
        separators: List[str] = ["\n", "。", " "],
        paragraph_aware: bool = True,
        language: str = "zh",
        sentence_end_punctuation: List[str] = None,
        protected_regex: List[str] = [
            # math formula - LaTeX style formulas enclosed in $$
            r"\$\$[\s\S]*?\$\$",
            # image - Markdown image syntax ![alt](url)
            r"!\[.*?\]\(.*?\)",
            # link - Markdown link syntax [text](url)
            r"\[.*?\]\(.*?\)",
            # table header - Markdown table header with separator line
            r"(?:\|[^|\n]*)+\|[\r\n]+\s*(?:\|\s*:?-{3,}:?\s*)+\|[\r\n]+",
            # table body - Markdown table rows
            r"(?:\|[^|\n]*)+\|[\r\n]+",
            # code header - Code block start with language identifier
            r"```(?:\w+)[\r\n]+[^\r\n]*",
        ],
        length_function: Callable[[str], int] = lambda x: len(x),
    ):
        """Initialize with parameters.

        Args:
            chunk_size: Maximum size of each chunk
            chunk_overlap: Number of tokens to overlap between chunks
            separators: List of separators to use for splitting (in priority order)
            paragraph_aware: Enable paragraph-aware splitting mode
            language: Primary language (zh or en) for sentence splitting
            sentence_end_punctuation: List of sentence-ending punctuation marks
            protected_regex: Regex patterns for content that should be kept intact
            length_function: Function to calculate text length (default: character count)

        Raises:
            ValueError: If chunk_overlap is larger than chunk_size
        """
        if chunk_overlap > chunk_size:
            raise ValueError(
                f"Got a larger chunk overlap ({chunk_overlap}) than chunk size "
                f"({chunk_size}), should be smaller."
            )

        if sentence_end_punctuation is None:
            sentence_end_punctuation = ["。", "！", "？", "；", ".", "!", "?", ";"]

        super().__init__(
            chunk_size=chunk_size,
            chunk_overlap=chunk_overlap,
            separators=separators,
            paragraph_aware=paragraph_aware,
            language=language,
            sentence_end_punctuation=sentence_end_punctuation,
            protected_regex=protected_regex,
            len_function=length_function,
        )
        # Compile all protected regex patterns for efficient matching
        self._protected_fns = [re.compile(reg) for reg in protected_regex]
        # Create split functions: one for each separator, plus character-level splitting as fallback
        self._split_fns = [split_by_sep(sep) for sep in separators] + [split_by_char()]

    def split_text(self, text: str) -> List[Tuple[int, int, str]]:
        """Split text into chunks with overlap and protected pattern handling.

        Args:
            text: The input text to split

        Returns:
            List of tuples (start_pos, end_pos, chunk_text) representing each chunk
        """
        if text == "":
            return []

        # Use paragraph-aware mode if enabled
        if self.paragraph_aware:
            return self._split_text_paragraph_aware(text)

        # Original splitting logic (legacy mode)
        # Step 1: Split text by separators recursively
        splits = self._split(text)
        # Step 2: Extract protected content positions
        protect = self._split_protected(text)
        # Step 3: Merge splits with protected content to ensure integrity
        splits = self._join(splits, protect)

        # Verify that joining all splits reconstructs the original text
        assert "".join(splits) == text

        # Step 4: Merge splits into final chunks with overlap
        chunks = self._merge(splits)
        return chunks

    def _split_text_paragraph_aware(self, text: str) -> List[Tuple[int, int, str]]:
        """Split text using paragraph-aware mode.
        
        This mode:
        1. Splits text into paragraphs first
        2. Keeps short paragraphs intact as single chunks
        3. For long paragraphs, splits at sentence boundaries only
        4. Never splits within a sentence (only at sentence-ending punctuation)
        
        Args:
            text: The input text to split
            
        Returns:
            List of tuples (start_pos, end_pos, chunk_text) representing each chunk
        """
        logger.info(f"Using paragraph-aware splitting mode (language: {self.language})")
        
        # Step 1: Split into paragraphs
        paragraphs = split_paragraphs(text)
        logger.info(f"Split text into {len(paragraphs)} paragraphs")
        
        all_chunks = []
        
        # Step 2: Process each paragraph
        for para_start, para_end, para_text in paragraphs:
            para_len = self.len_function(para_text)
            
            # If paragraph fits in chunk size, keep it as-is
            if para_len <= self.chunk_size:
                all_chunks.append((para_start, para_end, para_text))
                logger.info(f"Paragraph fits in chunk (length: {para_len})")
            else:
                # Paragraph is too long, split into sentences
                logger.info(f"Paragraph too long ({para_len}), splitting into sentences")
                para_chunks = self._split_long_paragraph(para_text, para_start)
                all_chunks.extend(para_chunks)
        
        logger.info(f"Created {len(all_chunks)} chunks using paragraph-aware mode")
        return all_chunks

    def _split_long_paragraph(
        self, para_text: str, para_start: int
    ) -> List[Tuple[int, int, str]]:
        """Split a long paragraph into chunks at sentence boundaries.
        
        Strategy:
        1. Split into sentences at sentence-ending punctuation
        2. Merge sentences to fill chunks up to chunk_size
        3. Apply overlap between chunks
        4. For extremely long sentences, apply fallback splitting
        
        Args:
            para_text: The paragraph text to split
            para_start: Starting position of paragraph in original text
            
        Returns:
            List of tuples (start_pos, end_pos, chunk_text)
        """
        # Split into sentences based on language
        if self.language == "zh":
            sentences = split_chinese_sentences(
                para_text, 
                self.sentence_end_punctuation,
                max_len=self.chunk_size * 2
            )
        else:
            sentences = split_english_sentences(
                para_text,
                [m for m in self.sentence_end_punctuation if m in [".", "!", "?", ";"]],
                max_len=self.chunk_size * 2
            )
        
        logger.info(f"Split paragraph into {len(sentences)} sentences")
        
        # Handle extremely long sentences (fallback to hard split)
        processed_sentences = []
        for sent in sentences:
            sent_len = self.len_function(sent)
            if sent_len > self.chunk_size:
                logger.warning(
                    f"Sentence exceeds chunk_size ({sent_len} > {self.chunk_size}), "
                    f"applying hard split at whitespace"
                )
                # Split at nearest whitespace
                parts = []
                remaining = sent
                while self.len_function(remaining) > self.chunk_size:
                    split_pos = split_at_nearest_space(remaining, self.chunk_size, window=100)
                    parts.append(remaining[:split_pos].strip())
                    remaining = remaining[split_pos:].strip()
                if remaining:
                    parts.append(remaining)
                processed_sentences.extend(parts)
            else:
                processed_sentences.append(sent)
        
        # Merge sentences into chunks with overlap
        chunks = []
        current_chunk_sentences = []
        current_length = 0
        
        for sent in processed_sentences:
            sent_len = self.len_function(sent)
            
            # Check if adding this sentence would exceed chunk size
            if current_length + sent_len > self.chunk_size and current_chunk_sentences:
                # Finalize current chunk
                chunk_text = "".join(current_chunk_sentences)
                chunks.append(chunk_text)
                
                # Prepare next chunk with overlap
                if self.chunk_overlap > 0:
                    # Keep sentences from the end for overlap
                    overlap_sentences = []
                    overlap_len = 0
                    for s in reversed(current_chunk_sentences):
                        if overlap_len + self.len_function(s) > self.chunk_overlap:
                            break
                        overlap_sentences.insert(0, s)
                        overlap_len += self.len_function(s)
                    current_chunk_sentences = overlap_sentences
                    current_length = overlap_len
                else:
                    current_chunk_sentences = []
                    current_length = 0
            
            # Add sentence to current chunk
            current_chunk_sentences.append(sent)
            current_length += sent_len
        
        # Add final chunk
        if current_chunk_sentences:
            chunk_text = "".join(current_chunk_sentences)
            chunks.append(chunk_text)
        
        # Convert to position tuples
        result = []
        current_pos = para_start
        for chunk in chunks:
            # Find actual position in paragraph
            # This is approximate since we may have modified spacing
            start = current_pos
            end = start + len(chunk)
            result.append((start, end, chunk))
            # Move position forward, accounting for overlap
            current_pos = end - self.chunk_overlap if self.chunk_overlap > 0 else end
        
        return result

    def _split(self, text: str) -> List[str]:
        """Break text into splits that are smaller than chunk size.

        This method recursively splits text using separators in priority order.
        It tries each separator until it finds one that can split the text,
        then recursively processes any splits that are still too large.

        NOTE: the splits contain the separators.

        Args:
            text: The text to split

        Returns:
            List of text splits, each smaller than chunk_size
        """
        # If text is already small enough, return as-is
        if self.len_function(text) <= self.chunk_size:
            return [text]

        # Try each split function in order until one successfully splits the text
        splits = []
        for split_fn in self._split_fns:
            splits = split_fn(text)
            if len(splits) > 1:
                break

        # Process each split: keep if small enough, otherwise recursively split further
        new_splits = []
        for split in splits:
            split_len = self.len_function(split)
            if split_len <= self.chunk_size:
                new_splits.append(split)
            else:
                # Recursively split oversized chunks
                new_splits.extend(self._split(split))
        return new_splits

    def _merge(self, splits: List[str]) -> List[Tuple[int, int, str]]:
        """Merge splits into chunks with overlap and header tracking.

        The high-level idea is to keep adding splits to a chunk until we
        exceed the chunk size, then we start a new chunk with overlap.

        When we start a new chunk, we pop off the first element of the previous
        chunk until the total length is less than the chunk size.

        Headers are tracked and prepended to chunks for context preservation.

        Args:
            splits: List of text splits to merge

        Returns:
            List of tuples (start_pos, end_pos, chunk_text) representing merged chunks
        """
        # Final list of chunks with their positions
        chunks: List[Tuple[int, int, str]] = []

        # Current chunk being built: list of (start, end, text) tuples
        cur_chunk: List[Tuple[int, int, str]] = []

        # Track current headers and chunk length
        cur_headers, cur_len = "", 0
        # Track position in original text
        cur_start, cur_end = 0, 0

        for split in splits:
            # Calculate position of current split in original text
            cur_end = cur_start + len(split)
            split_len = self.len_function(split)

            # Warn if a single split exceeds chunk size (shouldn't happen after _split)
            if split_len > self.chunk_size:
                logger.error(
                    f"Got a split of size {split_len}, ",
                    f"larger than chunk size {self.chunk_size}.",
                )

            # Update header tracking with current split
            self.header_hook.update(split)
            cur_headers = self.header_hook.get_headers()
            cur_headers_len = self.len_function(cur_headers)

            # If headers are too large, skip them to avoid oversized chunks
            if cur_headers_len > self.chunk_size:
                logger.error(
                    f"Got headers of size {cur_headers_len}, ",
                    f"larger than chunk size {self.chunk_size}.",
                )
                cur_headers, cur_headers_len = "", 0

            # Check if adding this split would exceed chunk size
            # If so, finalize current chunk and start a new one with overlap
            if cur_len + split_len + cur_headers_len > self.chunk_size:
                # Finalize the previous chunk if it has content
                if len(cur_chunk) > 0:
                    chunks.append(
                        (
                            cur_chunk[0][0],  # Start position of first element
                            cur_chunk[-1][1],  # End position of last element
                            "".join([c[2] for c in cur_chunk]),  # Concatenated text
                        )
                    )

                # Start a new chunk with overlap from previous chunk
                # Keep popping off the first element of the previous chunk until:
                #   1. the current chunk length is less than chunk overlap
                #   2. the total length is less than chunk size
                while cur_chunk and (
                    cur_len > self.chunk_overlap
                    or cur_len + split_len + cur_headers_len > self.chunk_size
                ):
                    # Remove the first element to reduce overlap
                    first_chunk = cur_chunk.pop(0)
                    cur_len -= self.len_function(first_chunk[2])

                # Prepend headers to new chunk if:
                # 1. Headers exist
                # 2. Headers + split fit in chunk size
                # 3. Headers are not already in the split
                if (
                    cur_headers
                    and split_len + cur_headers_len < self.chunk_size
                    and cur_headers not in split
                ):
                    next_start = cur_chunk[0][0] if cur_chunk else cur_start
                    header_start = max(0, next_start - cur_headers_len)
                    header_end = cur_end

                    cur_chunk.insert(0, (header_start, header_end, cur_headers))
                    cur_len += cur_headers_len

            # Add current split to the chunk
            cur_chunk.append((cur_start, cur_end, split))
            cur_len += split_len
            cur_start = cur_end

        # Handle the last chunk (there should always be at least one)
        assert cur_chunk
        chunks.append(
            (
                cur_chunk[0][0],
                cur_chunk[-1][1],
                "".join([c[2] for c in cur_chunk]),
            )
        )

        return chunks

    def _split_protected(self, text: str) -> List[Tuple[int, str]]:
        """Extract protected content from text based on regex patterns.

        Args:
            text: The input text to scan for protected patterns

        Returns:
            List of tuples (start_position, protected_text) for each protected match
        """
        # Find all matches for all protected patterns
        matches = [
            (match.start(), match.end())
            for pattern in self._protected_fns
            for match in pattern.finditer(text)
        ]
        # Sort by start position (ascending), then by length (descending) to handle overlaps
        matches.sort(key=lambda x: (x[0], -x[1]))

        res = []

        def fold(initial: int, current: Tuple[int, int]) -> int:
            """Accumulator function to filter overlapping matches."""
            # Only process if match starts after previous match ended
            if current[0] >= initial:
                # Only keep protected content if it fits within chunk size
                if current[1] - current[0] < self.chunk_size:
                    res.append((current[0], text[current[0] : current[1]]))
                else:
                    logger.warning(f"Protected text ignore: {current}")
            # Return the end position of the furthest match so far
            return max(initial, current[1])

        # Filter overlapping matches using accumulate
        list(itertools.accumulate(matches, fold, initial=-1))
        return res

    def _join(self, splits: List[str], protect: List[Tuple[int, str]]) -> List[str]:
        """Merge splits with protected content to ensure protected patterns remain intact.

        Merges and splits elements in splits array based on protected substrings.

        The function processes the input splits to ensure all protected substrings
        remain as single items. If a protected substring is concatenated with preceding
        or following content in any split element, it will be separated from
        the adjacent content. The final result maintains the original order of content
        while enforcing the integrity of protected substrings.

        Key behaviors:
        1. Preserves the complete structure of each protected substring
        2. Separates protected substrings from any adjacent non-protected content
        3. Maintains the original sequence of all content
        4. Handles cases where protected substrings are partially concatenated

        Args:
            splits: List of text splits from _split()
            protect: List of (position, text) tuples for protected content

        Returns:
            List of text splits with protected content properly isolated
        """
        j = 0  # Index for protected content list
        point, start = 0, 0  # Track current position in original text
        res = []  # Result list of merged splits

        for split in splits:
            # Calculate end position of current split
            end = start + len(split)

            # Get the portion of split starting from current point
            cur = split[point - start :]

            # Process all protected content that overlaps with current split
            while j < len(protect):
                p_start, p_content = protect[j]
                p_end = p_start + len(p_content)

                # If protected content is beyond current split, move to next split
                if end <= p_start:
                    break

                # Add content before protected section
                if point < p_start:
                    local_end = p_start - point
                    res.append(cur[:local_end])
                    cur = cur[local_end:]
                    point = p_start

                # Add the protected content as a single unit
                res.append(p_content)
                j += 1

                # Skip content that's part of the protected section
                if point < p_end:
                    local_start = p_end - point
                    cur = cur[local_start:]
                    point = p_end

                # If no more content in current split, break
                if not cur:
                    break

            # Add any remaining content from current split
            if cur:
                res.append(cur)
                point = end

            # Move to next split
            start = end
        return res


if __name__ == "__main__":
    s = """
    这是一些普通文本。

    | 姓名 | 年龄 | 城市 |
    |------|------|------|
    | 张三 | 25   | 北京 |
    | 李四 | 30   | 上海 |
    | 王五 | 28   | 广州 |
    | 张三 | 25   | 北京 |
    | 李四 | 30   | 上海 |
    | 王五 | 28   | 广州 |

    这是文本结束。

"""

    sp = TextSplitter(chunk_size=200, chunk_overlap=2)
    ck = sp.split_text(s)
    for c in ck:
        print("------", len(c))
        print(c)
    pass
