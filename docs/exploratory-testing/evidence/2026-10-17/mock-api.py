"""In-memory Prosie API fixture for the 2026-10-17 exploratory pass.

Request and response shapes follow routes/api.php and the CLI client.
GET /_fixture/requests and GET /_fixture/state expose captures.
POST /_fixture/mode sets stream, message, and delay behaviour.
POST /_fixture/reset clears stored data and the request log.
"""
import json
import time
from email.parser import BytesParser
from email.policy import HTTP
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit

state = {
    "books": {},
    "chapters": {},
    "conversations": {},
    "messages": {},
    "llm": {
        "provider": "openrouter",
        "model": "openai/gpt-test",
        "has_openai_key": False,
        "has_anthropic_key": False,
        "has_openrouter_key": False,
        "keys": {},
    },
}
requests = []
mode = {"stream": "normal", "fail_message": False, "delay": 0, "polls": 0}
ids = {"n": 100}


def next_id():
    ids["n"] += 1
    return ids["n"]


def now():
    return "2026-10-17T12:00:00+00:00"


def words(text):
    plain = text.replace("<p>", " ").replace("</p>", " ").replace("<br>", " ")
    return len([part for part in plain.split() if part])


def parse_multipart(content_type, raw):
    message = BytesParser(policy=HTTP).parsebytes(
        b"Content-Type: " + content_type.encode() + b"\r\n\r\n" + raw
    )
    fields, files = {}, {}
    for part in message.iter_parts():
        name = part.get_param("name", header="content-disposition")
        filename = part.get_filename()
        data = part.get_payload(decode=True) or b""
        if filename is not None:
            files[name] = {"filename": filename, "size": len(data), "data": data}
        else:
            fields[name] = data.decode()
    return fields, files


def chapter_view(chapter):
    view = dict(chapter)
    view["word_count"] = words(chapter.get("content") or "")
    return view


def book_view(book, with_chapters):
    view = dict(book)
    chapters = [chapter_view(c) for c in state["chapters"].values() if c["story_id"] == book["id"]]
    chapters.sort(key=lambda item: (item["order"], item["id"]))
    view["word_count"] = sum(item["word_count"] for item in chapters)
    if with_chapters:
        view["chapters"] = chapters
    else:
        view.pop("chapters", None)
    return view


def markdown_for(text):
    if "<p>" not in (text or ""):
        return text or ""
    parts = []
    for chunk in text.split("</p>"):
        plain = chunk.replace("<p>", "").replace("<br>", "\n").strip()
        if plain:
            parts.append(plain)
    return "\n\n".join(parts) + ("\n" if parts else "")


def reset():
    state["books"].clear()
    state["chapters"].clear()
    state["conversations"].clear()
    state["messages"].clear()
    requests.clear()
    mode.update({"stream": "normal", "fail_message": False, "delay": 0, "polls": 0})
    ids["n"] = 100


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *_args):
        return

    def send_json(self, status, value):
        data = json.dumps(value).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def send_bytes(self, status, data, content_type):
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def read_body(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        ctype = self.headers.get("Content-Type", "")
        entry = {
            "method": self.command,
            "path": self.path,
            "authorization": self.headers.get("Authorization", ""),
            "content_type": ctype,
        }
        if ctype.startswith("multipart/form-data"):
            fields, files = parse_multipart(ctype, raw)
            entry["body"] = {
                "fields": fields,
                "files": {key: {"filename": value["filename"], "size": value["size"]} for key, value in files.items()},
            }
            requests.append(entry)
            return {"fields": fields, "files": files}
        try:
            body = json.loads(raw) if raw else None
        except Exception:
            body = raw.decode(errors="replace")
        entry["body"] = body
        requests.append(entry)
        return body

    def authorized(self):
        if self.path.startswith("/_fixture") or self.path.startswith("/oauth/"):
            return True
        header = self.headers.get("Authorization", "")
        return header.startswith("Bearer ") and len(header) > len("Bearer ")

    def do_GET(self):
        requests.append({"method": "GET", "path": self.path, "authorization": self.headers.get("Authorization", "")})
        self.route("GET", None)

    def start_sse(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Connection", "close")
        self.end_headers()

    def do_POST(self):
        self.route("POST", self.read_body())

    def do_PATCH(self):
        self.route("PATCH", self.read_body())

    def do_DELETE(self):
        self.route("DELETE", self.read_body())

    def route(self, method, body):
        if not self.authorized():
            self.send_json(401, {"message": "unauthorized: invalid or missing authentication token"})
            return
        path = urlsplit(self.path).path
        query = parse_qs(urlsplit(self.path).query)
        if path == "/_fixture/requests":
            self.send_json(200, requests)
            return
        if path == "/_fixture/state":
            self.send_json(200, state)
            return
        if path == "/_fixture/mode" and method == "POST":
            mode.update(body or {})
            self.send_json(200, mode)
            return
        if path == "/_fixture/reset" and method == "POST":
            reset()
            self.send_json(200, {"reset": True})
            return
        if path == "/api/user" and method == "GET":
            self.send_json(200, {"id": 7, "name": "Ada Writer", "email": "ada@example.com"})
            return
        if path == "/oauth/device/code" and method == "POST":
            self.send_json(200, {
                "device_code": "device-1",
                "user_code": "ABCD-EFGH",
                "verification_uri": "https://prosie.test/activate",
                "expires_in": 30,
                "interval": 1,
            })
            return
        if path == "/oauth/token" and method == "POST":
            mode["polls"] += 1
            if mode["polls"] == 1:
                self.send_json(400, {"error": "authorization_pending"})
                return
            self.send_json(200, {"access_token": "device-token", "token_type": "Bearer", "scope": "read write generate"})
            return
        if path == "/api/stories" and method == "GET":
            self.send_json(200, {"data": [book_view(book, False) for book in state["books"].values()]})
            return
        if path == "/api/stories" and method == "POST":
            self.create_book(body or {})
            return
        if path == "/api/stories/import-docx" and method == "POST":
            self.import_docx(body)
            return
        if path.startswith("/api/stories/"):
            self.story(method, path, query, body)
            return
        if path.startswith("/api/scenes/"):
            self.scene(method, path, query, body)
            return
        if path.startswith("/api/conversations/"):
            self.conversation(method, path, body)
            return
        if path == "/api/llm-config" and method == "GET":
            self.send_json(200, {"data": self.public_llm()})
            return
        if path == "/api/llm-config" and method == "PATCH":
            self.update_llm(body or {})
            return
        if path == "/api/llm-models" and method == "GET":
            self.send_json(200, {"data": {
                "providers": {"openrouter": {"openai/gpt-test": "GPT Test"}, "openai": {"gpt-test": "GPT Test"}},
                "configured": [name for name, present in {
                    "openai": state["llm"]["has_openai_key"],
                    "anthropic": state["llm"]["has_anthropic_key"],
                    "openrouter": state["llm"]["has_openrouter_key"],
                }.items() if present],
            }})
            return
        self.send_json(404, {"message": "Not Found"})

    def create_book(self, body):
        book_id = next_id()
        chapter_id = next_id()
        book = {
            "id": book_id,
            "title": body.get("title") or "Untitled",
            "pov": "third",
            "tense": "past",
            "lore": body.get("lore"),
            "characters": body.get("characters"),
            "story_so_far": body.get("story_so_far"),
            "target_word_count": body.get("target_word_count"),
            "created_at": now(),
            "updated_at": now(),
        }
        chapter = {
            "id": chapter_id,
            "story_id": book_id,
            "order": 0,
            "name": "Chapter 1",
            "content": "",
            "summary": None,
            "created_at": now(),
            "updated_at": now(),
        }
        state["books"][str(book_id)] = book
        state["chapters"][str(chapter_id)] = chapter
        self.send_json(201, {"data": book_view(book, True)})

    def import_docx(self, body):
        files = (body or {}).get("files", {})
        document = files.get("document")
        if not document or not document["data"].startswith(b"PK"):
            self.send_json(422, {"message": "The document does not contain importable content.", "errors": {"document": ["The document does not contain importable content."]}})
            return
        title = (body.get("fields") or {}).get("title") or "Imported Book"
        book_id = next_id()
        chapter_id = next_id()
        book = {"id": book_id, "title": title, "pov": "third", "tense": "past", "lore": None, "characters": None, "story_so_far": None, "target_word_count": None, "created_at": now(), "updated_at": now()}
        chapter = {"id": chapter_id, "story_id": book_id, "order": 0, "name": "Imported", "content": "Imported paragraph one.\n\nImported paragraph two.", "summary": None, "created_at": now(), "updated_at": now()}
        state["books"][str(book_id)] = book
        state["chapters"][str(chapter_id)] = chapter
        self.send_json(201, {"data": book_view(book, True), "scenes": [chapter_view(chapter)]})

    def story(self, method, path, query, body):
        parts = path.split("/")
        book = state["books"].get(parts[3])
        if book is None and parts[3] != "import-docx":
            self.send_json(404, {"message": "Not Found"})
            return
        if len(parts) == 4 and method == "GET":
            self.send_json(200, {"data": book_view(book, True)})
            return
        if len(parts) == 4 and method == "PATCH":
            for key in ("title", "lore", "characters", "story_so_far", "target_word_count"):
                if key in body:
                    book[key] = body[key]
            book["updated_at"] = now()
            self.send_json(200, {"data": book_view(book, True)})
            return
        if len(parts) == 4 and method == "DELETE":
            for key, chapter in list(state["chapters"].items()):
                if chapter["story_id"] == book["id"]:
                    del state["chapters"][key]
            del state["books"][parts[3]]
            self.send_response(204)
            self.send_header("Content-Length", "0")
            self.end_headers()
            return
        if len(parts) == 5 and parts[4] == "duplicate" and method == "POST":
            self.duplicate(book)
            return
        if len(parts) == 5 and parts[4] == "export" and method == "GET":
            self.export_book(book, query.get("format", ["markdown"])[0])
            return
        if len(parts) == 5 and parts[4] == "scenes" and method == "GET":
            chapters = [chapter_view(c) for c in state["chapters"].values() if c["story_id"] == book["id"]]
            chapters.sort(key=lambda item: (item["order"], item["id"]))
            self.send_json(200, {"data": chapters})
            return
        if len(parts) == 5 and parts[4] == "scenes" and method == "POST":
            self.create_chapter(book, body or {})
            return
        if len(parts) == 6 and parts[4] == "scenes" and parts[5] == "reorder" and method == "POST":
            self.reorder(book, body or {})
            return
        if len(parts) == 5 and parts[4] == "conversations" and method == "GET":
            rows = [self.conversation_view(item, False) for item in state["conversations"].values() if item["story_id"] == book["id"]]
            self.send_json(200, {"data": rows})
            return
        if len(parts) == 5 and parts[4] == "conversations" and method == "POST":
            self.create_conversation(book, body or {})
            return
        if len(parts) == 6 and parts[5] == "import" and method == "POST":
            self.import_chat(book, body)
            return
        self.send_json(404, {"message": "Not Found"})

    def duplicate(self, book):
        copy = dict(book)
        copy["id"] = next_id()
        copy["title"] = book["title"] + " copy"
        copy["created_at"] = now()
        copy["updated_at"] = now()
        state["books"][str(copy["id"])] = copy
        for chapter in list(state["chapters"].values()):
            if chapter["story_id"] != book["id"]:
                continue
            cloned = dict(chapter)
            cloned["id"] = next_id()
            cloned["story_id"] = copy["id"]
            state["chapters"][str(cloned["id"])] = cloned
        self.send_json(201, {"data": book_view(copy, True)})

    def export_book(self, book, fmt):
        chapters = [chapter_view(c) for c in state["chapters"].values() if c["story_id"] == book["id"]]
        chapters.sort(key=lambda item: (item["order"], item["id"]))
        if fmt == "docx":
            self.send_bytes(200, b"PK\x03\x04" + book["title"].encode(), "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
            return
        blocks = ["# " + book["title"], ""]
        for chapter in chapters:
            blocks.append("## " + chapter["name"])
            blocks.append("")
            blocks.append(markdown_for(chapter["content"]))
            blocks.append("")
        self.send_bytes(200, "\n".join(blocks).encode(), "text/markdown")

    def create_chapter(self, book, body):
        orders = [c["order"] for c in state["chapters"].values() if c["story_id"] == book["id"]]
        order = body.get("order", (max(orders) + 1) if orders else 0)
        name = (body.get("name") or "").strip() or f"Chapter {order + 1}"
        chapter = {
            "id": next_id(),
            "story_id": book["id"],
            "order": order,
            "name": name,
            "content": body.get("content") or "",
            "summary": body.get("summary"),
            "created_at": now(),
            "updated_at": now(),
        }
        state["chapters"][str(chapter["id"])] = chapter
        self.send_json(201, {"data": chapter_view(chapter)})

    def reorder(self, book, body):
        wanted = []
        for raw in body.get("scene_ids") or []:
            key = str(raw)
            if key in wanted:
                continue
            chapter = state["chapters"].get(key)
            if chapter and chapter["story_id"] == book["id"]:
                wanted.append(key)
        rest = [str(c["id"]) for c in sorted(
            [c for c in state["chapters"].values() if c["story_id"] == book["id"] and str(c["id"]) not in wanted],
            key=lambda item: (item["order"], item["id"]),
        )]
        for index, key in enumerate(wanted + rest):
            state["chapters"][key]["order"] = index
        chapters = [chapter_view(state["chapters"][key]) for key in wanted + rest]
        self.send_json(200, {"data": chapters})

    def scene(self, method, path, query, body):
        parts = path.split("/")
        chapter = state["chapters"].get(parts[3])
        if chapter is None:
            self.send_json(404, {"message": "Not Found"})
            return
        action = parts[4] if len(parts) > 4 else ""
        if action == "" and method == "GET":
            self.send_json(200, {"data": chapter_view(chapter)})
            return
        if action == "" and method == "PATCH":
            for key, field in (("name", "name"), ("content", "content"), ("summary", "summary")):
                if field in (body or {}):
                    chapter[key] = body[field]
            chapter["updated_at"] = now()
            self.send_json(200, {"data": chapter_view(chapter)})
            return
        if action == "" and method == "DELETE":
            siblings = [c for c in state["chapters"].values() if c["story_id"] == chapter["story_id"]]
            if len(siblings) <= 1:
                self.send_json(422, {"message": "A book must keep at least one chapter."})
                return
            del state["chapters"][parts[3]]
            self.send_json(200, {"deleted": True})
            return
        if action == "export" and method == "GET":
            fmt = query.get("format", ["markdown"])[0]
            if fmt == "docx":
                self.send_bytes(200, b"PK\x03\x04chapter", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
                return
            text = "# " + chapter["name"] + "\n\n" + markdown_for(chapter["content"])
            if not text.endswith("\n"):
                text += "\n"
            self.send_bytes(200, text.encode(), "text/markdown")
            return
        if action in ("continue", "rewrite", "summarize") or path.endswith("/continue/stream") or path.endswith("/rewrite/stream") or path.endswith("/continue/cancel") or path.endswith("/reject-continuation") or path.endswith("/rewrite/undo"):
            self.generate(method, path, chapter, body or {})
            return
        self.send_json(404, {"message": "Not Found"})

    def generate(self, method, path, chapter, body):
        if path.endswith("/continue/cancel") and method == "POST":
            self.send_json(200, {"cancelled": True})
            return
        if path.endswith("/reject-continuation") and method == "POST":
            content = chapter["content"]
            marker = '<p class="llm-generated">'
            if marker in content:
                chapter["content"] = content[:content.rfind(marker)]
            self.send_json(200, {"data": chapter_view(chapter)})
            return
        if path.endswith("/rewrite/undo") and method == "POST":
            previous = chapter.get("before_rewrite")
            if previous is None:
                self.send_json(422, {"message": "No rewrite to undo."})
                return
            chapter["content"] = previous
            chapter["before_rewrite"] = None
            self.send_json(200, {"data": chapter_view(chapter)})
            return
        if mode["stream"] == "conflict" and ("continue" in path or "rewrite" in path or path.endswith("/summarize")):
            prose = "The unsaved ending.\n"
            if path.endswith("/stream"):
                self.sse_conflict(prose)
                return
            self.send_json(409, {"message": "This chapter changed. Nothing was saved.", "error": "conflict", "generated_content": prose})
            return
        if path.endswith("/summarize") and method == "POST":
            summary = "A cold hall and a closed door."
            if body.get("persist", True):
                chapter["summary"] = summary
            self.send_json(200, {"data": {"summary": summary, "persisted": bool(body.get("persist", True)), "model": "fixture"}})
            return
        if "/rewrite" in path and method == "POST":
            self.rewrite(path, chapter, body)
            return
        if "/continue" in path and method == "POST":
            self.continue_chapter(path, chapter, body)
            return
        self.send_json(404, {"message": "Not Found"})

    def continue_chapter(self, path, chapter, body):
        prose = "The lantern guttered. She opened the door."
        if body.get("instruction"):
            prose = "She ended the story as asked."
        html = f'<p class="llm-generated">{prose}</p>'
        if path.endswith("/stream"):
            self.sse_prose(path, prose, html, chapter, body.get("persist", True), "continue")
            return
        if body.get("persist", True):
            chapter["content"] = (chapter.get("content") or "") + html
        self.send_json(200, {"data": {"prose": prose, "html": html, "persisted": bool(body.get("persist", True)), "model": "fixture"}})

    def rewrite(self, path, chapter, body):
        selection = body.get("selection") or ""
        prose = "Tighter: " + selection.strip()
        if path.endswith("/stream"):
            self.sse_prose(path, prose, prose, chapter, body.get("persist", False), "rewrite")
            return
        applied = (chapter.get("content") or "").replace(selection, prose, 1)
        if body.get("persist"):
            chapter["before_rewrite"] = chapter.get("content")
            chapter["content"] = applied
        self.send_json(200, {"data": {"prose": prose, "persisted": bool(body.get("persist")), "applied_content": applied, "model": "fixture"}})

    def sse_prose(self, path, prose, html, chapter, persist, kind):
        if mode["stream"] == "error":
            self.start_sse()
            self.wfile.write(b'event: delta\ndata: {"delta":"Partial "}\n\n')
            self.wfile.write(b'event: error\ndata: {"message":"Upstream provider failed."}\n\n')
            self.close_connection = True
            return
        tokens = [prose[: len(prose) // 2], prose[len(prose) // 2 :]]
        if mode["stream"] == "truncate":
            tokens = tokens[:1]
        self.start_sse()
        for token in tokens:
            if mode.get("delay"):
                time.sleep(float(mode["delay"]))
            self.wfile.write(f'event: delta\ndata: {json.dumps({"delta": token})}\n\n'.encode())
            self.wfile.flush()
        if mode["stream"] == "truncate":
            self.close_connection = True
            return
        if persist and kind == "continue":
            chapter["content"] = (chapter.get("content") or "") + html
        if persist and kind == "rewrite":
            chapter["before_rewrite"] = chapter.get("content")
            chapter["content"] = prose
        done = {"prose": prose, "persisted": bool(persist), "model": "fixture"}
        if kind == "continue":
            done["html"] = html
        self.wfile.write(f"event: done\ndata: {json.dumps(done)}\n\n".encode())
        self.close_connection = True

    def sse_conflict(self, prose):
        self.start_sse()
        self.wfile.write(b'event: delta\ndata: {"delta":"The unsaved "}\n\n')
        payload = json.dumps({"message": "This chapter changed. Nothing was saved.", "error": "conflict", "generated_content": prose})
        self.wfile.write(f"event: error\ndata: {payload}\n\n".encode())
        self.close_connection = True

    def create_conversation(self, book, body):
        title = body.get("title") or None
        conversation = {"id": next_id(), "story_id": book["id"], "title": title, "model": state["llm"]["model"], "fidelity": "manuscript", "expires_in_days": 14, "created_at": now(), "updated_at": now()}
        state["conversations"][str(conversation["id"])] = conversation
        self.send_json(201, {"data": self.conversation_view(conversation, True)})

    def import_chat(self, book, body):
        files = (body or {}).get("files", {})
        uploaded = files.get("file")
        if not uploaded:
            self.send_json(422, {"message": "The file field is required.", "errors": {"file": ["The file field is required."]}})
            return
        try:
            payload = json.loads(uploaded["data"].decode())
        except Exception:
            self.send_json(422, {"message": "The conversation file is not valid JSON.", "errors": {"file": ["The conversation file is not valid JSON."]}})
            return
        title = payload.get("title") or "Imported chat"
        conversation = {"id": next_id(), "story_id": book["id"], "title": title, "model": "fixture", "fidelity": "manuscript", "expires_in_days": 14, "created_at": now(), "updated_at": now()}
        state["conversations"][str(conversation["id"])] = conversation
        for item in payload.get("messages") or []:
            message = {"id": next_id(), "conversation_id": conversation["id"], "role": item.get("role") or "user", "content": item.get("content") or "", "created_at": now()}
            state["messages"][str(message["id"])] = message
        self.send_json(201, {"data": self.conversation_view(conversation, True)})

    def conversation(self, method, path, body):
        parts = path.split("/")
        conversation = state["conversations"].get(parts[3])
        if conversation is None:
            self.send_json(404, {"message": "Not Found"})
            return
        if len(parts) == 4 and method == "GET":
            self.send_json(200, {"data": self.conversation_view(conversation, True)})
            return
        if len(parts) == 4 and method == "DELETE":
            for key, message in list(state["messages"].items()):
                if message["conversation_id"] == conversation["id"]:
                    del state["messages"][key]
            del state["conversations"][parts[3]]
            self.send_json(200, {"deleted": True})
            return
        if len(parts) == 5 and parts[4] == "export" and method == "GET":
            self.send_bytes(200, json.dumps(self.conversation_view(conversation, True), indent=2).encode() + b"\n", "application/json")
            return
        if len(parts) == 5 and parts[4] == "messages" and method == "POST":
            self.send_message(conversation, body or {}, stream=False)
            return
        if len(parts) == 6 and parts[5] == "stream" and method == "POST":
            self.send_message(conversation, body or {}, stream=True)
            return
        self.send_json(404, {"message": "Not Found"})

    def send_message(self, conversation, body, stream):
        if mode.get("fail_message"):
            self.send_json(502, {"message": "The assistant did not answer."})
            return
        user = {"id": next_id(), "conversation_id": conversation["id"], "role": "user", "content": body.get("content") or "", "created_at": now()}
        answer = "The door stays shut until chapter three."
        assistant = {"id": next_id(), "conversation_id": conversation["id"], "role": "assistant", "content": answer, "created_at": now()}
        state["messages"][str(user["id"])] = user
        state["messages"][str(assistant["id"])] = assistant
        conversation["updated_at"] = now()
        payload = {"message": assistant, "model": "fixture", "usage": {"prompt_tokens": 3, "completion_tokens": 8, "total_tokens": 11}}
        if not stream:
            self.send_json(200, {"data": payload})
            return
        if mode["stream"] == "truncate":
            self.start_sse()
            self.wfile.write(b'event: delta\ndata: {"delta":"The door "}\n\n')
            self.close_connection = True
            return
        self.start_sse()
        self.wfile.write(b'event: delta\ndata: {"delta":"The door stays shut until chapter three."}\n\n')
        self.wfile.write(f"event: done\ndata: {json.dumps(payload)}\n\n".encode())
        self.close_connection = True

    def conversation_view(self, conversation, with_messages):
        view = dict(conversation)
        if with_messages:
            view["messages"] = [m for m in state["messages"].values() if m["conversation_id"] == conversation["id"]]
        return view

    def public_llm(self):
        return {
            "provider": state["llm"]["provider"],
            "model": state["llm"]["model"],
            "has_openai_key": state["llm"]["has_openai_key"],
            "has_anthropic_key": state["llm"]["has_anthropic_key"],
            "has_openrouter_key": state["llm"]["has_openrouter_key"],
        }

    def update_llm(self, body):
        secret_keys = ("openai_api_key", "anthropic_api_key", "openrouter_api_key")
        for key in secret_keys:
            if key in body:
                state["llm"]["keys"][key] = body[key]
        if body.get("openai_api_key"):
            state["llm"]["has_openai_key"] = True
        if body.get("anthropic_api_key"):
            state["llm"]["has_anthropic_key"] = True
        if body.get("openrouter_api_key"):
            state["llm"]["has_openrouter_key"] = True
        if "provider" in body:
            state["llm"]["provider"] = body["provider"]
        if "model" in body:
            state["llm"]["model"] = body["model"]
        public = self.public_llm()
        leaked = [key for key in secret_keys if key in public]
        public["leaked_keys"] = leaked
        self.send_json(200, {"data": {k: v for k, v in public.items() if k != "leaked_keys"}})


if __name__ == "__main__":
    import sys
    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    port = server.server_address[1]
    sys.stdout.write(f"{port}\n")
    sys.stdout.flush()
    server.serve_forever()
