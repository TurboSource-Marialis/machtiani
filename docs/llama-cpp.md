# llamma cpp server

Local model usage if you don't have a Mac or M-series Apple Silicon.

## Deploy llama server

Run Qwen-2.5-Coder

```
docker run --name alpine-qwen2-instruct -p 80:8080 \                                                ✔
-e LLAMA_API_KEY="password" \
-e LLAMA_ARG_MODEL_URL="https://huggingface.co/Qwen/Qwen2.5-Coder-1.5B-Instruct-GGUF/resolve/main/qwen2.5-coder-1.5b-instruct-q4_k_m.gguf" \
samueltallet/alpine-llama-cpp-server
```

Or download the model, then mount and run.

```
docker run --name qwen2-coder \
  -p 8080:8080 \
  -e LLAMA_API_KEY="password" \
  -e LLAMA_ARG_MODEL="/your/path/to/Qwen2.5-Coder-1.5B-Instruct-Q6_K_L.gguf" \
  --mount type=bind,source="$(pwd)/Qwen2.5-Coder-1.5B-Instruct-Q6_K_L.gguf",target="/your/path/to/Qwen2.5-Coder-1.5B-Instruct-Q6_K_L.gguf" \
  samueltallet/alpine-llama-cpp-server
```


## Set mct envars


The api key can be anything, such as "password", as long as it's not empty.

```
export MCT_MODEL_API_KEY=password
```

This is so mct knows how to reach the llama server on port 8080.

```
export MCT_MODEL_BASE_URL="http://host.docker.internal:8080/v1"
```

## Syncing

When syncing a git project, we only want a single thread as the llama server can't handle multiple requests.

And we don't amplify, because it will make it take 20 times longer than without:

```
mct sync --cost  --model Qwen2.5-Coder-1.5B-Instruct-Q6_K_L.gguf --model-threads 1
```

### Prompting

Test it out with no file retrieval (--mode pure-chat).

```
mct "Ask whatever you want here" --mode pure-chat --model Qwen2.5-Coder-1.5B-Instruct-GGUF
```

For real repo aware chat use `--mode chat`.

If you want code suggestions to be applied, please wrap your command using Codex as shown in README. The small local model could be very janky for mct otherwise. In any case, without any `--mode` flag it will attempt to apply suggestions.