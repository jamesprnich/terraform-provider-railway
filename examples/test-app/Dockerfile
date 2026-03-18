FROM python:3.13.5-slim AS production

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY app.py .

ENV PORT=8080
EXPOSE ${PORT}

CMD gunicorn --bind 0.0.0.0:${PORT} app:app
