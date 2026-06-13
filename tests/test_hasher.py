import pytest
from unittest.mock import patch, MagicMock
from src.modules.automation.hasher import compute_content_hash, get_url_etag

def test_compute_content_hash_returns_sha256():
    text = "Hello, world!"
    # The SHA-256 hash of "Hello, world!"
    expected_hash = "315f5bdb76d078c43b8ac0064e4a0164612b1fce77c869345bfc94c75894edd3"
    assert compute_content_hash(text) == expected_hash

@patch("src.modules.automation.hasher.requests.head")
def test_get_url_etag_returns_etag_if_present(mock_head):
    mock_response = MagicMock()
    mock_response.headers = {"ETag": '"12345-abcde"'}
    mock_head.return_value = mock_response
    
    etag = get_url_etag("https://example.com")
    assert etag == '"12345-abcde"'
    mock_head.assert_called_once_with("https://example.com", timeout=10)

@patch("src.modules.automation.hasher.requests.head")
def test_get_url_etag_returns_none_if_missing(mock_head):
    mock_response = MagicMock()
    mock_response.headers = {}
    mock_head.return_value = mock_response
    
    etag = get_url_etag("https://example.com")
    assert etag is None
