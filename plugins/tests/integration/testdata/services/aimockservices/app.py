import os
import json
import time
from typing import List, Optional, Dict
from fastapi import FastAPI
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import uvicorn

# -------------------------------
# 环境变量
# -------------------------------
LLM_PORT = int(os.getenv("LLM_PORT", "8000"))

# -------------------------------
# 数据模型
# -------------------------------
class ChatMessage(BaseModel):
    role: str
    content: str

class ChatCompletionRequest(BaseModel):
    model: str
    messages: List[ChatMessage]
    stream: Optional[bool] = False

class ChatCompletionChoice(BaseModel):
    index: int
    message: Dict[str, str]
    finish_reason: Optional[str] = "stop"

class ChatCompletionResponse(BaseModel):
    id: str
    object: str
    created: int
    model: str
    choices: List[ChatCompletionChoice]
    usage: Dict[str, int]

# -------------------------------
# FastAPI 应用
# -------------------------------
app = FastAPI(
    title="Mock OpenAI-Compatible LLM Service",
    description="Simulates the OpenAI Chat Completions API (streaming + non-streaming).",
    version="1.0.0"
)

# -------------------------------
# 健康检查
# -------------------------------
@app.get("/health")
async def health_llm():
    return {"status": "healthy", "service": "llm"}

# -------------------------------
# 模拟 /v1/chat/completions
# -------------------------------
@app.post("/v1/chat/completions")
async def chat_completions(request: ChatCompletionRequest):
    """
    模拟标准的 OpenAI /v1/chat/completions 接口。
    """
    # 获取用户输入内容
    user_messages = [m.content for m in request.messages if m.role == "user"]
    user_content = " ".join(user_messages).strip()
    if not user_content:
        user_content = "(empty message)"

    # 模拟回复
    reply = f"这是模拟回复: {user_content}"

    # 非流式响应
    if not request.stream:
        response = ChatCompletionResponse(
            id="mock-chatcmpl-123",
            object="chat.completion",
            created=int(time.time()),
            model=request.model,
            choices=[
                ChatCompletionChoice(
                    index=0,
                    message={"role": "assistant", "content": reply},
                    finish_reason="stop"
                )
            ],
            usage={
                "prompt_tokens": len(user_content.split()),
                "completion_tokens": len(reply.split()),
                "total_tokens": len(user_content.split()) + len(reply.split())
            }
        )
        return response

    # 流式响应（SSE）
    def generate_stream():
        chunks = [reply[i:i+10] for i in range(0, len(reply), 10)]
        for i, chunk in enumerate(chunks):
            chunk_data = {
                "id": f"mock-stream-{i}",
                "object": "chat.completion.chunk",
                "created": int(time.time()),
                "model": request.model,
                "choices": [{
                    "index": 0,
                    "delta": {"content": chunk},
                    "finish_reason": None
                }]
            }
            yield f"data: {json.dumps(chunk_data, ensure_ascii=False)}\n\n"
            time.sleep(0.1)

        # 最后一个 stop chunk
        final_data = {
            "id": "mock-final",
            "object": "chat.completion.chunk",
            "created": int(time.time()),
            "model": request.model,
            "choices": [{
                "index": 0,
                "delta": {},
                "finish_reason": "stop"
            }]
        }
        yield f"data: {json.dumps(final_data, ensure_ascii=False)}\n\n"
        yield "data: [DONE]\n\n"

    return StreamingResponse(
        generate_stream(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "Access-Control-Allow-Origin": "*"
        }
    )

# -------------------------------
# 启动
# -------------------------------
if __name__ == "__main__":
    print(f"✅ Mock OpenAI LLM running at http://0.0.0.0:{LLM_PORT}/v1/chat/completions")
    print("Supports both streaming and non-streaming responses.")
    uvicorn.run(app, host="0.0.0.0", port=LLM_PORT, log_level="info")
