import os
import psycopg2
from flask import Flask

app = Flask(__name__)


def get_db_connection():
    return psycopg2.connect(os.environ["DATABASE_URL"], connect_timeout=5)


@app.route("/")
def index():
    try:
        conn = get_db_connection()
        cur = conn.cursor()
        cur.execute("SELECT version()")
        version = cur.fetchone()[0]
        cur.close()
        conn.close()
        return f"<h1>Hello from Railway!</h1><p>Postgres: {version}</p>"
    except Exception as e:
        return f"<h1>Hello from Railway!</h1><p>DB error: {e}</p>", 200


@app.route("/health")
def health():
    return "ok", 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
