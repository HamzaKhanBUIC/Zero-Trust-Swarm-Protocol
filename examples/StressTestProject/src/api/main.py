from fastapi import FastAPI, Depends
from sqlalchemy.orm import Session
from pydantic import BaseModel
from typing import List

from src.database.connection import get_db, engine, Base
from src.database.models import Task

# Ensure tables are created
Base.metadata.create_all(bind=engine)

app = FastAPI(title="Tasks API", description="Swarm Stress Test App (Tasks API)")

class TaskCreate(BaseModel):
    title: str

class TaskResponse(BaseModel):
    id: int
    title: str
    status: str

    class Config:
        orm_mode = True
        from_attributes = True

@app.post("/tasks", response_model=TaskResponse)
def create_task(task: TaskCreate, db: Session = Depends(get_db)):
    db_task = Task(title=task.title)
    db.add(db_task)
    db.commit()
    db.refresh(db_task)
    return db_task

@app.get("/tasks", response_model=List[TaskResponse])
def list_tasks(db: Session = Depends(get_db)):
    return db.query(Task).all()
