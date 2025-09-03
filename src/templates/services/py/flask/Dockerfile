FROM python:3.11-slim

RUN apt-get update && apt-get install -y \
    gcc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

RUN useradd -m -u 1001 python && \
    chown -R python:python /app

USER python

EXPOSE ${PORT:-5000}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:${PORT:-5000}/health').read()" || exit 1

CMD ["gunicorn", "--bind", "0.0.0.0:${PORT:-5000}", "--workers", "2", "app:app"]