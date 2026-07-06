#!/usr/bin/env python3
import json
import os
import shutil
import signal
import tempfile
import time
import errno
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import unquote, urlparse


CONFIG_PATH = os.environ.get("CONFIG_PATH", "/ServerStatus/server/config.json")
STATS_PATH = os.environ.get("STATS_PATH", "/usr/share/nginx/html/json/stats.json")
SERGATE_PID_FILE = os.environ.get("SERGATE_PID_FILE", "/tmp/serverstatus-sergate.pid")
ADMIN_TOKEN = os.environ.get("ADMIN_TOKEN", "")
API_BIND = os.environ.get("ADMIN_API_BIND", "127.0.0.1")
API_PORT = int(os.environ.get("ADMIN_API_PORT", "35602"))
CORS_ORIGIN = os.environ.get("ADMIN_CORS_ORIGIN", "")


class ApiError(Exception):
    def __init__(self, status, message, details=None):
        super().__init__(message)
        self.status = status
        self.message = message
        self.details = details


def load_config():
    with open(CONFIG_PATH, "r", encoding="utf-8") as f:
        return json.load(f)


def load_json_file(path):
    with open(path, "r", encoding="utf-8") as f:
        return json.load(f)


def load_stats_file():
    try:
        return load_json_file(STATS_PATH)
    except FileNotFoundError:
        return load_json_file(f"{STATS_PATH}~")


def write_json_file(path, data):
    directory = os.path.dirname(path) or "."
    os.makedirs(directory, exist_ok=True)
    mode = os.stat(path).st_mode if os.path.exists(path) else 0o644
    payload = json.dumps(data, ensure_ascii=False, indent="\t") + "\n"
    fd, tmp_path = tempfile.mkstemp(prefix=".json.", suffix=".tmp", dir=directory)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(payload)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(tmp_path, mode)
        os.replace(tmp_path, path)
    finally:
        if os.path.exists(tmp_path):
            os.unlink(tmp_path)


def normalize_required_strings(item, required, kind, index=None):
    missing = [field for field in required if item.get(field) is None or not str(item.get(field, "")).strip()]
    if missing:
        raise ApiError(400, f"{kind} has missing required fields", {"missing": missing, "index": index})
    normalized = dict(item)
    for field in required:
        normalized[field] = str(normalized[field]).strip()
    return normalized


def normalize_optional_string(item, field):
    value = item.get(field, "")
    item[field] = "" if value is None else str(value).strip()


def normalize_int(value, field, index=None, default=None, min_value=None, max_value=None):
    if value in (None, ""):
        value = default
    try:
        value = int(value)
    except (TypeError, ValueError):
        raise ApiError(400, f"{field} must be an integer", {"index": index})
    if min_value is not None:
        value = max(min_value, value)
    if max_value is not None:
        value = min(max_value, value)
    return value


def validate_server(server, index=None):
    if not isinstance(server, dict):
        raise ApiError(400, "server must be an object", {"index": index})
    required = ["username", "name", "type", "host", "location", "password"]
    normalized = normalize_required_strings(server, required, "server", index=index)
    normalized["monthstart"] = normalize_int(normalized.get("monthstart"), "monthstart", index=index, default=1, min_value=1, max_value=28)
    if "disabled" in normalized:
        normalized["disabled"] = bool(normalized["disabled"])
    return normalized


def validate_monitor(monitor, index=None):
    if not isinstance(monitor, dict):
        raise ApiError(400, "monitor must be an object", {"index": index})
    normalized = normalize_required_strings(monitor, ["name", "host", "type"], "monitor", index=index)
    normalized["interval"] = normalize_int(normalized.get("interval"), "interval", index=index, default=600, min_value=1)
    return normalized


def validate_sslcert(sslcert, index=None):
    if not isinstance(sslcert, dict):
        raise ApiError(400, "sslcert must be an object", {"index": index})
    normalized = normalize_required_strings(sslcert, ["name", "domain"], "sslcert", index=index)
    normalized["port"] = normalize_int(normalized.get("port"), "port", index=index, default=443, min_value=1, max_value=65535)
    normalized["interval"] = normalize_int(normalized.get("interval"), "interval", index=index, default=7200, min_value=1)
    normalize_optional_string(normalized, "callback")
    return normalized


def validate_watchdog(watchdog, index=None):
    if not isinstance(watchdog, dict):
        raise ApiError(400, "watchdog must be an object", {"index": index})
    normalized = normalize_required_strings(watchdog, ["name", "rule"], "watchdog", index=index)
    normalized["interval"] = normalize_int(normalized.get("interval"), "interval", index=index, default=600, min_value=1)
    normalize_optional_string(normalized, "callback")
    return normalized


COLLECTIONS = {
    "servers": {
        "item": "server",
        "id_field": "username",
        "validator": validate_server,
        "required": ["username", "name", "type", "host", "location", "password"],
        "optional": ["monthstart", "disabled"],
    },
    "monitors": {
        "item": "monitor",
        "id_field": "name",
        "validator": validate_monitor,
        "required": ["name", "host", "type"],
        "optional": ["interval"],
    },
    "sslcerts": {
        "item": "sslcert",
        "id_field": "name",
        "validator": validate_sslcert,
        "required": ["name", "domain"],
        "optional": ["port", "interval", "callback"],
    },
    "watchdog": {
        "item": "watchdog",
        "id_field": "name",
        "validator": validate_watchdog,
        "required": ["name", "rule"],
        "optional": ["interval", "callback"],
    },
}


def validate_config(config):
    if not isinstance(config, dict):
        raise ApiError(400, "config must be a JSON object")
    config = dict(config)
    for key, meta in COLLECTIONS.items():
        items = config.get(key, [])
        if not isinstance(items, list):
            raise ApiError(400, f"{key} must be an array")
        normalized = []
        seen = set()
        for index, item in enumerate(items):
            entry = meta["validator"](item, index=index)
            if key == "servers":
                username = entry["username"]
                if username in seen:
                    raise ApiError(409, "duplicate server username", {"username": username})
                seen.add(username)
            normalized.append(entry)
        config[key] = normalized
    return config


def write_config(config):
    config = validate_config(config)
    directory = os.path.dirname(CONFIG_PATH) or "."
    os.makedirs(directory, exist_ok=True)
    timestamp = time.strftime("%Y%m%d-%H%M%S")
    if os.path.exists(CONFIG_PATH):
        shutil.copy2(CONFIG_PATH, f"{CONFIG_PATH}.bak-{timestamp}")
        mode = os.stat(CONFIG_PATH).st_mode
    else:
        mode = 0o644
    data = json.dumps(config, ensure_ascii=False, indent="\t") + "\n"
    fd, tmp_path = tempfile.mkstemp(prefix=".config.", suffix=".tmp", dir=directory)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(data)
            f.flush()
            os.fsync(f.fileno())
        os.chmod(tmp_path, mode)
        try:
            os.replace(tmp_path, CONFIG_PATH)
        except OSError as exc:
            if exc.errno != errno.EBUSY:
                raise
            # A single-file Docker bind mount cannot be atomically replaced.
            # Keep the backup above, then rewrite the mounted file in place.
            with open(CONFIG_PATH, "w", encoding="utf-8") as f:
                f.write(data)
                f.flush()
                os.fsync(f.fileno())
    finally:
        if os.path.exists(tmp_path):
            os.unlink(tmp_path)
    return config


def write_and_reload(config):
    config = write_config(config)
    pid = signal_sergate(signal.SIGHUP)
    return config, pid


def read_body(handler):
    length = int(handler.headers.get("Content-Length", "0") or "0")
    if length <= 0:
        return None
    if length > 1024 * 1024:
        raise ApiError(413, "request body is too large")
    raw = handler.rfile.read(length).decode("utf-8")
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ApiError(400, "invalid JSON body", {"error": str(exc)})


def get_sergate_pid():
    try:
        with open(SERGATE_PID_FILE, "r", encoding="utf-8") as f:
            pid = int(f.read().strip())
        os.kill(pid, 0)
        return pid
    except Exception:
        pass
    proc_dir = "/proc"
    if not os.path.isdir(proc_dir):
        return None
    for name in os.listdir(proc_dir):
        if not name.isdigit():
            continue
        try:
            with open(os.path.join(proc_dir, name, "cmdline"), "rb") as f:
                cmdline = f.read().replace(b"\x00", b" ").decode("utf-8", "ignore")
        except Exception:
            continue
        if "sergate" in cmdline:
            return int(name)
    return None


def signal_sergate(sig):
    pid = get_sergate_pid()
    if not pid:
        raise ApiError(503, "sergate process was not found")
    os.kill(pid, sig)
    return pid


def wait_for_pid_exit(pid, timeout=2.5):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            os.kill(pid, 0)
        except OSError:
            return True
        time.sleep(0.05)
    return False


def wait_for_sergate_pid(previous_pid=None, timeout=4.0):
    deadline = time.time() + timeout
    while time.time() < deadline:
        pid = get_sergate_pid()
        if pid and pid != previous_pid:
            return pid
        time.sleep(0.1)
    return get_sergate_pid()


def find_server(config, username):
    servers = config.get("servers", [])
    for index, server in enumerate(servers):
        if server.get("username") == username:
            return index, server
    return -1, None


def find_collection_item(config, key, item_id):
    items = config.get(key, [])
    if item_id.isdigit():
        index = int(item_id)
        if 0 <= index < len(items):
            return index, items[index]
        return -1, None
    id_field = COLLECTIONS[key]["id_field"]
    matches = [(index, item) for index, item in enumerate(items) if str(item.get(id_field, "")) == item_id]
    if len(matches) > 1:
        raise ApiError(409, f"{key} has duplicate {id_field}; use numeric index instead", {"id": item_id})
    if matches:
        return matches[0]
    return -1, None


def stats_server_matches(config_server, stats_server):
    return all(str(stats_server.get(field, "")) == str(config_server.get(field, "")) for field in ["name", "type", "host", "location"])


def find_stats_server(stats, config_server):
    servers = stats.get("servers", [])
    if not isinstance(servers, list):
        raise ApiError(500, "stats.json has invalid servers data")
    for index, stats_server in enumerate(servers):
        if isinstance(stats_server, dict) and stats_server_matches(config_server, stats_server):
            return index, stats_server
    return -1, None


def as_counter(value, field):
    try:
        return int(value)
    except (TypeError, ValueError):
        raise ApiError(409, f"{field} is missing or invalid in stats.json")


def require_resettable_stats(stats, server, username):
    stats_index, stats_server = find_stats_server(stats, server)
    if stats_index < 0:
        raise ApiError(404, "server stats were not found", {"username": username})
    if "network_in" not in stats_server or "network_out" not in stats_server:
        raise ApiError(409, "server has no current traffic counters; it may be offline", {"username": username})
    return stats_index, stats_server


def reset_server_month_traffic(username):
    config = load_config()
    _, server = find_server(config, username)
    if server is None:
        raise ApiError(404, "server was not found", {"username": username})

    require_resettable_stats(load_stats_file(), server, username)

    old_pid = signal_sergate(signal.SIGTERM)
    wait_for_pid_exit(old_pid)

    stats = load_stats_file()
    stats_index, stats_server = require_resettable_stats(stats, server, username)

    network_in = as_counter(stats_server.get("network_in"), "network_in")
    network_out = as_counter(stats_server.get("network_out"), "network_out")
    previous_last_in = as_counter(stats_server.get("last_network_in", 0), "last_network_in")
    previous_last_out = as_counter(stats_server.get("last_network_out", 0), "last_network_out")

    stats["servers"][stats_index]["last_network_in"] = network_in
    stats["servers"][stats_index]["last_network_out"] = network_out
    stats["updated"] = str(int(time.time()))
    write_json_file(STATS_PATH, stats)

    new_pid = wait_for_sergate_pid(previous_pid=old_pid)
    if new_pid:
        os.kill(new_pid, signal.SIGHUP)

    return {
        "server": server,
        "stats": {
            "network_in": network_in,
            "network_out": network_out,
            "previous_last_network_in": previous_last_in,
            "previous_last_network_out": previous_last_out,
            "last_network_in": network_in,
            "last_network_out": network_out,
            "month_in_before": max(0, network_in - previous_last_in),
            "month_out_before": max(0, network_out - previous_last_out),
        },
        "oldPid": old_pid,
        "pid": new_pid,
    }


def collection_routes():
    endpoints = []
    for key in ["monitors", "sslcerts", "watchdog"]:
        endpoints.extend([
            {"method": "GET", "path": f"/api/{key}", "auth": True},
            {"method": "POST", "path": f"/api/{key}", "auth": True, "body": f"{COLLECTIONS[key]['item']} JSON"},
            {"method": "PUT", "path": f"/api/{key}/{{index-or-name}}", "auth": True, "body": f"{COLLECTIONS[key]['item']} JSON"},
            {"method": "DELETE", "path": f"/api/{key}/{{index-or-name}}", "auth": True},
        ])
    return endpoints


def api_schema():
    return {
        "auth": {
            "type": "bearer",
            "header": "Authorization: Bearer <ADMIN_TOKEN>",
            "enabled": bool(ADMIN_TOKEN),
        },
        "endpoints": [
            {"method": "GET", "path": "/api/health", "auth": False},
            {"method": "GET", "path": "/api/schema", "auth": False},
            {"method": "GET", "path": "/api/config", "auth": True},
            {"method": "PUT", "path": "/api/config", "auth": True, "body": "full config JSON"},
            {"method": "GET", "path": "/api/servers", "auth": True},
            {"method": "POST", "path": "/api/servers", "auth": True, "body": "server JSON"},
            {"method": "PUT", "path": "/api/servers/{username}", "auth": True, "body": "server JSON"},
            {"method": "DELETE", "path": "/api/servers/{username}", "auth": True},
            {"method": "POST", "path": "/api/servers/{username}/reset-traffic", "auth": True},
            *collection_routes(),
            {"method": "POST", "path": "/api/reload", "auth": True},
            {"method": "POST", "path": "/api/restart", "auth": True},
        ],
        "collections": {
            key: {
                "item": meta["item"],
                "idField": meta["id_field"],
                "required": meta["required"],
                "optional": meta["optional"],
            }
            for key, meta in COLLECTIONS.items()
        },
    }


class Handler(BaseHTTPRequestHandler):
    server_version = "ServerStatusManageAPI/1.0"

    def log_message(self, fmt, *args):
        print("%s - %s" % (self.address_string(), fmt % args), flush=True)

    def end_headers(self):
        self.send_header("Cache-Control", "no-store")
        if CORS_ORIGIN:
            self.send_header("Access-Control-Allow-Origin", CORS_ORIGIN)
            self.send_header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token")
            self.send_header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        super().end_headers()

    def send_json(self, status, payload):
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def send_error_json(self, err):
        payload = {"ok": False, "error": err.message}
        if err.details is not None:
            payload["details"] = err.details
        self.send_json(err.status, payload)

    def route(self):
        parsed = urlparse(self.path)
        path = parsed.path.rstrip("/") or "/"
        method = self.command.upper()
        if method == "OPTIONS":
            self.send_response(204)
            self.end_headers()
            return
        try:
            if path == "/api/health" and method == "GET":
                pid = get_sergate_pid()
                self.send_json(200, {
                    "ok": True,
                    "enabled": bool(ADMIN_TOKEN),
                    "sergate": {"running": bool(pid), "pid": pid},
                    "configPath": CONFIG_PATH,
                })
                return
            if path == "/api/schema" and method == "GET":
                self.send_json(200, {"ok": True, "schema": api_schema()})
                return
            self.require_auth()
            if path == "/api/config":
                if method == "GET":
                    self.send_json(200, {"ok": True, "config": load_config()})
                    return
                if method == "PUT":
                    config = read_body(self)
                    config, pid = write_and_reload(config)
                    self.send_json(200, {"ok": True, "reloaded": True, "pid": pid, "config": config})
                    return
            if path == "/api/servers":
                if method == "GET":
                    config = load_config()
                    self.send_json(200, {"ok": True, "servers": config.get("servers", [])})
                    return
                if method == "POST":
                    server = validate_server(read_body(self))
                    config = load_config()
                    if find_server(config, server["username"])[1] is not None:
                        raise ApiError(409, "server username already exists", {"username": server["username"]})
                    config.setdefault("servers", []).append(server)
                    config, pid = write_and_reload(config)
                    self.send_json(201, {"ok": True, "server": server, "reloaded": True, "pid": pid, "config": config})
                    return
            if path.startswith("/api/servers/") and path.endswith("/reset-traffic"):
                username = unquote(path[len("/api/servers/"):-len("/reset-traffic")].rstrip("/"))
                if not username:
                    raise ApiError(400, "username is required")
                if method == "POST":
                    result = reset_server_month_traffic(username)
                    self.send_json(200, {"ok": True, "operation": "reset-traffic", **result})
                    return
            if path.startswith("/api/servers/"):
                username = unquote(path[len("/api/servers/"):])
                if not username:
                    raise ApiError(400, "username is required")
                config = load_config()
                index, existing = find_server(config, username)
                if index < 0:
                    raise ApiError(404, "server was not found", {"username": username})
                if method == "PUT":
                    server = validate_server(read_body(self))
                    if server["username"] != username and find_server(config, server["username"])[1] is not None:
                        raise ApiError(409, "server username already exists", {"username": server["username"]})
                    config["servers"][index] = server
                    config, pid = write_and_reload(config)
                    self.send_json(200, {"ok": True, "server": server, "reloaded": True, "pid": pid, "config": config})
                    return
                if method == "DELETE":
                    removed = config["servers"].pop(index)
                    config, pid = write_and_reload(config)
                    self.send_json(200, {"ok": True, "removed": removed, "reloaded": True, "pid": pid, "config": config})
                    return
            for key in ["monitors", "sslcerts", "watchdog"]:
                base = f"/api/{key}"
                meta = COLLECTIONS[key]
                if path == base:
                    if method == "GET":
                        config = load_config()
                        self.send_json(200, {"ok": True, key: config.get(key, [])})
                        return
                    if method == "POST":
                        item = meta["validator"](read_body(self))
                        config = load_config()
                        config.setdefault(key, []).append(item)
                        config, pid = write_and_reload(config)
                        self.send_json(201, {"ok": True, meta["item"]: item, "reloaded": True, "pid": pid, "config": config})
                        return
                if path.startswith(base + "/"):
                    item_id = unquote(path[len(base) + 1:])
                    if not item_id:
                        raise ApiError(400, "item id is required")
                    config = load_config()
                    index, existing = find_collection_item(config, key, item_id)
                    if index < 0:
                        raise ApiError(404, f"{meta['item']} was not found", {"id": item_id})
                    if method == "PUT":
                        item = meta["validator"](read_body(self))
                        config[key][index] = item
                        config, pid = write_and_reload(config)
                        self.send_json(200, {"ok": True, meta["item"]: item, "reloaded": True, "pid": pid, "config": config})
                        return
                    if method == "DELETE":
                        removed = config[key].pop(index)
                        config, pid = write_and_reload(config)
                        self.send_json(200, {"ok": True, "removed": removed, "reloaded": True, "pid": pid, "config": config})
                        return
            if path == "/api/reload" and method == "POST":
                pid = signal_sergate(signal.SIGHUP)
                self.send_json(200, {"ok": True, "operation": "reload", "pid": pid})
                return
            if path == "/api/restart" and method == "POST":
                pid = signal_sergate(signal.SIGTERM)
                self.send_json(202, {"ok": True, "operation": "restart", "pid": pid})
                return
            raise ApiError(404, "endpoint was not found")
        except ApiError as err:
            self.send_error_json(err)
        except Exception as exc:
            self.send_error_json(ApiError(500, "internal server error", {"error": str(exc)}))

    def require_auth(self):
        if not ADMIN_TOKEN:
            raise ApiError(503, "management API is disabled; set ADMIN_TOKEN to enable it")
        auth = self.headers.get("Authorization", "")
        token = ""
        if auth.lower().startswith("bearer "):
            token = auth[7:].strip()
        token = token or self.headers.get("X-Admin-Token", "").strip()
        if token != ADMIN_TOKEN:
            raise ApiError(401, "invalid or missing admin token")

    def do_GET(self):
        self.route()

    def do_POST(self):
        self.route()

    def do_PUT(self):
        self.route()

    def do_DELETE(self):
        self.route()

    def do_OPTIONS(self):
        self.route()


def main():
    httpd = ThreadingHTTPServer((API_BIND, API_PORT), Handler)
    print(f"manage-api listening on {API_BIND}:{API_PORT}", flush=True)
    httpd.serve_forever()


if __name__ == "__main__":
    main()
