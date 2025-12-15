from __future__ import annotations
from contextlib import contextmanager
import psycopg
from psycopg.rows import dict_row
from .config import settings

def get_conn() -> psycopg.Connection:
    return psycopg.connect(settings.database_url, row_factory=dict_row)

@contextmanager
def tx():
    conn = get_conn()
    try:
        with conn:
            with conn.cursor() as cur:
                yield cur
    finally:
        conn.close()
