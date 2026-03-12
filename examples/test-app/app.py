import os
import time
import psycopg2
from flask import Flask

app = Flask(__name__)


def get_db_connection():
    return psycopg2.connect(os.environ["DATABASE_URL"])


def init_db():
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


with app.app_context():
    for attempt in range(30):
        try:
            init_db()
            break
        except psycopg2.OperationalError:
            if attempt == 29:
                raise
            print(f"Postgres not ready - retrying in 2s (attempt {attempt + 1}/30)...")
            time.sleep(2)


@app.route("/")
def index():
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
