"""Prueba pagada mínima por MCP real. La credencial se hereda, nunca se imprime."""

import json
import queue
import subprocess
import sys
import threading


def main():
    if len(sys.argv) < 2:
        raise SystemExit("Uso: python3 scripts/smoke.py <comando del servidor> [args...]")
    process = subprocess.Popen(
        sys.argv[1:], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True
    )
    messages = queue.Queue()

    def read_messages():
        for line in process.stdout:
            try:
                messages.put(json.loads(line))
            except ValueError:
                messages.put({"error": "El servidor escribió texto ajeno a JSON-RPC"})
        messages.put({"error": "El servidor cerró stdout"})

    threading.Thread(target=read_messages, daemon=True).start()

    def send(message):
        process.stdin.write(json.dumps({"jsonrpc": "2.0", **message}) + "\n")
        process.stdin.flush()

    def request(request_id, method, params):
        send({"id": request_id, "method": method, "params": params})
        while True:
            message = messages.get(timeout=45)
            if "error" in message:
                raise RuntimeError("Error JSON-RPC durante " + method)
            if message.get("id") == request_id:
                return message["result"]

    try:
        request(1, "initialize", {
            "protocolVersion": "2025-03-26", "capabilities": {},
            "clientInfo": {"name": "jev-mcp-smoke", "version": "1"},
        })
        send({"method": "notifications/initialized"})
        tools = request(2, "tools/list", {})["tools"]
        if [tool["name"] for tool in tools] != ["decide"]:
            raise RuntimeError("El servidor no expone la tool decide esperada")
        result = request(3, "tools/call", {"name": "decide", "arguments": {
            "state": "I was charged twice for my subscription. Please refund the duplicate charge.",
            "questions": {
                "department": {"type": "choice", "instructions": "Which team should handle this?", "criteria": {
                    "billing": "Payments, charges and refunds", "technical": "Software defects", "other": "None of the other teams fits",
                }},
                "refund": {"type": "noul", "instructions": "Does the customer explicitly request a refund?"},
                "explicitness": {"type": "score", "instructions": "How explicit is the refund request?", "criteria": [
                    "No refund requested", "Refund suggested indirectly", "Refund requested explicitly",
                ]},
            },
        }})
        if result.get("isError"):
            # Los errores del servidor ya están saneados, sin body ni headers del proveedor.
            raise RuntimeError("decide falló: " + json.dumps(result.get("content")))
        output = result["structuredContent"]
        answers = output["answers"]
        if answers["department"]["choice"] != "billing" or answers["refund"]["noul"] < 0.5:
            raise RuntimeError("La llamada completó pero no cumplió la expectativa semántica mínima")
        print(json.dumps({"status": "passed", **output}, indent=2))
    finally:
        process.stdin.close()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()


if __name__ == "__main__":
    main()
