from langchain_text_splitters import RecursiveCharacterTextSplitter

def chunk_markdown(text: str) -> list[str]:
    """
    Splits markdown text into token-aware chunks.
    Configured for 512 max tokens with a 100-token overlap to prevent fracturing code blocks.
    Uses tiktoken's cl100k_base encoder under the hood.
    """
    text_splitter = RecursiveCharacterTextSplitter.from_tiktoken_encoder(
        encoding_name="cl100k_base",
        chunk_size=512,
        chunk_overlap=100,
    )
    
    return text_splitter.split_text(text)
