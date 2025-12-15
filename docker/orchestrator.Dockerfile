FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir -U pip \
  && pip install --no-cache-dir fastapi uvicorn[standard] pydantic psycopg[binary] jsonschema

COPY services/orchestrator/src /app/src
COPY spec /app/spec

ENV PYTHONPATH=/app/src
EXPOSE 8000

CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
