import re

def clean_markdown(text: str) -> str:
    """
    Strips HTML comments, collapses excess newlines, and removes known telemetry links.
    """
    # Remove HTML comments
    text = re.sub(r'<!--.*?-->', '', text, flags=re.DOTALL)
    
    # Remove known telemetry/tracking markdown links
    text = re.sub(r'\[.*?\]\(https?://telemetry[^\)]+\)', '', text)
    
    # Collapse 3 or more newlines into exactly 2 newlines
    text = re.sub(r'\n{3,}', '\n\n', text)
    
    return text
