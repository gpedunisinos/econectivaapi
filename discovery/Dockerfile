FROM python:3.11-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 OMP_NUM_THREADS=1 OPENBLAS_NUM_THREADS=1 MKL_NUM_THREADS=1 MALLOC_ARENA_MAX=2
WORKDIR /service
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt && pip install --no-cache-dir https://github.com/explosion/spacy-models/releases/download/pt_core_news_sm-3.8.0/pt_core_news_sm-3.8.0-py3-none-any.whl
COPY app.py .
USER 10001:10001
EXPOSE 8091
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8091", "--workers", "1", "--no-access-log"]
