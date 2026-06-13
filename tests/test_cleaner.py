from src.modules.extractor.cleaner import clean_markdown

def test_clean_markdown_removes_html_comments():
    raw = "Hello\n<!-- some tracking pixel -->\nWorld!"
    expected = "Hello\n\nWorld!"
    assert clean_markdown(raw) == expected

def test_clean_markdown_collapses_newlines():
    raw = "Line 1\n\n\n\nLine 2\n\n\n\n\nLine 3"
    expected = "Line 1\n\nLine 2\n\nLine 3"
    assert clean_markdown(raw) == expected

def test_clean_markdown_removes_telemetry_links():
    raw = "Here is docs. [track-link](https://telemetry.example.com/click?id=123)"
    expected = "Here is docs. "
    assert clean_markdown(raw) == expected
