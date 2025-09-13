import os
import json
import time
import re
from typing import List, Optional
from fastapi import FastAPI, HTTPException, Response
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import uvicorn
from threading import Thread


LLM_PORT = int(os.getenv('LLM_PORT', '8000'))
AUDIT_PORT = int(os.getenv('AUDIT_PORT', '8001'))


# Model for the LLM service request.
class LLMRequest(BaseModel):
    response_message: str  # The full message to be sent by the LLM.
    stream: bool = False
    event_num: int = 5  # Number of events to stream.

class LLMResponse(BaseModel):
    content: str
    tokens_used: int

class AuditRequest(BaseModel):
    content: str
    unhealthy_words: List[str] # List of words to check for, provided by the caller.
    custom_error_message: Optional[str] = None # Optional custom message if audit fails.

class AuditResponse(BaseModel):
    is_safe: bool
    flagged_words: List[str]
    error_message: Optional[str] = None # The custom error message, if provided.

llm_app = FastAPI(
    title="Mock LLM Service",
    description="A mock service that mimics LLM chat completions, with streaming.",
    version="3.0.0"
)
audit_app = FastAPI(
    title="Mock Audit Service",
    description="A mock service to audit content for a user-specified list of unhealthy words.",
    version="3.0.0"
)



@llm_app.get("/health")
async def llm_health():
    return {"status": "healthy", "service": "llm"}

@llm_app.post("/v1/chat/completions")
async def chat_completions(request: LLMRequest):
    if not request.stream:
        content = request.response_message
        return LLMResponse(content=content, tokens_used=len(content.split()))
    else:
        def generate_stream():

            message = request.response_message
            num_events = request.event_num

            if num_events <= 0:
                num_events = 1

            total_len = len(message)
            base_size = total_len // num_events
            remainder = total_len % num_events

            chunks_content = []
            current_pos = 0
            for i in range(num_events):
                chunk_len = base_size + (1 if i < remainder else 0)
                chunk = message[current_pos : current_pos + chunk_len]
                chunks_content.append(chunk)
                current_pos += chunk_len

            for i, chunk_text in enumerate(chunks_content):
                chunk_data = {
                    "id": f"chunk_{i}",
                    "object": "chat.completion.chunk",
                    "choices": [{
                        "delta": {"content": chunk_text},
                        "index": 0,
                        "finish_reason": None
                    }]
                }
                yield f"data: {json.dumps(chunk_data)}\n\n"
                time.sleep(0.1)

            final_chunk = {
                "id": "final",
                "object": "chat.completion.chunk",
                "choices": [{
                    "delta": {},
                    "index": 0,
                    "finish_reason": "stop"
                }]
            }
            yield f"data: {json.dumps(final_chunk)}\n\n"
            yield "data: [DONE]\n\n"

        return StreamingResponse(
            generate_stream(),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
                "Access-Control-Allow-Origin": "*",
            }
        )


@audit_app.get("/health")
async def audit_health():
    return {"status": "healthy", "service": "audit"}

@audit_app.post("/audit", response_model=AuditResponse)
async def audit_content(request: AuditRequest):
    flagged_words = []

    for word in request.unhealthy_words:
        pattern = r'\b' + re.escape(word) + r'\b'

        if re.search(pattern, request.content, re.IGNORECASE):
            flagged_words.append(word)

    is_safe = len(flagged_words) == 0
    error_message = None

    if not is_safe and request.custom_error_message:
        error_message = request.custom_error_message

    unique_flagged_words = sorted(list(set(flagged_words)))

    return AuditResponse(
        is_safe=is_safe,
        flagged_words=unique_flagged_words,
        error_message=error_message
    )


def run_llm_service():
    uvicorn.run(llm_app, host="0.0.0.0", port=LLM_PORT, log_level="info")

def run_audit_service():
    uvicorn.run(audit_app, host="0.0.0.0", port=AUDIT_PORT, log_level="info")


if __name__ == "__main__":
    llm_thread = Thread(target=run_llm_service, daemon=True)
    audit_thread = Thread(target=run_audit_service, daemon=True)

    llm_thread.start()
    audit_thread.start()

    print(f"Mock LLM Service running on http://0.0.0.0:{LLM_PORT}")
    print(f"Mock Audit Service running on http://0.0.0.0:{AUDIT_PORT}")
    print("LLM streaming now splits by character into a fixed number of events via 'event_num'.")
    print("Services are running. Press Ctrl+C to shut down.")

    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\nShutting down services...")