import os
import psycopg2
from flask import Flask

app = Flask(__name__)

_db_initialized = False


def get_db_connection():
    return psycopg2.connect(os.environ["DATABASE_URL"])


def ensure_db():
    global _db_initialized
    if _db_initialized:
        return
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute(
        "CREATE TABLE IF NOT EXISTS hello ("
        "  id SERIAL PRIMARY KEY,"
        "  message TEXT NOT NULL"
        ")"
    )
    cur.execute("SELECT COUNT(*) FROM hello")
    if cur.fetchone()[0] == 0:
        cur.execute("INSERT INTO hello (message) VALUES ('Hello from Railway!')")
    conn.commit()
    cur.close()
    conn.close()
    _db_initialized = True


@app.route("/")
def index():
    ensure_db()
    conn = get_db_connection()
    cur = conn.cursor()
    cur.execute("SELECT message FROM hello LIMIT 1")
    row = cur.fetchone()
    cur.close()
    conn.close()

    message = row[0] if row else "No message found"
    return f"""<!DOCTYPE html>
<html>
<head><title>Railway Test App</title></head>
<body>
  <h1>{message}</h1>
  <p>Connected to Postgres via Railway private networking.</p>
</body>
</html>"""


@app.route("/health")
def health():
    return "ok", 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
