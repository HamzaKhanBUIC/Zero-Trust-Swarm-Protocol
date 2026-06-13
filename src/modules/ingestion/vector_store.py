import lancedb
from lancedb.embeddings import get_registry
from lancedb.pydantic import LanceModel, Vector
from src.config import get_settings

class VectorStore:
    def __init__(self):
        settings = get_settings()
        self.db = lancedb.connect("./omega_docs.db")
        self.table_name = "documentation"
        
        # We only want to instantiate the embedding function once.
        # Ollama registry uses the host from settings.
        self.embed_func = get_registry().get("ollama").create(
            name="nomic-embed-text",
            host=settings.ollama_base_url
        )

    def _get_schema(self):
        # We define the schema here to ensure the embed_func is correctly bound to it.
        class DocumentChunk(LanceModel):
            text: str = self.embed_func.SourceField()
            # nomic-embed-text generates 768-dimensional vectors
            vector: Vector(768) = self.embed_func.VectorField()
            url: str
        return DocumentChunk

    def upsert_documents(self, url: str, chunks: list[str]):
        """
        Takes a list of Markdown chunks and a source URL, and upserts them into LanceDB.
        """
        data = [{"text": chunk, "url": url} for chunk in chunks]
        
        table_names = self.db.table_names()
        
        if self.table_name not in table_names:
            self.db.create_table(self.table_name, schema=self._get_schema(), data=data)
        else:
            table = self.db.open_table(self.table_name)
            # Delete any existing chunks for this URL (simulating upsert)
            table.delete(f"url = '{url}'")
            table.add(data)
