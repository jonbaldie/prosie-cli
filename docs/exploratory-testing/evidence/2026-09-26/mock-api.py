import json
import re
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import urlsplit

state = {"books": {}, "chapters": {}, "conversations": {}}
requests = []

def payload(value):
    return json.dumps(value).encode()

class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_args):
        pass

    def send(self, status, value, content_type="application/json"):
        data = value if isinstance(value, bytes) else payload(value)
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def record(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        try:
            body = json.loads(raw) if raw else None
        except Exception:
            body = raw.decode(errors="replace")
        requests.append({"method": self.command, "path": self.path, "body": body,
                         "auth_valid": self.headers.get("Authorization") == "Bearer test-token"})
        return body

    def authorized(self):
        if self.headers.get("Authorization") != "Bearer test-token":
            self.send(401, {"message": "Invalid test token"})
            return False
        return True

    def do_GET(self):
        self.record()
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
            return self.send(200, {"data": list(state["books"].values())})
        match = re.fullmatch(r"/api/stories/(\d+)", path)
        if match:
            book = state["books"].get(int(match.group(1)))
            return self.send(200 if book else 404, {"data": book} if book else {"message": "Book not found"})
        match = re.fullmatch(r"/api/stories/(\d+)/scenes", path)
        if match:
            book_id = int(match.group(1))
            scenes = [v for v in state["chapters"].values() if v["story_id"] == book_id]
            return self.send(200, {"data": scenes})
        match = re.fullmatch(r"/api/scenes/(\d+)", path)
        if match:
            chapter = state["chapters"].get(int(match.group(1)))
            return self.send(200 if chapter else 404, {"data": chapter} if chapter else {"message": "Chapter not found"})
        match = re.fullmatch(r"/api/stories/(\d+)/export", path)
        if match:
            book = state["books"].get(int(match.group(1)))
            if not book:
                return self.send(404, {"message": "Book not found"})
            lines = ["# " + book["title"], ""]
            for chapter in book["chapters"]:
                lines.extend(["## " + chapter.get("title", chapter.get("name", "")), "", chapter.get("content", ""), ""])
            return self.send(200, "\n".join(lines).encode(), "text/markdown")
        match = re.fullmatch(r"/api/scenes/(\d+)/export", path)
        if match:
            chapter = state["chapters"].get(int(match.group(1)))
            if not chapter:
                return self.send(404, {"message": "Chapter not found"})
            return self.send(200, chapter.get("content", "").encode(), "text/markdown")
        match = re.fullmatch(r"/api/stories/(\d+)/conversations", path)
        if match:
            book_id = int(match.group(1))
            return self.send(200, {"data": [v for v in state["conversations"].values() if v["story_id"] == book_id]})
        match = re.fullmatch(r"/api/conversations/(\d+)/export", path)
        if match:
            conv = state["conversations"].get(int(match.group(1)))
            if not conv:
                return self.send(404, {"message": "Conversation not found"})
            return self.send(200, conv)
        return self.send(404, {"message": "Route not found"})

    def do_POST(self):
        body = self.record()
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        if path == "/api/stories":
            book_id = 100
            chapter = {"id": 101, "story_id": book_id, "order": 0, "name": "Chapter 1", "title": "Chapter 1", "content": "", "word_count": 0}
            book = {"id": book_id, "title": body["title"], "premise": body.get("premise"), "story_so_far": body.get("story_so_far"), "lore": body.get("lore"), "characters": body.get("characters"), "target_word_count": body.get("target_word_count"), "word_count": 0, "chapters": [chapter]}
            state["chapters"][chapter["id"]] = chapter
            state["books"][book_id] = book
            return self.send(201, {"data": book})
        match = re.fullmatch(r"/api/stories/(\d+)/scenes", path)
        if match:
            chapter_id = 102
            chapter = {"id": chapter_id, "story_id": int(match.group(1)), "order": 1, "name": body.get("name", "Chapter 2"), "title": body.get("title", body.get("name", "Chapter 2")), "content": body.get("content", ""), "summary": body.get("summary"), "word_count": len(body.get("content", "").split())}
            state["chapters"][chapter_id] = chapter
            book = state["books"][int(match.group(1))]
            book["chapters"].append(chapter)
            book["word_count"] = sum(ch.get("word_count", 0) for ch in book["chapters"])
            return self.send(201, {"data": chapter})
        match = re.fullmatch(r"/api/stories/(\d+)/conversations", path)
        if match:
            conv = {"id": 300, "story_id": int(match.group(1)), "title": body.get("title", "Test chat"), "fidelity": "balanced", "messages": []}
            state["conversations"][300] = conv
            return self.send(201, {"data": conv})
        match = re.fullmatch(r"/api/conversations/(\d+)/messages", path)
        if match:
            conv_id = int(match.group(1))
            message = {"id": 501 + len(state["conversations"][conv_id]["messages"]), "conversation_id": conv_id, "role": "assistant", "content": "The lantern is still lit."}
            state["conversations"][conv_id]["messages"].append({"role": "user", "content": body["content"]})
            state["conversations"][conv_id]["messages"].append(message)
            return self.send(200, {"data": {"message": message, "model": "fixture-model", "usage": {"prompt_tokens": 8, "completion_tokens": 6, "total_tokens": 14}}})
        match = re.fullmatch(r"/api/conversations/(\d+)/messages/stream", path)
        if match:
            conv_id = int(match.group(1))
            message = {"id": 599, "conversation_id": conv_id, "role": "assistant", "content": "The lantern is still lit."}
            state["conversations"][conv_id]["messages"].append({"role": "user", "content": body["content"]})
            state["conversations"][conv_id]["messages"].append(message)
            events = (
                'event:delta\ndata:{"delta":"The lantern "}\n\n'
                'event:delta\ndata:{"delta":"is still lit."}\n\n'
                'event:done\ndata:' + json.dumps({"message": message, "model": "fixture-model", "usage": {"prompt_tokens": 9, "completion_tokens": 6, "total_tokens": 15}}) + '\n\n'
            )
            return self.send(200, events.encode(), "text/event-stream")
        return self.send(404, {"message": "Route not found"})

    def do_PATCH(self):
        body = self.record()
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        match = re.fullmatch(r"/api/stories/(\d+)", path)
        if match:
            book = state["books"].get(int(match.group(1)))
            if not book:
                return self.send(404, {"message": "Book not found"})
            book.update(body)
            return self.send(200, {"data": book})
        match = re.fullmatch(r"/api/scenes/(\d+)", path)
        if match:
            chapter = state["chapters"].get(int(match.group(1)))
            if not chapter:
                return self.send(404, {"message": "Chapter not found"})
            chapter.update(body)
            return self.send(200, {"data": chapter})
        return self.send(404, {"message": "Route not found"})

    def do_DELETE(self):
        self.record()
        if not self.authorized():
            return
        path = urlsplit(self.path).path
        match = re.fullmatch(r"/api/stories/(\d+)", path)
        if match:
            book_id = int(match.group(1))
            state["books"].pop(book_id, None)
            state["chapters"] = {k: v for k, v in state["chapters"].items() if v["story_id"] != book_id}
            return self.send(204, b"", "application/json")
        match = re.fullmatch(r"/api/scenes/(\d+)", path)
        if match:
            state["chapters"].pop(int(match.group(1)), None)
            return self.send(204, b"", "application/json")
        return self.send(404, {"message": "Route not found"})

server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
print("http://127.0.0.1:%d" % server.server_address[1], flush=True)
server.serve_forever()
