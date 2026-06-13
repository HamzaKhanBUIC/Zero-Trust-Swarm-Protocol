# Package initializer
from .connection import Base, engine, get_db
from .models import Task

Base.metadata.create_all(bind=engine)
