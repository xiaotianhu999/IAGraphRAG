from dataclasses import dataclass, field


@dataclass
class ChunkingConfig:
    """
    Configuration for text chunking process.
    Controls how documents are split into smaller pieces for processing.
    """

    # Maximum size of each chunk in tokens/chars
    chunk_size: int = 512

    # Number of tokens/chars to overlap between chunks
    chunk_overlap: int = 50

    # Text separators in order of priority
    separators: list[str] = field(default_factory=lambda: ["\n\n", "\n", "。"])

    # Whether to enable multimodal processing (text + images)
    enable_multimodal: bool = False

    # Whether to enable paragraph-aware chunking (preserve paragraph integrity)
    paragraph_aware: bool = True

    # Primary language for sentence splitting (zh, en)
    language: str = "zh"

    # Sentence-ending punctuation marks for splitting
    # For Chinese: 。！？；
    # For English: . ! ? ;
    sentence_end_punctuation: list[str] = field(
        default_factory=lambda: ["。", "！", "？", "；", ".", "!", "?", ";"]
    )

    # Preferred field name going forward
    storage_config: dict[str, str] = field(default_factory=dict)

    # VLM configuration for image captioning
    vlm_config: dict[str, str] = field(default_factory=dict)
