# MLX-LM Installation & Quickstart


This guide walks you through installing and running **MLX-LM** (`mlx_lm.server`) on macOS using **Conda**, including workarounds for `protobuf` compatibility.

---

## 1. Prerequisites

- **macOS** (Apple Silicon)
- **Conda** (Miniconda or Anaconda)

If you haven’t installed Conda yet, see [Conda Installation](https://docs.conda.io/en/latest/miniconda.html).

---

## 2. Create & Activate a Conda Environment

```bash
# Update Conda
conda update -n base -c defaults conda

# Create an env named `mlx` with Python 3.9
conda create -n mlx python=3.9 -y

# Activate it
conda activate mlx
```

---

## 3. Install Dependencies

1. **Core build tools**
   ```bash
   conda install -c conda-forge cmake pkg-config -y
   ```
2. **SentencePiece**
   ```bash
   conda install -c conda-forge sentencepiece -y
   ```

---

## 4. Fix `protobuf` Compatibility

Transformer tokenizers ship with pre-generated protobuf code. Newer `protobuf` versions (>= 4.x) break this. You have two options:

- **Pin `protobuf`** to `<=3.20.1`
  ```bash
  conda install -c conda-forge protobuf=3.20.1 -y
  ```
- **Use the pure-Python Protobuf implementation**
  ```bash
  export PROTOCOL_BUFFERS_PYTHON_IMPLEMENTATION=python
  ```

You may include the `export` line in your shell profile (`~/.bashrc`, `~/.zshrc`) to persist it.

---

## 5. Install MLX-LM

With your Conda env active:

```bash
pip install --upgrade pip setuptools wheel
pip install mlx-lm
```

---

## 6. Launch the Server

By default, the server starts on `localhost:8080`:

**Qwen2.5-Coder-7B**

```bash
mlx_lm.server --model mlx-community/Qwen2.5-Coder-7B-Instruct-4bit --trust-remote-code
```

**Mistral-7B**

```bash
mlx_lm.server --model mlx-community/Mistral-7B-Instruct-v0.3-4bit --trust-remote-code
```

- `--model` sets the default model.
- `--trust-remote-code` allows loading custom model code.
- Use `--port <PORT>` to change the listening port.

---

### Troubleshooting

- **Missing `libmlx.so`**:
  Build from source:
  ```bash
  git clone https://github.com/ml-explore/mlx.git
  cd mlx
  git submodule update --init --recursive
  CMAKE_BUILD_PARALLEL_LEVEL=8 pip install .
  ```
- **`sentencepiece` wheel build errors**:
  Ensure `cmake` and `pkg-config` are installed (see Step 3).

---

## 8. Set mct envars

The api key can be anything, such as "password", as long as it's not empty.

```
export MCT_MODEL_API_KEY=password
```

This is so mct knows how to reach the llama server on port 8080.

```
export MCT_MODEL_BASE_URL="http://host.docker.internal:8080/v1"
```
---

9. ## Syncing

---

When syncing a git project, we only want a single thread as the llama server can't handle multiple requests.

And we don't amplify, because it will make it take 20 times longer than without:

```
mct sync --cost  --model mlx-community/Qwen2.5-Coder-7B-Instruct-4bit --model-threads 1
```

---

10. ## Prompting

Test it out with no file retrieval (--mode pure-chat).

```
mct "Ask whatever you want here" --mode pure-chat --model Qwen2.5-Coder-1.5B-Instruct-GGUF
```

For real repo aware chat use `--mode chat`.

If you want code suggestions to be applied, please wrap your command using Codex as shown in README. The small local model could be very janky for mct otherwise. In any case, without any `--mode` flag it will attempt to apply suggestions.


## Example Chat via `curl`, to test local llm in case issue with mct prompting.

You can override the model per request:

```bash
curl http://localhost:8080/v1/chat/completions   -H "Content-Type: application/json"   -d '{
    "model": "mlx-community/Mistral-7B-Instruct-v0.3-4bit",
    "messages": [
      {"role":"system","content":"You are a helpful assistant."},
      {"role":"user","content":"Say hello!"}
    ],
    "temperature": 0.6
  }'
```

**Expected Response** (JSON):

```json
{
  "id": "chatcmpl-...",
  "object": "chat.completion",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      }
    }
  ]
}
```

---
