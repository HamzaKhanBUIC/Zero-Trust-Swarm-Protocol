from src.modules.ingestion.chunker import chunk_markdown
import tiktoken

def test_chunk_markdown_respects_token_limit():
    # Arrange
    # Generate a long string of repeating words. 
    # "hello " is typically 1 token. We'll generate 1500 tokens.
    raw_text = "hello " * 1500
    
    # Act
    chunks = chunk_markdown(raw_text)
    
    # Assert
    assert len(chunks) > 1, "Text should be split into multiple chunks"
    
    encoder = tiktoken.get_encoding("cl100k_base")
    for chunk in chunks:
        token_count = len(encoder.encode(chunk))
        # The chunker is configured for 512 max size.
        assert token_count <= 512, f"Chunk exceeded 512 tokens: {token_count}"
