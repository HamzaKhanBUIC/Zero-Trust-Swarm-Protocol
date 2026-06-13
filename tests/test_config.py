import os
import pytest
from pydantic import ValidationError
from src.config import Settings, get_settings

def test_settings_loads_firecrawl_key(monkeypatch):
    """Test that the settings successfully load the Firecrawl API key from environment variables."""
    # Arrange
    test_api_key = "fc-test-key-12345"
    monkeypatch.setenv("FIRECRAWL_API_KEY", test_api_key)
    
    # Act
    settings = Settings()
    
    # Assert
    assert settings.firecrawl_api_key == test_api_key

def test_settings_missing_firecrawl_key_raises_error(monkeypatch):
    """Test that omitting the required FIRECRAWL_API_KEY raises a ValidationError."""
    # Arrange
    monkeypatch.delenv("FIRECRAWL_API_KEY", raising=False)
    
    # Act & Assert
    with pytest.raises(ValidationError):
        Settings()

def test_settings_ollama_base_url_default(monkeypatch):
    """Test that OLLAMA_BASE_URL defaults to localhost if not provided."""
    # Arrange
    monkeypatch.delenv("FIRECRAWL_API_KEY", raising=False)
    monkeypatch.setenv("FIRECRAWL_API_KEY", "dummy-key")
    monkeypatch.delenv("OLLAMA_BASE_URL", raising=False)
    
    # Act
    settings = Settings()
    
    # Assert
    assert settings.ollama_base_url == "http://localhost:11434"

def test_get_settings_is_singleton(monkeypatch):
    """Test that get_settings caches the Settings instance."""
    monkeypatch.setenv("FIRECRAWL_API_KEY", "dummy-key")
    settings_1 = get_settings()
    settings_2 = get_settings()
    assert settings_1 is settings_2
