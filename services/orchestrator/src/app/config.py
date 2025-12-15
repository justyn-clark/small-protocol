import os
from pydantic import BaseModel

class Settings(BaseModel):
    database_url: str = os.getenv("DATABASE_URL", "postgresql://postgres:postgres@db:5432/orchestrator")
    service_name: str = os.getenv("SERVICE_NAME", "orchestrator")

settings = Settings()
