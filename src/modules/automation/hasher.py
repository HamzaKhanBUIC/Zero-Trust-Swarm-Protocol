import hashlib
import requests
from typing import Optional

def compute_content_hash(text: str) -> str:
    """
    Computes a SHA-256 hash of the given string to detect changes.
    """
    return hashlib.sha256(text.encode('utf-8')).hexdigest()

def get_url_etag(url: str) -> Optional[str]:
    """
    Issues a HEAD request to the URL and returns the ETag if provided by the server.
    """
    try:
        response = requests.head(url, timeout=10)
        return response.headers.get("ETag")
    except requests.RequestException:
        return None
