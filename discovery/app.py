"""Internal, CPU-only reference extractor. No external requests or text logging."""
import os
import re
import unicodedata
import threading
from contextlib import asynccontextmanager

import spacy
from fastapi import FastAPI
from pydantic import BaseModel, Field
from spacy.matcher import PhraseMatcher


def normalize(text):
    return " ".join("".join(c.lower() if c.isalnum() else " " for c in unicodedata.normalize("NFKD", text) if not unicodedata.combining(c)).split())


def normalized_offsets(text):
    chars, offsets = [], []
    for index, original in enumerate(text):
        for char in unicodedata.normalize("NFKD", original).lower():
            if unicodedata.combining(char):
                continue
            if char.isalnum():
                chars.append(char)
                offsets.append(index)
            elif chars and chars[-1] != " ":
                chars.append(" ")
                offsets.append(index)
    return "".join(chars), offsets


@asynccontextmanager
async def lifespan(app):
    global nlp
    nlp = spacy.load("pt_core_news_sm", exclude=["parser", "lemmatizer", "morphologizer", "attribute_ruler"])
    yield


app = FastAPI(lifespan=lifespan)
nlp = None
analysis_lock = threading.Lock()


class Entry(BaseModel):
    id: int
    names: list[str] = Field(max_length=30)


class Request(BaseModel):
    source_id: int
    text: str = Field(max_length=200000)
    name: str = Field(max_length=255)
    entries: list[Entry] = Field(max_length=10000)


def extract(request, pipeline):
    matcher = PhraseMatcher(pipeline.vocab, attr="LOWER")
    targets = {}
    for entry in request.entries:
        patterns = set(normalize(name) for name in entry.names)
        patterns = {p for p in patterns if len(p) >= 2 and p not in {"de", "da", "do", "das", "dos", "em", "na", "no", "as", "os", "um", "uma", "ao", "eu", "se", "ou", "com", "para", "pessoa", "projeto", "pesquisa", "educacao", "outros"}}
        if patterns:
            key = str(entry.id)
            matcher.add(key, [pipeline.make_doc(p) for p in sorted(patterns)])
            targets[key] = entry.id
    normalized, offsets = normalized_offsets(request.text)
    doc = pipeline.make_doc(normalized)
    found = {}
    recognized = set()
    for match, start, end in matcher(doc):
        span = doc[start:end]
        target = targets[pipeline.vocab.strings[match]]
        original_start, original_end = offsets[span.start_char], offsets[span.end_char - 1] + 1
        evidence = request.text[max(0, original_start - 70):min(len(request.text), original_end + 90)].strip()
        recognized.add(normalize(span.text))
        # Prefer the longer, more specific mention for the same target.
        if target not in found or len(span.text) > len(found[target]["mention"]):
            found[target] = {"target_id": target, "mention": request.text[original_start:original_end], "evidence": evidence}
    # A pair is only an alias candidate for THIS clue when the long form
    # identifies its own name, never because another institution was mentioned.
    own_names = {normalize(request.name)}
    for entry in request.entries:
        if entry.id == request.source_id:
            own_names.update(normalize(n) for n in entry.names)
    aliases = []
    for match in re.finditer(r"([^\n.!?;()]{3,180})\s*\(([^()\n]{2,80})\)", request.name + "\n" + request.text):
        long, short = match.group(1).strip(), match.group(2).strip()
        if (normalize(long) in own_names or normalize(match.group(0)) == normalize(request.name)) and normalize(short) != normalize(request.name):
            aliases.append({"name": short, "evidence": match.group(0)[:260]})
            if normalize(match.group(0)) == normalize(request.name):
                aliases.append({"name": long, "evidence": match.group(0)[:260]})
    explicit = re.compile(r"([^\n.!?;]{3,180}?),?\s+(?:também conhecid[oa] como|conhecid[oa] como|sigla)\s+[\"“]?([^\n.!?;\"”]{2,80})", re.I)
    for match in explicit.finditer(request.text):
        if normalize(match.group(1).strip()) in own_names:
            aliases.append({"name": match.group(2).strip(), "evidence": match.group(0)[:260]})
    entities = []
    # Chunk long content at whitespace to keep runtime allocations bounded.
    chunks = []
    remaining = request.text
    while remaining:
        cut = min(12000, len(remaining))
        if cut < len(remaining):
            boundary = remaining.rfind(" ", 0, cut)
            if boundary > 6000:
                cut = boundary
        chunks.append(remaining[:cut])
        remaining = remaining[cut:]
    for processed in pipeline.pipe(chunks, batch_size=1):
        for ent in processed.ents:
            value = normalize(ent.text)
            if ent.label_ in {"ORG", "PER", "LOC"} and value not in recognized and value not in own_names and len(value) >= 3:
                if not any(normalize(e["name"]) == value for e in entities):
                    entities.append({"name": ent.text[:120], "kind": ent.label_})
            if len(entities) >= 10:
                break
        if len(entities) >= 10:
            break
    return {"references": list(found.values()), "alias_candidates": aliases[:10], "unresolved": entities[:10]}


@app.get("/health")
def health():
    return {"status": "ok", "model": "pt_core_news_sm", "pipeline": nlp.pipe_names}


@app.post("/analyze")
def analyze(request: Request):
    with analysis_lock:
        return extract(request, nlp)
