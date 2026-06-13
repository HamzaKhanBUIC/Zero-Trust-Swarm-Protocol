import json
import os
import time
from src.modules.extractor.scraper import DocumentationScraper
from src.modules.extractor.cleaner import clean_markdown
from src.modules.ingestion.chunker import chunk_markdown
from src.modules.ingestion.vector_store import VectorStore
from src.modules.automation.hasher import compute_content_hash, get_url_etag
from src.modules.scheduler.cron import schedule_sync_job

HASH_FILE = ".doc_hashes.json"

def load_hashes() -> dict:
    if os.path.exists(HASH_FILE):
        with open(HASH_FILE, "r") as f:
            return json.load(f)
    return {}

def save_hashes(hashes: dict):
    with open(HASH_FILE, "w") as f:
        json.dump(hashes, f, indent=4)

def sync_documentation(url: str):
    print(f"\n[{time.strftime('%Y-%m-%d %H:%M:%S')}] Starting sync for {url}")
    hashes = load_hashes()
    
    # 1. Check ETag first to save Firecrawl credits
    etag = get_url_etag(url)
    if etag and hashes.get(f"{url}_etag") == etag:
        print(f"  -> ETag matched ({etag}). No changes detected. Skipping.")
        return

    # 2. Scrape Documentation
    print("  -> Scraping documentation via Firecrawl...")
    scraper = DocumentationScraper()
    try:
        docs = scraper.scrape_docs(url)
    except Exception as e:
        print(f"  -> Scrape failed: {e}")
        return
    
    if not docs:
        print("  -> No content returned.")
        return
        
    raw_markdown = docs[0].get("markdown", "")
    
    # 3. Fallback to SHA-256 hash if ETag wasn't available or changed
    content_hash = compute_content_hash(raw_markdown)
    if hashes.get(f"{url}_hash") == content_hash:
        print("  -> Content hash matched. No changes detected. Skipping ingestion.")
        if etag:
            hashes[f"{url}_etag"] = etag
            save_hashes(hashes)
        return
        
    print("  -> Content changes detected. Cleaning and Chunking...")
    # 4. Clean & Chunk
    cleaned_text = clean_markdown(raw_markdown)
    chunks = chunk_markdown(cleaned_text)
    
    print(f"  -> Upserting {len(chunks)} chunks to LanceDB (Local RAG)...")
    # 5. Ingest
    store = VectorStore()
    store.upsert_documents(url, chunks)
    
    # 6. Update cache tracker
    hashes[f"{url}_etag"] = etag
    hashes[f"{url}_hash"] = content_hash
    save_hashes(hashes)
    print("  -> Sync complete! Vector Store updated.")

def main():
    target_url = "https://langchain-ai.github.io/langgraph/"
    
    print("=== LIVE DOCUMENTATION SYNC ===")
    print(f"Targeting: {target_url}")
    
    # 1. Run an immediate sync on startup
    sync_documentation(target_url)
    
    # 2. Schedule the background job
    print("\nStarting background scheduler (interval: 12 hours)...")
    scheduler = schedule_sync_job(lambda: sync_documentation(target_url), hours=12)
    
    try:
        # Keep the orchestrator alive
        while True:
            time.sleep(60)
    except (KeyboardInterrupt, SystemExit):
        scheduler.shutdown()
        print("\nOrchestrator shut down gracefully.")

if __name__ == "__main__":
    main()
