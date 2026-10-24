"""Run public CLI journeys against an isolated in-memory HTTP fixture.

Usage: python3 docs/exploratory-testing/evidence/2026-10-24/run.py build/exploratory-2026-10-24/prosie
Uses only the Python standard library. Scratch files use TMPDIR.
The fixture is not a live Prosie service or a backend conformance test.
"""
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shlex
import stat
import subprocess
import sys
import tempfile
import threading
import zipfile

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("fixture", HERE.parent / "2026-10-17" / "mock-api.py")
fixture = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fixture)


def document():
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as archive:
        for name, content in {
            "[Content_Types].xml": '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>',
            "_rels/.rels": '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>',
            "word/document.xml": '<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Paragraph one.</w:t></w:r></w:p><w:p><w:r><w:t>Paragraph two.</w:t></w:r></w:p></w:body></w:document>',
        }.items():
            info = zipfile.ZipInfo(name, (2026, 10, 24, 0, 0, 0))
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, content)
    return output.getvalue()


DOCX = document()


class Handler(fixture.Handler):
    def export_book(self, book, fmt):
        if fmt == "docx":
            self.send_bytes(200, DOCX, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
        else:
            super().export_book(book, fmt)

    def scene(self, method, path, query, body):
        if method == "GET" and path.endswith("/export") and query.get("format") == ["docx"]:
            self.send_bytes(200, DOCX, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
        else:
            super().scene(method, path, query, body)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def main():
    binary = str(Path(sys.argv[1]).resolve())
    server = fixture.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        with tempfile.TemporaryDirectory(prefix="prosie-explore-", dir=os.environ["TMPDIR"]) as scratch:
            scratch = Path(scratch)
            env = {key: value for key, value in os.environ.items() if not key.startswith("PROSIE_")}
            env.update(PROSIE_API_URL=f"http://127.0.0.1:{server.server_port}", PROSIE_CONFIG_PATH=str(scratch / "config.json"))
            with (HERE / "transcript.txt").open("w") as log:
                def note(text):
                    log.write(text + "\n")
                    log.flush()

                def run(*args, stdin=None, expected=0):
                    result = subprocess.run([binary, *args], input=stdin, capture_output=True, env=env, cwd=scratch, timeout=15)
                    note("$ prosie " + shlex.join(args))
                    note("exit=" + str(result.returncode))
                    note("stdout=" + repr(result.stdout.decode("utf-8", errors="backslashreplace")))
                    note("stderr=" + repr(result.stderr.decode("utf-8", errors="backslashreplace")))
                    assert result.returncode == expected, (args, result.returncode, result.stderr)
                    return result.stdout

                note("Source: " + subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip())
                note("Go: " + subprocess.check_output(["go", "version"], text=True).strip())
                note("All tokens and identities in this evidence are synthetic.")
                note("JOURNEY 1: sign in, reopen, and sign out")
                run("auth", "status", "--json", expected=1)
                run("auth", "login", "--token", "fixture-token", "--json")
                mode = stat.S_IMODE((scratch / "config.json").stat().st_mode)
                note(f"new credential file mode={mode:o}")
                assert mode == 0o600
                run("auth", "status", "--json")
                (scratch / "config.json").chmod(0o644)
                run("auth", "login", "--token", "replacement-fixture-token", "--json")
                mode = stat.S_IMODE((scratch / "config.json").stat().st_mode)
                note(f"replaced credential file mode={mode:o}")
                assert mode == 0o600
                run("auth", "logout", "--json")
                assert not (scratch / "config.json").exists()
                run("auth", "status", "--json", expected=1)
                run("auth", "login", "--no-browser", "--json")
                run("auth", "status", "--json")

                note("JOURNEY 2: save prose and export it for backup")
                book = json.loads(run("book", "create", "--title", "The Winter Door", "--target-words", "1000", "--json"))
                book_id = str(book["id"])
                chapter_id = str(book["chapters"][0]["id"])
                prose = "  Paragraph one.\n\nParagraph two.\n"
                run("chapter", "update", chapter_id, "--file", "-", "--json", stdin=prose.encode())
                saved = json.loads(run("chapter", "show", chapter_id, "--json"))
                assert saved["content"] == prose
                run("book", "create", "--title", "Bad target", "--target-words", "0", "--json", expected=1)
                for resource, rid in (("book", book_id), ("chapter", chapter_id)):
                    md = run(resource, "export", rid)
                    exported = json.loads(run(resource, "export", rid, "--json"))
                    assert exported["content"].encode() == md
                    run(resource, "export", rid, "--format", "docx", "-o", resource + ".docx", "--json")
                    file_data = (scratch / (resource + ".docx")).read_bytes()
                    assert file_data == DOCX
                    raw = run(resource, "export", rid, "--format", "docx")
                    assert raw == file_data
                    assert zipfile.ZipFile(io.BytesIO(file_data)).testzip() is None
                    note(f"{resource} DOCX file: bytes={len(file_data)} sha256={sha(file_data)} zip=valid")
                    run(resource, "export", rid, "-o", "missing/backup.md", "--json", expected=1)
                    for replay in (1, 2):
                        data = json.loads(run(resource, "export", rid, "--format", "docx", "--json"))
                        reconstructed = data["content"].encode()
                        try:
                            valid = zipfile.ZipFile(io.BytesIO(reconstructed)).testzip() is None
                        except (zipfile.BadZipFile, ValueError) as error:
                            valid = False
                            note(f"{resource} replay {replay}: archive error={error}")
                        note(f"{resource} replay {replay}: reported_bytes={data['bytes']} reconstructed_bytes={len(reconstructed)} sha256={sha(reconstructed)} equal_to_download={reconstructed == DOCX} zip_valid={valid} replacement_characters={data['content'].count(chr(0xfffd))}")

                note("JOURNEY 3: ask a question, keep the thread, and recover after a failed send")
                turn = json.loads(run("chat", "send", book_id, "When does the door open?", "--title", "Door notes", "--json"))
                cid = str(turn["conversation_id"])
                run("chat", "send", "--conversation", cid, "Keep the first word.", "--json")
                thread_data = json.loads(run("chat", "show", cid, "--json"))
                assert thread_data["messages"][2]["content"] == "Keep the first word."
                run("chat", "export", cid, "-o", "thread.json", "--json")
                archived = json.loads((scratch / "thread.json").read_text())
                assert archived["messages"] == thread_data["messages"]
                imported = json.loads(run("chat", "import", book_id, "thread.json", "--json"))
                restored = json.loads(run("chat", "show", str(imported["id"]), "--json"))
                assert [m["content"] for m in restored["messages"]] == [m["content"] for m in archived["messages"]]
                (scratch / "bad.json").write_text("{broken")
                before = len(fixture.state["conversations"])
                run("chat", "import", book_id, "bad.json", "--json", expected=1)
                assert len(fixture.state["conversations"]) == before
                fixture.mode["fail_message"] = True
                run("chat", "send", "--conversation", cid, "Retry this question.", "--json", expected=1)
                fixture.mode["fail_message"] = False
                run("chat", "send", "--conversation", cid, "Retry this question.", "--json")
                run("chat", "stream", "--conversation", cid, "One more question.", "--json")
                run("chat", "show", cid, "--json")
                run("auth", "logout", "--json")
                note("Scratch data removed on driver exit. HTTP server stopped in finally. No Docker resources used.")
            # Do not persist synthetic credential headers; retain request bodies for replay evidence.
            captures = [{k: v for k, v in entry.items() if k != "authorization"} for entry in fixture.requests]
            (HERE / "requests.json").write_text(json.dumps(captures, indent=2) + "\n")
            (HERE / "docx-source.json").write_text(json.dumps({"bytes": len(DOCX), "sha256": sha(DOCX), "members": zipfile.ZipFile(io.BytesIO(DOCX)).namelist()}, indent=2) + "\n")
    finally:
        server.shutdown()
        server.server_close()
        thread.join()


if __name__ == "__main__":
    main()
