from firecrawl import FirecrawlApp
from src.config import get_settings

class DocumentationScraper:
    def __init__(self):
        settings = get_settings()
        self.app = FirecrawlApp(api_key=settings.firecrawl_api_key)

    def scrape_docs(self, url: str) -> list[dict]:
        """
        Crawls a documentation URL and extracts the content in Markdown format.
        """
        response = self.app.crawl_url(
            url,
            params={
                'limit': 100,
                'scrapeOptions': {'formats': ['markdown']}
            }
        )
        
        if not response.get('success'):
            error_message = response.get('error', 'Unknown error during crawl')
            raise RuntimeError(error_message)
            
        return response.get('data', [])
