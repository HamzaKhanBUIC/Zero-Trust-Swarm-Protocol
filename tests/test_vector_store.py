from unittest.mock import MagicMock, patch
from src.modules.ingestion.vector_store import VectorStore

@patch("src.modules.ingestion.vector_store.lancedb.connect")
def test_vector_store_initialization(mock_connect, monkeypatch):
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://localhost:11434")
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    
    store = VectorStore()
    
    # Verify we attempted to connect to the local db
    mock_connect.assert_called_once_with("./omega_docs.db")
    assert store.db is not None

@patch("src.modules.ingestion.vector_store.lancedb.connect")
def test_vector_store_upsert_documents(mock_connect, monkeypatch):
    monkeypatch.setenv("OLLAMA_BASE_URL", "http://localhost:11434")
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    
    mock_db = MagicMock()
    mock_table = MagicMock()
    mock_db.create_table.return_value = mock_table
    mock_connect.return_value = mock_db
    
    store = VectorStore()
    
    chunks = ["This is chunk 1", "This is chunk 2"]
    url = "https://example.com/docs"
    
    store.upsert_documents(url, chunks)
    
    # Should create table if not exists, and add data
    mock_db.create_table.assert_called_once()
    
    # Check the data format passed to create_table
    kwargs = mock_db.create_table.call_args[1]
    added_data = kwargs["data"]
    assert len(added_data) == 2
    assert added_data[0]["text"] == "This is chunk 1"
    assert added_data[0]["url"] == url
