from unittest.mock import MagicMock
import pytest
from src.modules.extractor.scraper import DocumentationScraper

def test_scraper_initialization(monkeypatch):
    """Test that the scraper initializes with the API key from settings."""
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    
    scraper = DocumentationScraper()
    assert scraper.app is not None
    # We assume the wrapper sets up the underlying FirecrawlApp properly.

def test_scraper_scrape_docs(monkeypatch):
    """Test that scrape_docs calls FirecrawlApp with the correct parameters and returns markdown."""
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    
    # Mock the FirecrawlApp instance
    mock_app = MagicMock()
    mock_app.crawl_url.return_value = {
        "success": True,
        "data": [
            {
                "url": "https://example.com/docs",
                "markdown": "# Example Docs\n\nThis is a test."
            }
        ]
    }
    
    scraper = DocumentationScraper()
    scraper.app = mock_app
    
    result = scraper.scrape_docs("https://example.com/docs")
    
    mock_app.crawl_url.assert_called_once_with(
        "https://example.com/docs", 
        params={
            'limit': 100, 
            'scrapeOptions': {'formats': ['markdown']}
        }
    )
    
    assert len(result) == 1
    assert result[0]["url"] == "https://example.com/docs"
    assert "Example Docs" in result[0]["markdown"]

def test_scraper_handles_failure(monkeypatch):
    """Test that the scraper handles unsuccessful crawls gracefully."""
    monkeypatch.setenv("FIRECRAWL_API_KEY", "fc-test-key")
    
    mock_app = MagicMock()
    mock_app.crawl_url.return_value = {
        "success": False,
        "error": "Failed to crawl"
    }
    
    scraper = DocumentationScraper()
    scraper.app = mock_app
    
    with pytest.raises(RuntimeError, match="Failed to crawl"):
        scraper.scrape_docs("https://example.com/docs")
