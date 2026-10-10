"""In-memory Prosie API fixture for the 2026-10-10 exploratory pass.

Shapes follow the prosie backend (routes/api.php, Api controllers, Responders).
GET /_fixture/requests and /_fixture/state expose captures.
POST /_fixture/mode {"stream": "normal"|"truncate"|"error"} sets the SSE mode.
"""
import json
import re
import sys
import time
from email.parser import BytesParser
from email.policy import HTTP
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit

state = {"books": {}, "chapters": {}, "conversations": {}, "codex": {}, "series": {}}
requests = []
mode = {"stream": "normal"}
ids = {"n": 100}


def next_id():
    ids["n"] += 1
    return ids["n"]


def now():
    return "2026-10-10T12:00:00+00:00"


def parse_multipart(content_type, raw):
    msg = BytesParser(policy=HTTP).parsebytes(
        b"Content-Type: " + content_type.encode() + b"\r\n\r\n" + raw)
    fields, files = {}, {}
    for part in msg.iter_parts():
        name = part.get_param("name", header="content-disposition")
        filename = part.get_filename()
        data = part.get_payload(decode=True) or b""
        if filename is not None:
            files[name] = {"filename": filename, "size": len(data), "data": data}
        else:
            fields[name] = data.decode()
    return fields, files


def book_view(book):
    view = dict(book)
    view["chapters"] = [c for c in state["chapters"].values() if c["story_id"] == book["id"]]
    return view


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *_args):
        pass

    def send(self, status, value=None, content_type="application/json"):
        data = b"" if value is None else (value if isinstance(value, bytes) else json.dumps(value).encode())
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def read(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        ctype = self.headers.get("Content-Type", "")
        entry = {"method": self.command, "path": self.path, "content_type": ctype}
        body = None
        if ctype.startswith("multipart/form-data"):
            fields, files = parse_multipart(ctype, raw)
            body = {"fields": fields, "files": {k: {"filename": v["filename"], "size": v["size"]} for k, v in files.items()}}
            entry["body"] = body
            body = {"fields": fields, "files": files}
        else:
            try:
                body = json.loads(raw) if raw else None
            except Exception:
                body = raw.decode(errors="replace")
            entry["body"] = body
        if not self.path.startswith("/_fixture"):
            requests.append(entry)
        return body

    def authorized(self):
        if self.headers.get("Authorization") != "Bearer test-token":
            self.send(401, {"message": "Unauthenticated."})
            return False
        return True

    def invalid(self, errors):
        first = next(iter(errors.values()))[0]
        self.send(422, {"message": first, "errors": errors})

    def sse(self, events, close_early=False):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Transfer-Encoding", "chunked")
        self.end_headers()
        for event, data in events:
            chunk = ("event: %s\ndata: %s\n\n" % (event, json.dumps(data))).encode()
            self.wfile.write(b"%x\r\n%s\r\n" % (len(chunk), chunk))
            self.wfile.flush()
            time.sleep(0.05)
        self.wfile.write(b"0\r\n\r\n")
        self.wfile.flush()

    # ---- GET ----
    def do_GET(self):
        self.read()
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        if path == "/_fixture/requests":
            return self.send(200, requests)
        if path == "/_fixture/state":
            return self.send(200, state)
        if path == "/api/user":
            return self.send(200, {"id": 17, "name": "Isolated Tester", "email": "tester@example.invalid"})
        if path == "/api/stories":
            return self.send(200, {"data": [book_view(b) for b in state["books"].values()]})
        if path == "/api/series":
            return self.send(200, {"data": list(state["series"].values())})
        m = re.fullmatch(r"/api/stories/(\d+)", path)
        if m:
            b = state["books"].get(int(m.group(1)))
            return self.send(200, {"data": book_view(b)}) if b else self.send(404, {"message": "Not Found"})
        m = re.fullmatch(r"/api/stories/(\d+)/scenes", path)
        if m:
            return self.send(200, {"data": [c for c in state["chapters"].values() if c["story_id"] == int(m.group(1))]})
        m = re.fullmatch(r"/api/stories/(\d+)/codex-entries", path)
        if m:
            if int(m.group(1)) not in state["books"]:
                return self.send(404, {"message": "Not Found"})
            return self.send(200, {"data": [e for e in state["codex"].values() if e["story_id"] == int(m.group(1))]})
        m = re.fullmatch(r"/api/stories/(\d+)/conversations", path)
        if m:
            return self.send(200, {"data": [c for c in state["conversations"].values() if c["story_id"] == int(m.group(1))]})
        m = re.fullmatch(r"/api/conversations/(\d+)", path)
        if m:
            c = state["conversations"].get(int(m.group(1)))
            return self.send(200, {"data": c}) if c else self.send(404, {"message": "Not Found"})
        m = re.fullmatch(r"/api/scenes/(\d+)", path)
        if m:
            c = state["chapters"].get(int(m.group(1)))
            return self.send(200, {"data": c}) if c else self.send(404, {"message": "Not Found"})
        m = re.fullmatch(r"/api/codex-entries/(\d+)", path)
        if m:
            e = state["codex"].get(int(m.group(1)))
            return self.send(200, {"data": e}) if e else self.send(404, {"message": "Not Found"})
        m = re.fullmatch(r"/api/series/(\d+)", path)
        if m:
            s = state["series"].get(int(m.group(1)))
            return self.send(200, {"data": s}) if s else self.send(404, {"message": "Not Found"})
        return self.send(404, {"message": "Not Found"})

    # ---- POST / PUT / PATCH ----
    def do_POST(self):
        body = self.read()
        if self.path == "/_fixture/mode":
            mode.update(body or {})
            return self.send(200, mode)
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        if path == "/api/stories":
            if not (body or {}).get("title"):
                return self.invalid({"title": ["The title field is required."]})
            bid = next_id()
            state["books"][bid] = {"id": bid, "title": body["title"], "word_count": 0, "created_at": now(), "updated_at": now()}
            cid = next_id()
            state["chapters"][cid] = {"id": cid, "story_id": bid, "title": "Chapter 1", "content": "", "order": 0}
            return self.send(201, {"data": book_view(state["books"][bid])})
        if path == "/api/stories/import-docx":
            files, fields = (body or {}).get("files", {}), (body or {}).get("fields", {})
            doc = files.get("document")
            if not doc:
                return self.invalid({"document": ["The document field is required."]})
            if not doc["filename"].lower().endswith(".docx") or not doc["data"].startswith(b"PK"):
                return self.invalid({"document": ["The document field must be a file of type: docx."]})
            if not fields.get("title"):
                return self.invalid({"title": ["The title field is required."]})
            bid = next_id()
            state["books"][bid] = {"id": bid, "title": fields["title"], "word_count": 3, "created_at": now(), "updated_at": now()}
            cid = next_id()
            state["chapters"][cid] = {"id": cid, "story_id": bid, "title": "Chapter 1", "content": "Imported prose here.", "order": 0}
            scenes = [c for c in state["chapters"].values() if c["story_id"] == bid]
            return self.send(201, {"data": dict(state["books"][bid]), "scenes": scenes})
        m = re.fullmatch(r"/api/stories/(\d+)/scenes", path)
        if m:
            bid = int(m.group(1))
            if bid not in state["books"]:
                return self.send(404, {"message": "Not Found"})
            cid = next_id()
            state["chapters"][cid] = {"id": cid, "story_id": bid, "title": (body or {}).get("title", ""),
                                      "content": (body or {}).get("content", ""), "order": len(state["chapters"])}
            return self.send(201, {"data": state["chapters"][cid]})
        m = re.fullmatch(r"/api/stories/(\d+)/conversations/import", path)
        if m:
            bid = int(m.group(1))
            if bid not in state["books"]:
                return self.send(404, {"message": "Not Found"})
            f = (body or {}).get("files", {}).get("file")
            try:
                data = json.loads(f["data"]) if f else None
                msgs = data["messages"]
            except Exception:
                return self.invalid({"file": ["The conversation export is not valid JSON."]})
            cid = next_id()
            conv = {"id": cid, "story_id": bid, "title": data.get("title"), "fidelity": "full",
                    "messages": [{"id": next_id(), "role": x["role"], "content": x["content"]} for x in msgs],
                    "created_at": now(), "updated_at": now()}
            state["conversations"][cid] = conv
            return self.send(201, {"data": conv})
        m = re.fullmatch(r"/api/stories/(\d+)/codex-entries", path)
        if m:
            bid = int(m.group(1))
            if bid not in state["books"]:
                return self.send(404, {"message": "Not Found"})
            errs = {}
            if (body or {}).get("category") not in ("lore", "character"):
                errs["category"] = ["The category field is required." if not (body or {}).get("category") else "The selected category is invalid."]
            if not (body or {}).get("name"):
                errs["name"] = ["The name field is required."]
            if not (body or {}).get("content"):
                errs["content"] = ["The content field is required."]
            if errs:
                return self.invalid(errs)
            eid = next_id()
            e = {"id": eid, "story_id": bid, "series_id": None, "parent_id": None, "category": body["category"],
                 "name": body["name"], "aliases": body.get("aliases"), "content": body["content"],
                 "order": body.get("order", 0), "created_at": now(), "updated_at": now()}
            state["codex"][eid] = e
            return self.send(201, {"data": e})
        if path == "/api/series":
            if not (body or {}).get("name"):
                return self.invalid({"name": ["The name field is required."]})
            sid = next_id()
            s = {"id": sid, "name": body["name"], "description": body.get("description"), "story_ids": [],
                 "created_at": now(), "updated_at": now()}
            state["series"][sid] = s
            return self.send(201, {"data": s})
        m = re.fullmatch(r"/api/scenes/(\d+)/(continue|continue/stream|rewrite|rewrite/stream|rewrite/undo|summarize|reject-continuation|continue/cancel)", path)
        if m:
            return self.generation(int(m.group(1)), m.group(2), body or {})
        return self.send(404, {"message": "Not Found"})

    def generation(self, cid, action, body):
        ch = state["chapters"].get(cid)
        if not ch:
            return self.send(404, {"message": "Not Found"})
        if action == "continue/cancel":
            return self.send(204)
        if action in ("rewrite/undo", "reject-continuation"):
            prev = ch.pop("_before_" + ("rewrite" if action == "rewrite/undo" else "continue"), None)
            if prev is None:
                return self.invalid({"scene": ["There is nothing to undo."]})
            ch["content"] = prev
            return self.send(200, {"data": ch})
        if action == "summarize":
            summary = "A test summary."
            if body.get("persist", True):
                ch["summary"] = summary
            return self.send(200, {"data": {"summary": summary, "persisted": body.get("persist", True), "model": "fixture/model"}})
        if action.startswith("rewrite"):
            if not body.get("selection"):
                return self.invalid({"selection": ["The selection field is required."]})
            prose = "REWRITTEN(" + body["selection"] + ")"
            persist = body.get("persist", False)
            if persist and body["selection"] in ch["content"]:
                ch["_before_rewrite"] = ch["content"]
                ch["content"] = ch["content"].replace(body["selection"], prose, 1)
        else:
            prose = "The lantern guttered. She stepped into the dark."
            persist = body.get("persist", True)
        if action.endswith("/stream"):
            words = prose.split(" ")
            deltas = [("delta", {"delta": w + (" " if i < len(words) - 1 else "")}) for i, w in enumerate(words)]
            if mode["stream"] == "truncate":
                return self.sse(deltas[:max(1, len(deltas) // 2)])
            if mode["stream"] == "error":
                return self.sse(deltas[:2] + [("error", {"message": "Upstream provider failed."})])
            self.apply(ch, action, prose, persist)
            return self.sse(deltas + [("done", {"prose": prose, "persisted": persist, "model": "fixture/model",
                                                "usage": {"prompt_tokens": 10, "completion_tokens": 9, "total_tokens": 19}})])
        self.apply(ch, action, prose, persist)
        return self.send(200, {"data": {"prose": prose, "persisted": persist, "model": "fixture/model"}})

    def apply(self, ch, action, prose, persist):
        if persist and action.startswith("continue"):
            ch["_before_continue"] = ch["content"]
            ch["content"] = (ch["content"] + "\n\n" + prose).strip()

    def do_PUT(self):
        self.read()
        if not self.authorized():
            return
        m = re.fullmatch(r"/api/series/(\d+)/stories/(\d+)", urlsplit(self.path).path)
        if m:
            s, b = state["series"].get(int(m.group(1))), state["books"].get(int(m.group(2)))
            if not s or not b:
                return self.send(404, {"message": "Not Found"})
            b["series_id"] = s["id"]
            if b["id"] not in s["story_ids"]:
                s["story_ids"].append(b["id"])
            return self.send(200, {"data": book_view(b)})
        return self.send(404, {"message": "Not Found"})

    def do_PATCH(self):
        body = self.read() or {}
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        m = re.fullmatch(r"/api/codex-entries/(\d+)", path)
        if m:
            e = state["codex"].get(int(m.group(1)))
            if not e:
                return self.send(404, {"message": "Not Found"})
            if "category" in body and body["category"] not in ("lore", "character"):
                return self.invalid({"category": ["The selected category is invalid."]})
            for k in ("category", "name", "aliases", "content", "order"):
                if k in body:
                    e[k] = body[k]
            return self.send(200, {"data": e})
        m = re.fullmatch(r"/api/series/(\d+)", path)
        if m:
            s = state["series"].get(int(m.group(1)))
            if not s:
                return self.send(404, {"message": "Not Found"})
            for k in ("name", "description"):
                if k in body:
                    s[k] = body[k]
            return self.send(200, {"data": s})
        m = re.fullmatch(r"/api/scenes/(\d+)", path)
        if m:
            c = state["chapters"].get(int(m.group(1)))
            if not c:
                return self.send(404, {"message": "Not Found"})
            c.update({k: v for k, v in body.items() if k in ("title", "content", "summary")})
            return self.send(200, {"data": c})
        return self.send(404, {"message": "Not Found"})

    def do_DELETE(self):
        self.read()
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        for pattern, key in ((r"/api/codex-entries/(\d+)", "codex"), (r"/api/series/(\d+)", "series"),
                             (r"/api/stories/(\d+)", "books"), (r"/api/conversations/(\d+)", "conversations")):
            m = re.fullmatch(pattern, path)
            if m:
                if state[key].pop(int(m.group(1)), None) is None:
                    return self.send(404, {"message": "Not Found"})
                if key == "series":
                    for b in state["books"].values():
                        if b.get("series_id") == int(m.group(1)):
                            b["series_id"] = None
                return self.send(204)
        m = re.fullmatch(r"/api/series/(\d+)/stories/(\d+)", path)
        if m:
            s, b = state["series"].get(int(m.group(1))), state["books"].get(int(m.group(2)))
            if not s or not b:
                return self.send(404, {"message": "Not Found"})
            b["series_id"] = None
            s["story_ids"] = [i for i in s["story_ids"] if i != b["id"]]
            return self.send(204)
        return self.send(404, {"message": "Not Found"})


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), Handler).serve_forever()
